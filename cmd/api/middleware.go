package main

import (
	"errors"
	"expvar"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/Bekian/greenlight/internal/data"
	"github.com/Bekian/greenlight/internal/validator"

	"github.com/tomasen/realip"
	"golang.org/x/time/rate"
)

func (app *application) recoverPanic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// deferred functions will be run in the event of a panic
		defer func() {
			// get panic value
			pv := recover()
			if pv != nil {
				// set a connection close header
				w.Header().Set("Connection", "close")

				// write the error response
				app.serverErrorResponse(w, r, fmt.Errorf("%v", pv))
			}
		}()

		// call next handler
		next.ServeHTTP(w, r)
	})
}

func (app *application) rateLimit(next http.Handler) http.Handler {
	// skip if disabled in config
	if !app.config.limiter.enabled {
		return next
	}
	// struct to hold ratelimiter and last seen time
	type client struct {
		limiter  *rate.Limiter
		lastSeen time.Time
	}
	// mutex and map to hold client IP and ratelimiters
	var (
		mu      sync.Mutex
		clients = make(map[string]*client)
	)

	// start a background reap-loop goroutine to purge clients every minute
	go func() {
		for {
			time.Sleep(time.Minute)

			// lock to prevent race condition during cleanup
			mu.Lock()

			// cleanup clients with a 3 minute outstanding timer
			for ip, client := range clients {
				if time.Since(client.lastSeen) > 3*time.Minute {
					delete(clients, ip)
				}
			}

			mu.Unlock()
		}
	}()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// ip from request
		ip := realip.FromRequest(r)

		mu.Lock()
		// if the ip address is not found then add it
		if _, found := clients[ip]; !found {
			// create limiter for user
			clients[ip] = &client{
				limiter: rate.NewLimiter(rate.Limit(app.config.limiter.rps), app.config.limiter.burst),
			}
		}

		// update last seen for the requesting client
		clients[ip].lastSeen = time.Now()

		// check if the limiter doesn't a request through
		if !clients[ip].limiter.Allow() {
			mu.Unlock()
			app.rateLimitExceededResponse(w, r)
			return
		}

		// ensure mutex is unlocked
		mu.Unlock()
		// call next handler
		next.ServeHTTP(w, r)
	})
}

func (app *application) authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// add "Vary: Authorization" header
		// any caches that see this will know that the response will vary depending
		// on the value of the authorization header
		w.Header().Add("Vary", "Authorization")

		// retrieve value of auth header from the req,
		// or receive "" if no header found
		authorizationHeader := r.Header.Get("Authorization")

		// if no auth is found, set the user to the anonymous user
		if authorizationHeader == "" {
			r = app.contextSetUser(r, data.AnonymousUser)
			next.ServeHTTP(w, r)
			return
		}

		// split the auth header into its parts,
		// if its not in the format we expect return 401
		headerParts := strings.Split(authorizationHeader, " ")
		if len(headerParts) != 2 || headerParts[0] != "Bearer" {
			app.invalidAuthenticationTokenResponse(w, r)
			return
		}

		// get the token from the header parts
		token := headerParts[1]

		// init a validator instance so we can validate the token
		v := validator.New()

		// if token isnt valid, send invalid authentication token response
		if data.ValidateTokenPlaintext(v, token); !v.Valid() {
			app.invalidAuthenticationTokenResponse(w, r)
			return
		}

		// get the user that matches the auth token,
		// otherwise use invalid auth token response
		user, err := app.models.Users.GetForToken(data.ScopeAuthentication, token)
		if err != nil {
			switch {
			case errors.Is(err, data.ErrRecordNotFound):
				app.invalidAuthenticationTokenResponse(w, r)
			default:
				app.serverErrorResponse(w, r, err)
			}
			return
		}

		// use contextSetUser helper to add user to context
		r = app.contextSetUser(r, user)

		// call next handler
		next.ServeHTTP(w, r)
	})
}

// middleware to check if a user isnt anonymous
func (app *application) requireAuthenticatedUser(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := app.contextGetUser(r)

		if user.IsAnonymous() {
			app.authenticationRequiredResponse(w, r)
			return
		}

		// call next handler
		next.ServeHTTP(w, r)
	})
}

// check if a user is both authenticated and activated
func (app *application) requireActivatedUser(next http.HandlerFunc) http.HandlerFunc {
	// we assign it to a variable so we can pass it later
	fn := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := app.contextGetUser(r)

		// check if the user is activated
		if !user.Activated {
			app.inactiveAccountResponse(w, r)
			return
		}

		next.ServeHTTP(w, r)
	})

	// use the middleware with the func
	return app.requireAuthenticatedUser(fn)
}

// first param is perm code that the user must have to use the endpoint
// DIFF Note: 16.4 is called "requirePermission"
func (app *application) requirePerm(code string, next http.HandlerFunc) http.HandlerFunc {
	fn := func(w http.ResponseWriter, r *http.Request) {
		// get user from context
		user := app.contextGetUser(r)

		// get perms from user
		// DIFF Note: perms is "permisisons"
		perms, err := app.models.Permissions.GetAllForUser(user.ID)
		if err != nil {
			app.serverErrorResponse(w, r, err)
			return
		}

		// check if slice has the required perm code
		if !perms.Include(code) {
			app.notPermittedResponse(w, r)
			return
		}

		// at this point the user has the correct perms
		next.ServeHTTP(w, r)
	}

	// wrap this with requireActivatedUser middleware
	return app.requireActivatedUser(fn)
}

// temporary middleware wrapper to allow all cors requests
func (app *application) enableCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Vary", "Origin")

		w.Header().Add("Vary", "Access-Control-Request-Method")

		// get req's origin header value
		origin := r.Header.Get("Origin")

		// only perform this block if theres a header
		if origin != "" {
			// loop through the config trusted origins
			// check if the header is a trusted origin
			for i := range app.config.cors.trustedOrigins {
				if origin == app.config.cors.trustedOrigins[i] {
					// set origin to allow the found origin
					w.Header().Set("Access-Control-Allow-Origin", origin)

					// check if the request is a preflight request by checking the following parameters
					if r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != "" {
						// set preflight headers
						w.Header().Set("Access-Control-Allow-Methods", "OPTIONS, PUT, PATCH, DELETE")
						w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
						// write 200 status
						w.WriteHeader(http.StatusOK)
						return
					}

					break
				}
			}
		}
		// call next handler
		next.ServeHTTP(w, r)
	})
}

// DIFF Note: the third variable here uses the greek letter mu here for microseconds,
// im too lazy to type that, so im using "us" instead
func (app *application) metrics(next http.Handler) http.Handler {
	// init variables when the middleware chain is created
	var (
		totalRequestsReceived           = expvar.NewInt("total_requests_received")
		totalResponsesSent              = expvar.NewInt("total_responses_sent")
		totalProcessingTimeMicroseconds = expvar.NewInt("total_processing_time_us")
	)

	// wrap the following around each request
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// get start time
		start := time.Now()
		// increment number of reqs received
		totalRequestsReceived.Add(1)

		// call next handler
		next.ServeHTTP(w, r)

		// increment number of responses sent
		totalResponsesSent.Add(1)
		// get duration of proccsesing time and add to total
		duration := time.Since(start).Microseconds()
		totalProcessingTimeMicroseconds.Add(duration)
	})
}

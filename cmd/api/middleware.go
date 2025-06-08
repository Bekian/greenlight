package main

import (
	"errors"
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

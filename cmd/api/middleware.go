package main

import (
	"fmt"
	"net/http"
	"sync"
	"time"

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

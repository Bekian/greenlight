package main

import (
	"fmt"
	"net/http"
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

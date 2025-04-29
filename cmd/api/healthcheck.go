package main

import (
	"fmt"
	"net/http"
)

// currently* a static function to write plain response and config values
func (app *application) healthcheckHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "status: available")
	fmt.Fprintf(w, "environment: %s\n", app.config.env)
	fmt.Fprintf(w, "version %s\n", version)
}

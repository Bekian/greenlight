package main

import (
	"net/http"
)

// currently* a static function to write plain response and config values
func (app *application) healthcheckHandler(w http.ResponseWriter, r *http.Request) {
	// map to hold data to send in response
	env := envelope{
		"status":      "available",
		"environment": app.config.env,
		"version":     version,
	}

	// marshal above map into json
	err := app.writeJSON(w, http.StatusOK, env, nil)
	if err != nil {
		app.logger.Error(err.Error())
		http.Error(w, "The server encountered a problem and could not process your request", http.StatusInternalServerError)
	}
}

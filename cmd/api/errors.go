package main

import (
	"fmt"
	"net/http"
)

// log helper to call a logger error from an http request
func (app *application) logError(r *http.Request, err error) {
	var (
		method = r.Method
		uri    = r.URL.RequestURI()
	)

	app.logger.Error(err.Error(), "method", method, "uri", uri)
}

// write a formatted json error response
// DIFF Note: is called "errorResponse"
func (app *application) errResponse(w http.ResponseWriter, r *http.Request, status int, message any) {
	env := envelope{"error": message}

	err := app.writeJSON(w, status, env, nil)
	if err != nil {
		app.logError(r, err)
		w.WriteHeader(500)
	}
}

// 400
func (app *application) badRequestResponse(w http.ResponseWriter, r *http.Request, err error) {
	app.errResponse(w, r, http.StatusBadRequest, err.Error())
}

// 404
func (app *application) notFoundResponse(w http.ResponseWriter, r *http.Request) {
	msg := "the requested resource could not be found"
	app.errResponse(w, r, http.StatusNotFound, msg)
}

// 405
func (app *application) methodNotAllowedResponse(w http.ResponseWriter, r *http.Request) {
	msg := fmt.Sprintf("the %s method is not supported for this resource", r.Method)
	app.errResponse(w, r, http.StatusMethodNotAllowed, msg)
}

// 409
func (app *application) editConflictResponse(w http.ResponseWriter, r *http.Request) {
	message := "unable to update the record due to an edit conflict, please try again"
	app.errResponse(w, r, http.StatusConflict, message)
}

// 429
func (app *application) rateLimitExceededResponse(w http.ResponseWriter, r *http.Request) {
	message := "rate limit exceeded, wait a few seconds before trying again"
	app.errResponse(w, r, http.StatusTooManyRequests, message)
}

// 500
func (app *application) serverErrorResponse(w http.ResponseWriter, r *http.Request, err error) {
	app.logError(r, err)

	msg := "the server encountered a problem and could not process your request"
	app.errResponse(w, r, http.StatusInternalServerError, msg)
}

// generate a response using errors from a validator
func (app *application) failedValidationResponse(w http.ResponseWriter, r *http.Request, errors map[string]string) {
	app.errResponse(w, r, http.StatusUnprocessableEntity, errors)
}

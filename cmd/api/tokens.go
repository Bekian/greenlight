package main

import (
	"errors"
	"net/http"
	"time"

	"github.com/Bekian/greenlight/internal/data"
	"github.com/Bekian/greenlight/internal/validator"
)

func (app *application) createAuthenticationTokenHandler(w http.ResponseWriter, r *http.Request) {
	// parse email ad pass from req body
	var input struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	err := app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	// validate input fields
	v := validator.New()

	data.ValidateEmail(v, input.Email)
	data.ValidatePasswordPlaintext(v, input.Password)

	if !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	// lookup user by email, otherwise send 401
	user, err := app.models.Users.GetByEmail(input.Email)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.invalidCredentialsResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	// ensure pass is correct
	match, err := user.Password.Matches(input.Password)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	// if passwords dont match call invalid creds helper
	if !match {
		app.invalidCredentialsResponse(w, r)
		return
	}

	// generate new token with 24h expiry and auth scope
	token, err := app.models.Tokens.New(user.ID, 24*time.Hour, data.ScopeAuthentication)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	// encode the token and wrap it into the response
	err = app.writeJSON(w, http.StatusCreated, envelope{"authentication_token": token}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

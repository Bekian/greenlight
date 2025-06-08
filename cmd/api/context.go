package main

import (
	"context"
	"net/http"

	"github.com/Bekian/greenlight/internal/data"
)

// a custom contextKey type, with the underlying type of string
// the key will be used to retrieve a related value from the context
type contextKey string

// convert string "user" to contextKey type, and assign it to the constant
// this is will be the key for getting and setting user info in the context
const userContextKey = contextKey("user")

// this method returns a copy of the request with the user struct attached to the request context
func (app *application) contextSetUser(r *http.Request, user *data.User) *http.Request {
	ctx := context.WithValue(r.Context(), userContextKey, user)
	return r.WithContext(ctx)
}

// contextGetUser retrieves the user struct from the request context
func (app *application) contextGetUser(r *http.Request) *data.User {
	user, ok := r.Context().Value(userContextKey).(*data.User)
	if !ok {
		panic("missing user value in request context")
	}

	return user
}

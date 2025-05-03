package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/julienschmidt/httprouter"
)

// read id param from request context and convert
func (app *application) readIDParam(r *http.Request) (int64, error) {
	params := httprouter.ParamsFromContext(r.Context())

	id, err := strconv.ParseInt(params.ByName("id"), 10, 64)
	if err != nil || id < 1 {
		return 0, errors.New("invalid id parameter")
	}

	return id, nil
}

// define an envelope type
type envelope map[string]any

// writejson writes provided data into a json response
// book diff: i renamed the js variable to json to avoid confusion
func (app *application) writeJSON(w http.ResponseWriter, status int, data envelope, headers http.Header) error {
	// empty string is an empty line prefix, tab prefixes each element.
	json, err := json.MarshalIndent(data, "", "\t")
	if err != nil {
		return err
	}

	// append newline for pretty UwU
	json = append(json, '\n')

	// add headers from header map
	for key, value := range headers {
		// i think this key value syntax on the method is a bit odd but w.e.
		w.Header()[key] = value
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write(json)
	return nil
}

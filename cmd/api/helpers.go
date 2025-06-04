package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/Bekian/greenlight/internal/validator"

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

func (app *application) readJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	// set max allowable request body size to 1mb
	r.Body = http.MaxBytesReader(w, r.Body, 1_048_576)
	// decode req body
	dec := json.NewDecoder(r.Body)
	// disallow unknown fields *before* decoding
	dec.DisallowUnknownFields()

	err := dec.Decode(dst)
	// triage any errors
	if err != nil {
		var syntaxError *json.SyntaxError
		var unmarshalTypeError *json.UnmarshalTypeError
		var invalidUnmarshalError *json.InvalidUnmarshalError
		var maxBytesError *http.MaxBytesError

		switch {
		// provide plaintext response of syntax error location
		case errors.As(err, &syntaxError):
			return fmt.Errorf("body contains mal-formed JSON (at character %d)", syntaxError.Offset)
		case errors.Is(err, io.ErrUnexpectedEOF):
			return errors.New("body contains badly-formed JSON")
		// when the field and it's value's type is mismatched
		case errors.As(err, &unmarshalTypeError):
			if unmarshalTypeError.Field != "" {
				return fmt.Errorf("body contains incorrect JSON type for field %q", unmarshalTypeError.Field)
			}
			return fmt.Errorf("body contains incorrect JSON type (at character %d)", unmarshalTypeError.Offset)
		// when the provided json body is empty
		case errors.Is(err, io.EOF):
			return errors.New("body must not be empty")
		// when the json contains a field that cannot be mapped to the dst
		case strings.HasPrefix(err.Error(), "json: unknown field "):
			fieldName := strings.TrimPrefix(err.Error(), "json: uknown field ")
			return fmt.Errorf("body contains unknown key %s", fieldName)
		// when the request body exceeds 1mb
		case errors.As(err, &maxBytesError):
			return fmt.Errorf("body must not be larger than %d bytes", maxBytesError.Limit)
		// panic when an unsupported value is passed incorrectly (dev error)
		case errors.As(err, &invalidUnmarshalError):
			panic(err)
		default:
			return err
		}
	}
	// call decode again to ensure there is only 1 json value
	err = dec.Decode(&struct{}{})
	// this will raise an eof error if theres multiple json values
	if !errors.Is(err, io.EOF) {
		return errors.New("body must only contain a single JSON value")
	}

	return nil
}

// returns a string value from a query string given a key string,
// otherwise it will return the provided default value
func (app *application) readString(qs url.Values, key string, defaultValue string) string {
	// extract the value for a given key from the query string.
	s := qs.Get(key)

	// if nothing is found an empty string is provided,
	// use that to return the default value.
	if s == "" {
		return defaultValue
	}

	return s
}

// finds and reads a string from the query string,
// and splits it on the comma character.
// if it is not found, it will return the provided default value
func (app *application) readCSV(qs url.Values, key string, defaultValue []string) []string {
	// extract the value for a given key from the query string.
	csv := qs.Get(key)

	// if nothing is found an empty string is provided,
	// use that to return the default value.
	if csv == "" {
		return defaultValue
	}

	// split and return the slice
	return strings.Split(csv, ",")
}

// attempt to find a string from the query value,
// then attempt to convert to an integer,
// if either fail, then return default value
func (app *application) readInt(qs url.Values, key string, defaultValue int, v *validator.Validator) int {
	// attempt to search for value
	s := qs.Get(key)

	// if no key exists or empty value, return default value.
	if s == "" {
		return defaultValue
	}

	// attempt to convert
	// otherwise add an error to the validator
	i, err := strconv.Atoi(s)
	if err != nil {
		v.AddError(key, "must be an integer value")
		return defaultValue
	}

	// sucess!
	return i
}

// background helper to run a function in the background
// and recover from a possible panic during function execution
func (app *application) background(fn func()) {
	// increment wg counter
	app.wg.Add(1)
	// start background func with panic recovery
	go func() {
		// defer wg decrement
		defer app.wg.Done()
		// recover panic
		defer func() {
			pv := recover()
			if pv != nil {
				app.logger.Error(fmt.Sprintf("%v", pv))
			}
		}()

		// execute function
		fn()
	}()
}

package data

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// custom error when we're unable to decode the value
var ErrInvalidRuntimeFormat = errors.New("invalid runtime format")

// type mirrors movie struct field type with the same name
type Runtime int32

// MarshalJSON method to satisfy the ENCODER interface
func (r Runtime) MarshalJSON() ([]byte, error) {
	jsonValue := fmt.Sprintf("%d mins", r)

	// add double quotes to make it JSON compatible
	quotedJSONValue := strconv.Quote(jsonValue)

	return []byte(quotedJSONValue), nil
}

// UNarshal method to satisfy the DECODER interface
func (r *Runtime) UnmarshalJSON(jsonValue []byte) error {
	// remove quotes from json value
	unquotedJSONValue, err := strconv.Unquote(string(jsonValue))
	if err != nil {
		return ErrInvalidRuntimeFormat
	}

	// split the remaining value into parts
	parts := strings.Split(unquotedJSONValue, " ")

	// ensure the correct amount of parts so the format is correct
	if len(parts) != 2 || parts[1] != "mins" {
		return ErrInvalidRuntimeFormat
	}

	// attempt to parse value into int32
	i, err := strconv.ParseInt(parts[0], 10, 32)
	if err != nil {
		return ErrInvalidRuntimeFormat
	}

	// convert to runtime type
	*r = Runtime(i)

	return nil
}

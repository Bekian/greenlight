package data

import (
	"fmt"
	"strconv"
)

// type mirrors movie struct field type with the same name
type Runtime int32

// MarshalJSON method to satisfy interface
func (r Runtime) MarshalJSON() ([]byte, error) {
	jsonValue := fmt.Sprintf("%d mins", r)

	// add double quotes to make it JSON compatible
	quotedJSONValue := strconv.Quote(jsonValue)

	return []byte(quotedJSONValue), nil
}

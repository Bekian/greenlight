package validator

import (
	"regexp"
	"slices"
)

// regexp to validate emails
var (
	EmailRX = regexp.MustCompile("^[a-zA-Z0-9.!#$%&'*+/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$")
)

// validator type to hold map of validation errors
type Validator struct {
	Errors map[string]string
}

// helper to create an instance of a Validator with an empty error map
func New() *Validator {
	return &Validator{Errors: make(map[string]string)}
}

// returns true if errors map does not contain any entries.
func (v *Validator) Valid() bool {
	return len(v.Errors) == 0
}

// adds an error message to the map,
// so long as a value with that key doesnt exist
func (v *Validator) AddError(key, message string) {
	if _, exists := v.Errors[key]; !exists {
		v.Errors[key] = message
	}
}

// adds an error message to the map only if the check is not 'ok'
// i.e. ok != true, then add message
func (v *Validator) Check(ok bool, key, message string) {
	if !ok {
		v.AddError(key, message)
	}
}

// returns true if a value is in a list of comparable values
func PermittedValue[T comparable](value T, permittedValues ...T) bool {
	return slices.Contains(permittedValues, value)
}

// returns true if a string value matches a specific regexp pattern
func Matches(value string, rx *regexp.Regexp) bool {
	return rx.MatchString(value)
}

// returns true if all values in a given slice are unique
func Unique[T comparable](values []T) bool {
	uniqueValues := make(map[T]bool)

	for _, value := range values {
		uniqueValues[value] = true
	}

	return len(values) == len(uniqueValues)
}

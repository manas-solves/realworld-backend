package validator

import (
	"regexp"
	"slices"
	"strings"
)

// EmailRX taken from https://html.spec.whatwg.org/#valid-e-mail-address.
var (
	EmailRX = regexp.MustCompile("^[a-zA-Z0-9.!#$%&'*+/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$")
)

// Validator contains a map of validation errors.
type Validator struct {
	Errors []string
}

// New is a helper which creates a new Validator instance with an empty errors list.
func New() *Validator {
	return &Validator{}
}

// Valid returns true if the errors list doesn't contain any entries.
func (v *Validator) Valid() bool {
	return len(v.Errors) == 0
}

// AddError adds an error message to the errors list
func (v *Validator) AddError(message string) {
	v.Errors = append(v.Errors, message)
}

// Check adds an error message to the list only if a validation check is not 'ok'.
func (v *Validator) Check(ok bool, message string) {
	if !ok {
		v.AddError(message)
	}
}

// PermittedValue returns true if a specific value is in a list of permitted values.
func PermittedValue[T comparable](value T, permittedValues ...T) bool {
	return slices.Contains(permittedValues, value)
}

// Matches returns true if a string value matches a specific regexp pattern.
func Matches(value string, rx *regexp.Regexp) bool {
	return rx.MatchString(value)
}

// Unique returns true if all values in a slice are unique.
func Unique[T comparable](values []T) bool {
	uniqueValues := make(map[T]bool)

	for _, value := range values {
		uniqueValues[value] = true
	}

	return len(values) == len(uniqueValues)
}

// NotEmptyOrWhitespace returns true if a string is empty or contains only whitespace characters.
func NotEmptyOrWhitespace(value string) bool {
	return strings.TrimSpace(value) != ""
}

package qac

import "fmt"

// AssertionResult is the container of verification results.
type AssertionResult struct {
	description string
	errors      []error
}

func (r *AssertionResult) addErrorf(format string, a ...interface{}) {
	err := fmt.Errorf(format, a...)
	r.addError(err)
}

func (r *AssertionResult) addError(err error) {
	r.errors = append(r.errors, err)
}
func (r *AssertionResult) addErrors(errors []error) {
	r.errors = append(r.errors, errors...)
}

// Description is the textual representation of the assertion.
func (r *AssertionResult) Description() string {
	return r.description
}

// Errors returns the errors list.
func (r *AssertionResult) Errors() []error {
	return r.errors
}

// Success returns if an assertion completed with no error.
func (r *AssertionResult) Success() bool {
	return len(r.errors) == 0
}

type assertion interface {
	verify(context planContext) AssertionResult
}

package qac

import (
	"errors"
	"fmt"
)

// AssertionResult is the container of verification results.
type AssertionResult struct {
	description string
	errors      []error
}

func (r *AssertionResult) addErrorf(format string, a ...interface{}) {
	r.errors = append(r.errors, &Error{
		Kind: KindAssertionFailure,
		msg:  fmt.Sprintf(format, a...),
	})
}

func (r *AssertionResult) addError(err error) {
	var qe *Error
	if !errors.As(err, &qe) {
		err = &Error{Kind: KindAssertionFailure, Cause: err, msg: err.Error()}
	}
	r.errors = append(r.errors, err)
}

func (r *AssertionResult) addErrors(errs []error) {
	for _, err := range errs {
		r.addError(err)
	}
}

func (r *AssertionResult) addInfraError(err error) {
	r.errors = append(r.errors, asInfraError(err))
}

func (r *AssertionResult) addConfigError(err error) {
	r.errors = append(r.errors, asConfigError(err))
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

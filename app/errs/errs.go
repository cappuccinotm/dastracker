// Package errs contains declarations of domain-level errors
// wrappers and methods to map them for client identification of the error.
package errs

import (
	"errors"
	"fmt"
)

// Standard errors.
var (
	ErrNotFound = errors.New("resource not found")
	ErrExists   = errors.New("resource already exists")
)

// ErrMethodParseFailed indicates that the Request contains
// an invalid path to the method.
type ErrMethodParseFailed string

// Error returns the string representation of the error.
func (e ErrMethodParseFailed) Error() string {
	return fmt.Sprintf("method path is invalid: %s", string(e))
}

// ErrTrackerNotRegistered indicates about the call to the tracker, that was
// not registered by the Dispatcher.
type ErrTrackerNotRegistered string

// Error returns the string representation of the error.
func (e ErrTrackerNotRegistered) Error() string {
	return fmt.Sprintf("tracker %q is not registered", string(e))
}

// ErrTrackerRegistered indicates that the attempt to register a new tracker
// handler has been failed as this tracker is already registered in the Manager.
type ErrTrackerRegistered string

// Error returns the string representation of the error.
func (e ErrTrackerRegistered) Error() string {
	return fmt.Sprintf("tracker %q is not registered", string(e))
}

// ErrActionNotSupported indicates that the action is not supported by the
// tracker.
type ErrActionNotSupported string

// Error returns the string representation of the error.
func (e ErrActionNotSupported) Error() string {
	return fmt.Sprintf("action %q is not supported", string(e))
}

// ErrGithubAPI describes any error responded by the Github API.
type ErrGithubAPI struct {
	ResponseStatus int    `json:"-"`
	Message        string `json:"message"`
	Errors         []struct {
		Code     string `json:"code"`
		Message  string `json:"message"`
		Resource string `json:"resource"`
	} `json:"errors"`
}

// Error returns the string representation of the error.
func (e ErrGithubAPI) Error() string {
	return fmt.Sprintf("github api responded error with status %d, message: %s, errors: %+v",
		e.ResponseStatus, e.Message, e.Errors)
}

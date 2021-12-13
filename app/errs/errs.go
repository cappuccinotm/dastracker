// Package errs contains declarations of domain-level errors
// wrappers and methods to map them for client identification of the error.
package errs

import (
	"errors"
	"fmt"
)

// Standard errors.
var (
	ErrNotFound  = errors.New("resource not found")
	ErrExists    = errors.New("resource already exists")
	ErrNoWebhook = errors.New("no webhook in the provided context")
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

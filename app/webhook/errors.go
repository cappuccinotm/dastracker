package webhook

import (
	"errors"
	"fmt"
)

// ErrNoWebhook indicates that the webhook in the provided context was not found.
var ErrNoWebhook = errors.New("no webhook in the provided context")

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

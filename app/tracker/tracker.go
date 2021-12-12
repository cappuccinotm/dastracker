package tracker

import (
	"context"
	"fmt"
	"github.com/cappuccinotm/dastracker/app/store"
	"strings"
)

//go:generate rm -f interface_mock.go
//go:generate moq -out interface_mock.go -fmt goimports . Interface

// Interface defines methods that each task tracker must implement.
type Interface interface {
	// Name returns the name of the tracker to match in the configuration.
	Name() string

	// Call makes a request to the tracker with specified method name,
	// variables and dastracker's TaskID. Response should contain the
	// TaskID of the ticket in the tracker.
	Call(ctx context.Context, req Request) (Response, error)

	// Subscribe makes a trigger with specified parameters and returns the
	// channel, to which updates will be published.
	Subscribe(ctx context.Context, req SubscribeReq) error

	// Listen runs the tracker's listener.
	// When the app is shutting down (ctx is canceled),
	// all trackers must unset all webhooks.
	Listen(ctx context.Context, h Handler) error
}

// Request describes a requests to tracker's action.
type Request struct {
	MethodURI string
	Ticket    store.Ticket
	Vars      store.Vars
}

// ParseMethodURI parses the MethodURI field of the request, assuming that
// the method is composed in form of "tracker/method". If the assumption does
// not hold, it returns empty strings instead.
func (r Request) ParseMethodURI() (tracker, method string, err error) {
	dividerIdx := strings.IndexRune(r.MethodURI, '/')
	if dividerIdx == -1 || dividerIdx == len(r.MethodURI)-1 || dividerIdx == 0 {
		return "", "", ErrMethodParseFailed(r.MethodURI)
	}

	return r.MethodURI[:dividerIdx], r.MethodURI[dividerIdx+1:], nil
}

// Response describes possible return values of the Interface.Call
type Response struct {
	Tracker string // tracker, from which the response was received
	TaskID  string // id of the created task in the tracker.
}

// SubscribeReq describes parameters of the subscription for task updates.
type SubscribeReq struct {
	TriggerName string
	Tracker     string
	Vars        store.Vars
}

// ErrMethodParseFailed indicates that the Request contains
// an invalid path to the method.
type ErrMethodParseFailed string

// Error returns the string representation of the error.
func (e ErrMethodParseFailed) Error() string {
	return fmt.Sprintf("method path is invalid: %s", string(e))
}

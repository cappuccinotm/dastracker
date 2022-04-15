package tracker

import (
	"context"

	"github.com/cappuccinotm/dastracker/app/store"
	"github.com/cappuccinotm/dastracker/lib"
	"net/http"
)

//go:generate rm -f interface_mock.go
//go:generate moq -out interface_mock.go -fmt goimports . Interface

// Interface defines methods that each task tracker must implement.
type Interface interface {
	// Name returns the name of the tracker to match in the configuration.
	Name() string

	// Call makes a request to the tracker with specified method name,
	// variables and dastracker's ID. Response should contain the
	// ID of the ticket in the tracker.
	Call(ctx context.Context, req Request) (Response, error)

	// Subscribe makes a trigger with specified parameters and returns the
	// channel, to which updates will be published.
	Subscribe(ctx context.Context, req SubscribeReq) (SubscribeResp, error)

	// Unsubscribe removes the trigger from the tracker.
	Unsubscribe(ctx context.Context, req UnsubscribeReq) error

	// HandleWebhook handles the update, received from the tracker.
	// It must parse the received request and call the provided to Listen
	// Handler.
	HandleWebhook(w http.ResponseWriter, r *http.Request)

	// Listen runs the tracker's listener.
	// When the app is shutting down (ctx is canceled),
	// all trackers must unset all webhooks.
	Listen(ctx context.Context, h Handler) error
}

// Request describes a requests to tracker's action.
type Request struct {
	// TaskID in the target tracker, might be empty if the request is for creation
	TaskID string
	Method string
	Vars   lib.Vars
}

// Response describes possible return values of the Interface.Call
type Response struct{ store.Task }

// SubscribeReq describes parameters of the subscription for task updates.
type SubscribeReq struct {
	Vars       lib.Vars
	WebhookURL string
}

// SubscribeResp describes response from tracker on subscription request.
type SubscribeResp struct {
	TrackerRef string
}

// UnsubscribeReq describes parameters for the unsubscription from task updates.
type UnsubscribeReq struct {
	TrackerRef string
}

// Handler handles the update, received from the Tracker.
type Handler interface {
	Handle(ctx context.Context, upd store.Update)
}

// HandlerFunc is an adapter to use ordinary functions as Handler.
type HandlerFunc func(context.Context, store.Update)

// Handle calls the wrapped function.
func (f HandlerFunc) Handle(ctx context.Context, upd store.Update) { f(ctx, upd) }

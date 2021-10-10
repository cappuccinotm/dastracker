package tracker

import (
	"context"
	"net/http"

	"github.com/cappuccinotm/dastracker/app/store"
	"github.com/google/uuid"
)

// Interface defines methods for each tracker.
// All computable values from Vars must be already evaluated, thus
// the finite values are provided.
type Interface interface {
	Call(ctx context.Context, call Request) (Response, error)
	SetUpTrigger(ctx context.Context, vars Vars, cb Callback) error
	Close(ctx context.Context) error
}

// Callback invokes when some action that trigger describes has been appeared.
type Callback interface {
	Do(ctx context.Context, update store.Update) error
}

// CallbackFunc is an adapter to use ordinary functions as Callbacks.
type CallbackFunc func(context.Context, store.Update) error

// Do invokes the wrapped method with provided arguments.
func (f CallbackFunc) Do(ctx context.Context, upd store.Update) error { return f(ctx, upd) }

// Request describes a requests to tracker's action.
type Request struct {
	Method   string
	Vars     Vars
	TicketID string // might be empty, in case if task is not registered yet
}

// Response describes possible return values of the Interface.Call
type Response struct {
	ID string // id of the created task in the tracker.
}

// WebhookProps describes parameters needed to tracker
// in order to instantiate a webhook.
type WebhookProps struct {
	Mux     *http.ServeMux
	BaseURL string
}

func (w *WebhookProps) newWebHook(fn func(w http.ResponseWriter, r *http.Request)) (url string) {
	url = w.BaseURL + "/" + uuid.NewString()
	w.Mux.HandleFunc(url, fn)
	return url
}

// Props describes basic properties for tracker.
type Props struct {
	Name      string
	Variables Vars
}

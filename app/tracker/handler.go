package tracker

import (
	"context"
	"github.com/cappuccinotm/dastracker/app/store"
)

// Handler handles the update, received from the Tracker.
type Handler interface {
	Handle(ctx context.Context, upd store.Update)
}

// HandlerFunc is an adapter to use ordinary functions as Handler.
type HandlerFunc func(context.Context, store.Update)

// Handle calls the wrapped function.
func (f HandlerFunc) Handle(ctx context.Context, upd store.Update) { f(ctx, upd) }

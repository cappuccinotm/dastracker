package service

import (
	"context"
	"fmt"
	"github.com/cappuccinotm/dastracker/app/store"
	"github.com/cappuccinotm/dastracker/app/tracker"
)

// Listen runs a Listener with specified tracker, handler and context.
// It will run unless the provided context is done.
func Listen(ctx context.Context, trk tracker.Interface, h Handler) error {
	return NewListener(trk, h).Listen(ctx)
}

// Listener is a listener for tracker updates.
type Listener struct {
	tracker tracker.Interface
	handler Handler
}

// NewListener makes new instances of Listener.
func NewListener(trk tracker.Interface, h Handler) *Listener {
	return &Listener{handler: h, tracker: trk}
}

// Listen runs the updates' listener.
// Always returns non-nil error. Blocking call.
func (l *Listener) Listen(baseCtx context.Context) error {
	updates := l.tracker.Updates()
	for {
		select {
		case <-baseCtx.Done():
			return fmt.Errorf("run stopped, reason: %w", baseCtx.Err())
		case upd := <-updates:
			go l.handler.Handle(baseCtx, upd)
		}
	}
}

// Handler handles the update, received from the Tracker.
type Handler interface {
	Handle(ctx context.Context, upd store.Update)
}

// HandlerFunc is an adapter to use ordinary functions as Handler.
type HandlerFunc func(context.Context, store.Update)

// Handle calls the wrapped function.
func (f HandlerFunc) Handle(ctx context.Context, upd store.Update) { f(ctx, upd) }

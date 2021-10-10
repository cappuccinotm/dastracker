package engine

import (
	"context"
	"errors"

	"github.com/cappuccinotm/dastracker/app/store"
)

// Interface describes methods each storage should implement.
type Interface interface {
	Create(ctx context.Context, ticket store.Ticket) (ticketID string, err error)
	Update(ctx context.Context, ticket store.Ticket) error
	Get(ctx context.Context, trackerName, trackerTaskID string) (store.Ticket, error)
}

var ErrNotFound = errors.New("not found")

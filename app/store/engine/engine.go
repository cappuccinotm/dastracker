package engine

import (
	"context"
	"errors"

	"github.com/cappuccinotm/dastracker/app/store"
)

//go:generate moq -out engine_mock.go -fmt goimports . Interface

// Interface describes methods each storage should implement.
type Interface interface {
	Create(ctx context.Context, ticket store.Ticket) (ticketID string, err error)
	Update(ctx context.Context, ticket store.Ticket) error
	Get(ctx context.Context, req GetRequest) (store.Ticket, error)
}

// GetRequest describes parameters to get a single ticket.
type GetRequest struct {
	Locator  store.Locator `json:"locator"`
	TicketID string        `json:"ticket_id"`
}

// ErrNotFound shows that the requested entity was not found in the store.
var ErrNotFound = errors.New("not found")

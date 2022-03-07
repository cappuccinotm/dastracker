package engine

import (
	"context"

	"github.com/cappuccinotm/dastracker/app/store"
)

//go:generate rm -f ticket_mock.go webhook_mock.go
//go:generate moq -out ticket_mock.go -fmt goimports . Tickets
//go:generate moq -out webhook_mock.go -fmt goimports . Webhooks

// Tickets describes methods each storage should implement.
type Tickets interface {
	Create(ctx context.Context, ticket store.Ticket) (ticketID string, err error)
	Update(ctx context.Context, ticket store.Ticket) error
	Get(ctx context.Context, req GetRequest) (store.Ticket, error)
}

// Webhooks defines methods to store and load information about webhooks.
type Webhooks interface {
	Create(ctx context.Context, wh store.Webhook) (whID string, err error)
	Get(ctx context.Context, whID string) (store.Webhook, error)
	Delete(ctx context.Context, whID string) error
	Update(ctx context.Context, wh store.Webhook) error

	// List returns all webhooks if trackerID = ""
	List(ctx context.Context, trackerID string) ([]store.Webhook, error)
}

// GetRequest describes parameters to get a single ticket.
type GetRequest struct {
	Locator  store.Locator `json:"locator"`
	TicketID string        `json:"ticket_id"`
}

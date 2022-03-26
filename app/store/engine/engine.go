package engine

import (
	"context"

	"github.com/cappuccinotm/dastracker/app/store"
)

//go:generate rm -f ticket_mock.go webhook_mock.go flow_mock.go
//go:generate moq -out ticket_mock.go -fmt goimports . Tickets
//go:generate moq -out subscriptions_mock.go -fmt goimports . Subscriptions
//go:generate moq -out flow_mock.go -fmt goimports . Flow

// Tickets describes methods each storage should implement.
type Tickets interface {
	Create(ctx context.Context, ticket store.Ticket) (ticketID string, err error)
	Update(ctx context.Context, ticket store.Ticket) error
	Get(ctx context.Context, req GetRequest) (store.Ticket, error)
}

// Subscriptions defines methods to store and load information about subscriptions.
type Subscriptions interface {
	Create(ctx context.Context, sub store.Subscription) (subID string, err error)
	Get(ctx context.Context, subID string) (store.Subscription, error)
	Delete(ctx context.Context, subID string) error
	Update(ctx context.Context, sub store.Subscription) error
	// List returns all subscriptions if trackerID = ""
	List(ctx context.Context, trackerID string) ([]store.Subscription, error)
}

// Flow defines methods to access the flow configuration.
type Flow interface {
	ListSubscribedJobs(ctx context.Context, triggerName string) ([]store.Job, error)
	ListTrackers(ctx context.Context) ([]store.Tracker, error)
	ListTriggers(ctx context.Context) ([]store.Trigger, error)
}

// GetRequest describes parameters to get a single ticket.
type GetRequest struct {
	Locator  store.Locator `json:"locator"`
	TicketID string        `json:"ticket_id"`
}

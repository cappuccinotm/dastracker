package engine

import (
	"context"

	"github.com/cappuccinotm/dastracker/app/store"
)

type Bolt struct{}

func (b *Bolt) Create(ctx context.Context, ticket store.Ticket) error {
	panic("implement me")
}

func (b *Bolt) Update(ctx context.Context, ticket store.Ticket) error {
	panic("implement me")
}

func (b *Bolt) Get(ctx context.Context, trackerName, trackerTaskID string) (store.Ticket, error) {
	panic("implement me")
}

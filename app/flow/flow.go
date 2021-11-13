package flow

import (
	"context"
	"errors"
	"github.com/cappuccinotm/dastracker/app/store"
)

//go:generate rm -f provider_mock.go
//go:generate moq -out interface_mock.go -fmt goimports . Interface

// Interface defines methods to access the flow configuration.
type Interface interface {
	GetSubscribedJobs(ctx context.Context, triggerName string) ([]store.Job, error)
}

// ErrNotFound indicates that the entity was not found
var ErrNotFound = errors.New("not found")

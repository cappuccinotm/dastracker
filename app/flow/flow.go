package flow

import (
	"context"

	"github.com/cappuccinotm/dastracker/app/store"
)

//go:generate rm -f interface_mock.go
//go:generate moq -out interface_mock.go -fmt goimports . Interface

// Interface defines methods to access the flow configuration.
type Interface interface {
	ListSubscribedJobs(ctx context.Context, triggerName string) ([]store.Job, error)
	ListTrackers(context.Context) ([]Tracker, error)
	ListTriggers(context.Context) ([]store.Trigger, error)
}

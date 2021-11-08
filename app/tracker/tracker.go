package tracker

import (
	"context"
	"github.com/cappuccinotm/dastracker/app/store"
)

//go:generate rm -f tracker_mock.go
//go:generate moq -out tracker_mock.go -fmt goimports . Interface

// Interface defines methods that each task tracker must implement.
type Interface interface {
	// Name returns the name of the tracker to match in the configuration.
	Name() string

	// Call makes a request to the tracker with specified method name,
	// variables and dastracker's TaskID. Response should contain the
	// TaskID of the ticket in the tracker.
	Call(ctx context.Context, req Request) (Response, error)

	// Subscribe makes a trigger with specified parameters and returns the
	// channel, to which updates will be published.
	Subscribe(ctx context.Context, vars store.Vars) (<-chan store.Update, error)

	// Close closes the connection to the tracker.
	Close(ctx context.Context) error
}

// Request describes a requests to tracker's action.
type Request struct {
	Method string
	Vars   store.Vars
	TaskID string // might be empty, in case if task is not registered yet
}

// Response describes possible return values of the Interface.Call
type Response struct {
	TaskID string // id of the created task in the tracker.
}

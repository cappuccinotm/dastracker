// Package provider provides methods and structures representing each element
// of the control flow tree and some common methods for them.
package provider

import (
	"context"
	"github.com/cappuccinotm/dastracker/app/store"
)

//go:generate rm -f provider_mock.go
//go:generate moq -out provider_mock.go -fmt goimports . Interface

// Interface defines methods for loading jobs and triggers.
type Interface interface {
	GetJob(ctx context.Context, name string) (store.Job, error)
}

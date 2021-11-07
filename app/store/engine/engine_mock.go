// Code generated by moq; DO NOT EDIT.
// github.com/matryer/moq

package engine

import (
	"context"
	"sync"

	"github.com/cappuccinotm/dastracker/app/store"
)

// Ensure, that InterfaceMock does implement Interface.
// If this is not the case, regenerate this file with moq.
var _ Interface = &InterfaceMock{}

// InterfaceMock is a mock implementation of Interface.
//
//     func TestSomethingThatUsesInterface(t *testing.T) {
//
//         // make and configure a mocked Interface
//         mockedInterface := &InterfaceMock{
//             CreateFunc: func(ctx context.Context, ticket store.Ticket) (string, error) {
// 	               panic("mock out the Create method")
//             },
//             GetFunc: func(ctx context.Context, req GetRequest) (store.Ticket, error) {
// 	               panic("mock out the Get method")
//             },
//             UpdateFunc: func(ctx context.Context, ticket store.Ticket) error {
// 	               panic("mock out the Update method")
//             },
//         }
//
//         // use mockedInterface in code that requires Interface
//         // and then make assertions.
//
//     }
type InterfaceMock struct {
	// CreateFunc mocks the Create method.
	CreateFunc func(ctx context.Context, ticket store.Ticket) (string, error)

	// GetFunc mocks the Get method.
	GetFunc func(ctx context.Context, req GetRequest) (store.Ticket, error)

	// UpdateFunc mocks the Update method.
	UpdateFunc func(ctx context.Context, ticket store.Ticket) error

	// calls tracks calls to the methods.
	calls struct {
		// Create holds details about calls to the Create method.
		Create []struct {
			// Ctx is the ctx argument value.
			Ctx context.Context
			// Ticket is the ticket argument value.
			Ticket store.Ticket
		}
		// Get holds details about calls to the Get method.
		Get []struct {
			// Ctx is the ctx argument value.
			Ctx context.Context
			// Req is the req argument value.
			Req GetRequest
		}
		// Update holds details about calls to the Update method.
		Update []struct {
			// Ctx is the ctx argument value.
			Ctx context.Context
			// Ticket is the ticket argument value.
			Ticket store.Ticket
		}
	}
	lockCreate sync.RWMutex
	lockGet    sync.RWMutex
	lockUpdate sync.RWMutex
}

// Create calls CreateFunc.
func (mock *InterfaceMock) Create(ctx context.Context, ticket store.Ticket) (string, error) {
	if mock.CreateFunc == nil {
		panic("InterfaceMock.CreateFunc: method is nil but Interface.Create was just called")
	}
	callInfo := struct {
		Ctx    context.Context
		Ticket store.Ticket
	}{
		Ctx:    ctx,
		Ticket: ticket,
	}
	mock.lockCreate.Lock()
	mock.calls.Create = append(mock.calls.Create, callInfo)
	mock.lockCreate.Unlock()
	return mock.CreateFunc(ctx, ticket)
}

// CreateCalls gets all the calls that were made to Create.
// Check the length with:
//     len(mockedInterface.CreateCalls())
func (mock *InterfaceMock) CreateCalls() []struct {
	Ctx    context.Context
	Ticket store.Ticket
} {
	var calls []struct {
		Ctx    context.Context
		Ticket store.Ticket
	}
	mock.lockCreate.RLock()
	calls = mock.calls.Create
	mock.lockCreate.RUnlock()
	return calls
}

// Get calls GetFunc.
func (mock *InterfaceMock) Get(ctx context.Context, req GetRequest) (store.Ticket, error) {
	if mock.GetFunc == nil {
		panic("InterfaceMock.GetFunc: method is nil but Interface.Get was just called")
	}
	callInfo := struct {
		Ctx context.Context
		Req GetRequest
	}{
		Ctx: ctx,
		Req: req,
	}
	mock.lockGet.Lock()
	mock.calls.Get = append(mock.calls.Get, callInfo)
	mock.lockGet.Unlock()
	return mock.GetFunc(ctx, req)
}

// GetCalls gets all the calls that were made to Get.
// Check the length with:
//     len(mockedInterface.GetCalls())
func (mock *InterfaceMock) GetCalls() []struct {
	Ctx context.Context
	Req GetRequest
} {
	var calls []struct {
		Ctx context.Context
		Req GetRequest
	}
	mock.lockGet.RLock()
	calls = mock.calls.Get
	mock.lockGet.RUnlock()
	return calls
}

// Update calls UpdateFunc.
func (mock *InterfaceMock) Update(ctx context.Context, ticket store.Ticket) error {
	if mock.UpdateFunc == nil {
		panic("InterfaceMock.UpdateFunc: method is nil but Interface.Update was just called")
	}
	callInfo := struct {
		Ctx    context.Context
		Ticket store.Ticket
	}{
		Ctx:    ctx,
		Ticket: ticket,
	}
	mock.lockUpdate.Lock()
	mock.calls.Update = append(mock.calls.Update, callInfo)
	mock.lockUpdate.Unlock()
	return mock.UpdateFunc(ctx, ticket)
}

// UpdateCalls gets all the calls that were made to Update.
// Check the length with:
//     len(mockedInterface.UpdateCalls())
func (mock *InterfaceMock) UpdateCalls() []struct {
	Ctx    context.Context
	Ticket store.Ticket
} {
	var calls []struct {
		Ctx    context.Context
		Ticket store.Ticket
	}
	mock.lockUpdate.RLock()
	calls = mock.calls.Update
	mock.lockUpdate.RUnlock()
	return calls
}

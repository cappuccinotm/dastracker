// Code generated by moq; DO NOT EDIT.
// github.com/matryer/moq

package engine

import (
	"context"
	"sync"

	"github.com/cappuccinotm/dastracker/app/store"
)

// Ensure, that TicketsMock does implement Tickets.
// If this is not the case, regenerate this file with moq.
var _ Tickets = &TicketsMock{}

// TicketsMock is a mock implementation of Tickets.
//
// 	func TestSomethingThatUsesTickets(t *testing.T) {
//
// 		// make and configure a mocked Tickets
// 		mockedTickets := &TicketsMock{
// 			CreateFunc: func(ctx context.Context, ticket store.Ticket) (string, error) {
// 				panic("mock out the Create method")
// 			},
// 			GetFunc: func(ctx context.Context, req GetRequest) (store.Ticket, error) {
// 				panic("mock out the Get method")
// 			},
// 			UpdateFunc: func(ctx context.Context, ticket store.Ticket) error {
// 				panic("mock out the Update method")
// 			},
// 		}
//
// 		// use mockedTickets in code that requires Tickets
// 		// and then make assertions.
//
// 	}
type TicketsMock struct {
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
func (mock *TicketsMock) Create(ctx context.Context, ticket store.Ticket) (string, error) {
	if mock.CreateFunc == nil {
		panic("TicketsMock.CreateFunc: method is nil but Tickets.Create was just called")
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
//     len(mockedTickets.CreateCalls())
func (mock *TicketsMock) CreateCalls() []struct {
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
func (mock *TicketsMock) Get(ctx context.Context, req GetRequest) (store.Ticket, error) {
	if mock.GetFunc == nil {
		panic("TicketsMock.GetFunc: method is nil but Tickets.Get was just called")
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
//     len(mockedTickets.GetCalls())
func (mock *TicketsMock) GetCalls() []struct {
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
func (mock *TicketsMock) Update(ctx context.Context, ticket store.Ticket) error {
	if mock.UpdateFunc == nil {
		panic("TicketsMock.UpdateFunc: method is nil but Tickets.Update was just called")
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
//     len(mockedTickets.UpdateCalls())
func (mock *TicketsMock) UpdateCalls() []struct {
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

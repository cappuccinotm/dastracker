// Code generated by moq; DO NOT EDIT.
// github.com/matryer/moq

package tracker

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
//             CallFunc: func(ctx context.Context, req Request) (Response, error) {
// 	               panic("mock out the Call method")
//             },
//             NameFunc: func() string {
// 	               panic("mock out the Name method")
//             },
//             RunFunc: func(ctx context.Context) error {
// 	               panic("mock out the Listen method")
//             },
//             SubscribeFunc: func(ctx context.Context, req SubscribeReq) error {
// 	               panic("mock out the Subscribe method")
//             },
//             UpdatesFunc: func() <-chan store.Update {
// 	               panic("mock out the Updates method")
//             },
//         }
//
//         // use mockedInterface in code that requires Interface
//         // and then make assertions.
//
//     }
type InterfaceMock struct {
	// CallFunc mocks the Call method.
	CallFunc func(ctx context.Context, req Request) (Response, error)

	// NameFunc mocks the Name method.
	NameFunc func() string

	// RunFunc mocks the Listen method.
	RunFunc func(ctx context.Context) error

	// SubscribeFunc mocks the Subscribe method.
	SubscribeFunc func(ctx context.Context, req SubscribeReq) error

	// UpdatesFunc mocks the Updates method.
	UpdatesFunc func() <-chan store.Update

	// calls tracks calls to the methods.
	calls struct {
		// Call holds details about calls to the Call method.
		Call []struct {
			// Ctx is the ctx argument value.
			Ctx context.Context
			// Req is the req argument value.
			Req Request
		}
		// Name holds details about calls to the Name method.
		Name []struct {
		}
		// Listen holds details about calls to the Listen method.
		Run []struct {
			// Ctx is the ctx argument value.
			Ctx context.Context
		}
		// Subscribe holds details about calls to the Subscribe method.
		Subscribe []struct {
			// Ctx is the ctx argument value.
			Ctx context.Context
			// Req is the req argument value.
			Req SubscribeReq
		}
		// Updates holds details about calls to the Updates method.
		Updates []struct {
		}
	}
	lockCall      sync.RWMutex
	lockName      sync.RWMutex
	lockRun       sync.RWMutex
	lockSubscribe sync.RWMutex
	lockUpdates   sync.RWMutex
}

// Call calls CallFunc.
func (mock *InterfaceMock) Call(ctx context.Context, req Request) (Response, error) {
	if mock.CallFunc == nil {
		panic("InterfaceMock.CallFunc: method is nil but Interface.Call was just called")
	}
	callInfo := struct {
		Ctx context.Context
		Req Request
	}{
		Ctx: ctx,
		Req: req,
	}
	mock.lockCall.Lock()
	mock.calls.Call = append(mock.calls.Call, callInfo)
	mock.lockCall.Unlock()
	return mock.CallFunc(ctx, req)
}

// CallCalls gets all the calls that were made to Call.
// Check the length with:
//     len(mockedInterface.CallCalls())
func (mock *InterfaceMock) CallCalls() []struct {
	Ctx context.Context
	Req Request
} {
	var calls []struct {
		Ctx context.Context
		Req Request
	}
	mock.lockCall.RLock()
	calls = mock.calls.Call
	mock.lockCall.RUnlock()
	return calls
}

// Name calls NameFunc.
func (mock *InterfaceMock) Name() string {
	if mock.NameFunc == nil {
		panic("InterfaceMock.NameFunc: method is nil but Interface.Name was just called")
	}
	callInfo := struct {
	}{}
	mock.lockName.Lock()
	mock.calls.Name = append(mock.calls.Name, callInfo)
	mock.lockName.Unlock()
	return mock.NameFunc()
}

// NameCalls gets all the calls that were made to Name.
// Check the length with:
//     len(mockedInterface.NameCalls())
func (mock *InterfaceMock) NameCalls() []struct {
} {
	var calls []struct {
	}
	mock.lockName.RLock()
	calls = mock.calls.Name
	mock.lockName.RUnlock()
	return calls
}

// Listen calls RunFunc.
func (mock *InterfaceMock) Listen(ctx context.Context) error {
	if mock.RunFunc == nil {
		panic("InterfaceMock.RunFunc: method is nil but Interface.Listen was just called")
	}
	callInfo := struct {
		Ctx context.Context
	}{
		Ctx: ctx,
	}
	mock.lockRun.Lock()
	mock.calls.Run = append(mock.calls.Run, callInfo)
	mock.lockRun.Unlock()
	return mock.RunFunc(ctx)
}

// RunCalls gets all the calls that were made to Listen.
// Check the length with:
//     len(mockedInterface.RunCalls())
func (mock *InterfaceMock) RunCalls() []struct {
	Ctx context.Context
} {
	var calls []struct {
		Ctx context.Context
	}
	mock.lockRun.RLock()
	calls = mock.calls.Run
	mock.lockRun.RUnlock()
	return calls
}

// Subscribe calls SubscribeFunc.
func (mock *InterfaceMock) Subscribe(ctx context.Context, req SubscribeReq) error {
	if mock.SubscribeFunc == nil {
		panic("InterfaceMock.SubscribeFunc: method is nil but Interface.Subscribe was just called")
	}
	callInfo := struct {
		Ctx context.Context
		Req SubscribeReq
	}{
		Ctx: ctx,
		Req: req,
	}
	mock.lockSubscribe.Lock()
	mock.calls.Subscribe = append(mock.calls.Subscribe, callInfo)
	mock.lockSubscribe.Unlock()
	return mock.SubscribeFunc(ctx, req)
}

// SubscribeCalls gets all the calls that were made to Subscribe.
// Check the length with:
//     len(mockedInterface.SubscribeCalls())
func (mock *InterfaceMock) SubscribeCalls() []struct {
	Ctx context.Context
	Req SubscribeReq
} {
	var calls []struct {
		Ctx context.Context
		Req SubscribeReq
	}
	mock.lockSubscribe.RLock()
	calls = mock.calls.Subscribe
	mock.lockSubscribe.RUnlock()
	return calls
}

// Updates calls UpdatesFunc.
func (mock *InterfaceMock) Updates() <-chan store.Update {
	if mock.UpdatesFunc == nil {
		panic("InterfaceMock.UpdatesFunc: method is nil but Interface.Updates was just called")
	}
	callInfo := struct {
	}{}
	mock.lockUpdates.Lock()
	mock.calls.Updates = append(mock.calls.Updates, callInfo)
	mock.lockUpdates.Unlock()
	return mock.UpdatesFunc()
}

// UpdatesCalls gets all the calls that were made to Updates.
// Check the length with:
//     len(mockedInterface.UpdatesCalls())
func (mock *InterfaceMock) UpdatesCalls() []struct {
} {
	var calls []struct {
	}
	mock.lockUpdates.RLock()
	calls = mock.calls.Updates
	mock.lockUpdates.RUnlock()
	return calls
}

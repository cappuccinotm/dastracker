package tracker

import (
	"context"
	"github.com/cappuccinotm/dastracker/app/store"
	"github.com/stretchr/testify/assert"
	"log"
	"reflect"
	"runtime"
	"testing"
	"time"
)

func TestDispatcher_Call(t *testing.T) {
	trk1, trk2, svc := prepareDispatcher(t)

	t.Run("valid calls", func(t *testing.T) {
		trk1.CallFunc = func(ctx context.Context, req Request) (Response, error) {
			assert.Equal(t, context.TODO(), ctx)
			assert.Equal(t, Request{
				Method: "trk1/method", Ticket: store.Ticket{ID: "ticket-id"},
			}, req)
			return Response{Tracker: "trk1", TaskID: "task-id"}, nil
		}
		resp, err := svc.Call(context.TODO(), Request{
			Method: "trk1/method", Ticket: store.Ticket{ID: "ticket-id"},
		})
		assert.NoError(t, err)
		assert.Equal(t, Response{Tracker: "trk1", TaskID: "task-id"}, resp)

		trk2.CallFunc = func(ctx context.Context, req Request) (Response, error) {
			assert.Equal(t, context.TODO(), ctx)
			assert.Equal(t, Request{
				Method: "trk2/method", Ticket: store.Ticket{ID: "ticket-id"},
			}, req)
			return Response{Tracker: "trk2", TaskID: "task-id-2"}, nil
		}

		resp, err = svc.Call(context.TODO(), Request{
			Method: "trk2/method", Ticket: store.Ticket{ID: "ticket-id"},
		})
		assert.NoError(t, err)
		assert.Equal(t, Response{Tracker: "trk2", TaskID: "task-id-2"}, resp)
	})

	t.Run("tracker not registered", func(t *testing.T) {
		resp, err := svc.Call(context.TODO(), Request{
			Method: "trk3/method", Ticket: store.Ticket{ID: "ticket-id"},
		})
		assert.Empty(t, resp)
		var errTrackerNotRegistered ErrTrackerNotRegistered
		assert.ErrorAs(t, err, &errTrackerNotRegistered)
		assert.Equal(t, "trk3", string(errTrackerNotRegistered))
	})

	t.Run("invalid tracker method", func(t *testing.T) {
		resp, err := svc.Call(context.TODO(), Request{
			Method: "method", Ticket: store.Ticket{ID: "ticket-id"},
		})
		assert.Empty(t, resp)
		var errMethodParse ErrMethodParseFailed
		assert.ErrorAs(t, err, &errMethodParse)
		assert.Equal(t, "method", string(errMethodParse))
	})

	t.Run("empty method name", func(t *testing.T) {
		resp, err := svc.Call(context.TODO(), Request{
			Method: "", Ticket: store.Ticket{ID: "ticket-id"},
		})
		assert.Empty(t, resp)
		var errMethodParse ErrMethodParseFailed
		assert.ErrorAs(t, err, &errMethodParse)
		assert.Empty(t, string(errMethodParse))
	})
}

func TestDispatcher_Name(t *testing.T) {
	_, _, svc := prepareDispatcher(t)
	// due to the randomization of the map we cannot (and must not) rely on the
	// order of trackers in it, so we have to consider multiple variants of the name
	possibleExpectedNames := []string{
		"Dispatcher[trk1, trk2]",
		"Dispatcher[trk2, trk1]",
	}
	assert.Contains(t, possibleExpectedNames, svc.Name(), "name is unexpected")
}

func TestDispatcher_Subscribe(t *testing.T) {
	trk1, trk2, svc := prepareDispatcher(t)

	t.Run("valid calls", func(t *testing.T) {
		trk1.SubscribeFunc = func(ctx context.Context, req SubscribeReq) error {
			assert.Equal(t, context.TODO(), ctx)
			assert.Equal(t, SubscribeReq{TriggerName: "trigger-name", Tracker: "trk1"}, req)
			return nil
		}
		err := svc.Subscribe(context.TODO(), SubscribeReq{
			TriggerName: "trigger-name",
			Tracker:     "trk1",
		})
		assert.NoError(t, err)

		trk2.SubscribeFunc = func(ctx context.Context, req SubscribeReq) error {
			assert.Equal(t, context.TODO(), ctx)
			assert.Equal(t, SubscribeReq{TriggerName: "trigger-name-2", Tracker: "trk2"}, req)
			return nil
		}
		err = svc.Subscribe(context.TODO(), SubscribeReq{
			TriggerName: "trigger-name-2",
			Tracker:     "trk2",
		})
		assert.NoError(t, err)
	})

	t.Run("tracker not registered", func(t *testing.T) {
		err := svc.Subscribe(context.TODO(), SubscribeReq{
			TriggerName: "some-trigger",
			Tracker:     "trk3",
		})
		var errTrackerNotRegistered ErrTrackerNotRegistered
		assert.ErrorAs(t, err, &errTrackerNotRegistered)
		assert.Equal(t, "trk3", string(errTrackerNotRegistered))
	})

	t.Run("tracker name empty", func(t *testing.T) {
		err := svc.Subscribe(context.TODO(), SubscribeReq{
			TriggerName: "some-trigger",
			Tracker:     "",
		})
		var errTrackerNotRegistered ErrTrackerNotRegistered
		assert.ErrorAs(t, err, &errTrackerNotRegistered)
		assert.Equal(t, "", string(errTrackerNotRegistered))
	})
}

func TestDispatcher_Listen(t *testing.T) {
	getFuncName := func(i interface{}) string {
		return runtime.FuncForPC(reflect.ValueOf(i).Pointer()).Name()
	}

	t.Run("correct call", func(t *testing.T) {
		handler := HandlerFunc(func(context.Context, store.Update) {})
		expectedHandler := getFuncName(handler)

		baseCtx, stop := context.WithCancel(context.Background())

		trk1, trk2, svc := prepareDispatcher(t)
		trk1.ListenFunc = func(ctx context.Context, h Handler) error {
			assert.Equal(t, expectedHandler, getFuncName(h))
			<-ctx.Done()
			return ctx.Err()
		}
		trk2.ListenFunc = trk1.ListenFunc

		run := signaler()
		closed := signaler()

		var closeErr error

		go func() {
			run.done()
			closeErr = svc.Listen(baseCtx, handler)
			closed.done()
		}()

		waitTimeout(t, run, time.Second, "run")
		stop()
		waitTimeout(t, closed, time.Second, "stop")

		assert.Len(t, trk1.ListenCalls(), 1)
		assert.Len(t, trk2.ListenCalls(), 1)
		assert.ErrorIs(t, closeErr, context.Canceled)
	})
}

func waitTimeout(t *testing.T, done <-chan struct{}, timeout time.Duration, msgs ...interface{}) {
	t.Helper()

	tm := time.NewTimer(timeout)
	for {
		select {
		case <-tm.C:
			assert.FailNow(t, "timed out", msgs...)
		case <-done:
			tm.Stop()
			return
		}
	}
}

type trkMock struct {
	*InterfaceMock
}

func prepareDispatcher(t *testing.T) (
	trk1 *trkMock,
	trk2 *trkMock,
	svc *Dispatcher,
) {
	t.Helper()
	trk1 = &trkMock{InterfaceMock: &InterfaceMock{NameFunc: func() string { return "trk1" }}}
	trk2 = &trkMock{InterfaceMock: &InterfaceMock{NameFunc: func() string { return "trk2" }}}
	svc, err := NewDispatcher(log.Default(), []Interface{trk1, trk2})
	assert.NoError(t, err)
	return trk1, trk2, svc
}

func TestErrTrackerNotRegistered_Error(t *testing.T) {
	assert.Equal(t, `tracker "blah" is not registered`, ErrTrackerNotRegistered("blah").Error())
}

type sign chan struct{}

func (d sign) done() {
	select {
	case <-d:
		return
	default:
		close(d)
	}
}

func signaler() sign { return make(chan struct{}) }

package service

import (
	"context"
	"github.com/cappuccinotm/dastracker/app/store"
	"github.com/cappuccinotm/dastracker/app/tracker"
	"github.com/stretchr/testify/assert"
	"log"
	"testing"
	"time"
)

func TestDispatcher_Call(t *testing.T) {
	trk1, trk2, svc := prepareDispatcher(t)

	t.Run("valid calls", func(t *testing.T) {
		trk1.CallFunc = func(ctx context.Context, req tracker.Request) (tracker.Response, error) {
			assert.Equal(t, context.TODO(), ctx)
			assert.Equal(t, tracker.Request{
				Method: "trk1/method", Ticket: store.Ticket{ID: "ticket-id"},
			}, req)
			return tracker.Response{Tracker: "trk1", TaskID: "task-id"}, nil
		}
		resp, err := svc.Call(context.TODO(), tracker.Request{
			Method: "trk1/method", Ticket: store.Ticket{ID: "ticket-id"},
		})
		assert.NoError(t, err)
		assert.Equal(t, tracker.Response{Tracker: "trk1", TaskID: "task-id"}, resp)

		trk2.CallFunc = func(ctx context.Context, req tracker.Request) (tracker.Response, error) {
			assert.Equal(t, context.TODO(), ctx)
			assert.Equal(t, tracker.Request{
				Method: "trk2/method", Ticket: store.Ticket{ID: "ticket-id"},
			}, req)
			return tracker.Response{Tracker: "trk2", TaskID: "task-id-2"}, nil
		}

		resp, err = svc.Call(context.TODO(), tracker.Request{
			Method: "trk2/method", Ticket: store.Ticket{ID: "ticket-id"},
		})
		assert.NoError(t, err)
		assert.Equal(t, tracker.Response{Tracker: "trk2", TaskID: "task-id-2"}, resp)
	})

	t.Run("tracker not registered", func(t *testing.T) {
		resp, err := svc.Call(context.TODO(), tracker.Request{
			Method: "trk3/method", Ticket: store.Ticket{ID: "ticket-id"},
		})
		assert.Empty(t, resp)
		var errTrackerNotRegistered ErrTrackerNotRegistered
		assert.ErrorAs(t, err, &errTrackerNotRegistered)
		assert.Equal(t, "trk3", string(errTrackerNotRegistered))
	})

	t.Run("invalid tracker method", func(t *testing.T) {
		resp, err := svc.Call(context.TODO(), tracker.Request{
			Method: "method", Ticket: store.Ticket{ID: "ticket-id"},
		})
		assert.Empty(t, resp)
		var errMethodParse tracker.ErrMethodParseFailed
		assert.ErrorAs(t, err, &errMethodParse)
		assert.Equal(t, "method", string(errMethodParse))
	})

	t.Run("empty method name", func(t *testing.T) {
		resp, err := svc.Call(context.TODO(), tracker.Request{
			Method: "", Ticket: store.Ticket{ID: "ticket-id"},
		})
		assert.Empty(t, resp)
		var errMethodParse tracker.ErrMethodParseFailed
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
		trk1.SubscribeFunc = func(ctx context.Context, req tracker.SubscribeReq) error {
			assert.Equal(t, context.TODO(), ctx)
			assert.Equal(t, tracker.SubscribeReq{TriggerName: "trigger-name", Tracker: "trk1"}, req)
			return nil
		}
		err := svc.Subscribe(context.TODO(), tracker.SubscribeReq{
			TriggerName: "trigger-name",
			Tracker:     "trk1",
		})
		assert.NoError(t, err)

		trk2.SubscribeFunc = func(ctx context.Context, req tracker.SubscribeReq) error {
			assert.Equal(t, context.TODO(), ctx)
			assert.Equal(t, tracker.SubscribeReq{TriggerName: "trigger-name-2", Tracker: "trk2"}, req)
			return nil
		}
		err = svc.Subscribe(context.TODO(), tracker.SubscribeReq{
			TriggerName: "trigger-name-2",
			Tracker:     "trk2",
		})
		assert.NoError(t, err)
	})

	t.Run("tracker not registered", func(t *testing.T) {
		err := svc.Subscribe(context.TODO(), tracker.SubscribeReq{
			TriggerName: "some-trigger",
			Tracker:     "trk3",
		})
		var errTrackerNotRegistered ErrTrackerNotRegistered
		assert.ErrorAs(t, err, &errTrackerNotRegistered)
		assert.Equal(t, "trk3", string(errTrackerNotRegistered))
	})

	t.Run("tracker name empty", func(t *testing.T) {
		err := svc.Subscribe(context.TODO(), tracker.SubscribeReq{
			TriggerName: "some-trigger",
			Tracker:     "",
		})
		var errTrackerNotRegistered ErrTrackerNotRegistered
		assert.ErrorAs(t, err, &errTrackerNotRegistered)
		assert.Equal(t, "", string(errTrackerNotRegistered))
	})
}

func TestDispatcher_Run(t *testing.T) {
	t.Run("without lost", func(t *testing.T) {
		trk1, trk2, svc := prepareDispatcher(t)

		chn1 := make(chan store.Update)
		chn2 := make(chan store.Update)

		trk1.UpdatesFunc = func() <-chan store.Update { return chn1 }
		trk2.UpdatesFunc = func() <-chan store.Update { return chn2 }

		ctx, stop := context.WithCancel(context.Background())
		closed := signaler()
		go func() {
			err := svc.Listen(ctx)
			assert.ErrorIs(t, err, context.Canceled)
			closed.done()
		}()

		chn1 <- store.Update{URL: "ticket-1"}
		assert.Equal(t, store.Update{URL: "ticket-1"}, <-svc.Updates())
		chn2 <- store.Update{URL: "ticket-2"}
		assert.Equal(t, store.Update{URL: "ticket-2"}, <-svc.Updates())

		stop()
		waitTimeout(t, closed, time.Second)
	})

	t.Run("with lost", func(t *testing.T) {
		trk1, trk2, svc := prepareDispatcher(t, LostTimeout(time.Millisecond))
		// replacing with a channel with buffer 1
		close(svc.chn)
		svc.chn = make(chan store.Update, 1)

		chn1 := make(chan store.Update)
		chn2 := make(chan store.Update)

		trk1.UpdatesFunc = func() <-chan store.Update { return chn1 }
		trk2.UpdatesFunc = func() <-chan store.Update { return chn2 }

		ctx, stop := context.WithCancel(context.Background())
		closed := signaler()
		go func() {
			err := svc.Listen(ctx)
			assert.ErrorIs(t, err, context.Canceled)
			closed.done()
		}()

		svc.chn <- store.Update{URL: "buffer-is-busy"}
		chn1 <- store.Update{URL: "ticket-1", TriggerName: "trk1"}
		time.Sleep(5 * time.Millisecond)
		assert.Equal(t, store.Update{URL: "buffer-is-busy"}, <-svc.Updates())

		// we lost ticket-1, so the new call must succeed

		chn2 <- store.Update{URL: "ticket-2"}
		assert.Equal(t, store.Update{URL: "ticket-2"}, <-svc.Updates())

		stop()
		waitTimeout(t, closed, time.Second)
	})
}

func TestDispatcher_Updates(t *testing.T) {
	_, _, svc := prepareDispatcher(t)
	assert.Equal(t, (<-chan store.Update)(svc.chn), svc.Updates())
}

func prepareDispatcher(t *testing.T, opts ...DispatcherOption) (
	trk1 *tracker.InterfaceMock,
	trk2 *tracker.InterfaceMock,
	svc *Dispatcher,
) {
	t.Helper()
	trk1 = &tracker.InterfaceMock{NameFunc: func() string { return "trk1" }}
	trk2 = &tracker.InterfaceMock{NameFunc: func() string { return "trk2" }}
	svc, err := NewDispatcher(log.Default(), []tracker.Interface{trk1, trk2}, opts...)
	assert.NoError(t, err)
	return trk1, trk2, svc
}

func TestErrTrackerNotRegistered_Error(t *testing.T) {
	assert.Equal(t, `tracker "blah" is not registered`, ErrTrackerNotRegistered("blah").Error())
}

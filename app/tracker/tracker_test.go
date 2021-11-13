package tracker

import (
	"context"
	"github.com/cappuccinotm/dastracker/app/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"log"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestMultiTracker(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	trk1 := &trackerMock{name: "trk1"}
	trk2 := &trackerMock{name: "trk2"}

	mtrk, err := NewMultiTracker(log.Default(), []Interface{trk1, trk2})
	require.NoError(t, err)

	closed := atomicFalse

	go func() {
		err := mtrk.Run(ctx)
		assert.ErrorIs(t, err, context.Canceled)
		atomic.StoreInt32(&closed, atomicTrue)
	}()

	updates := mtrk.Updates()

	trk1.Publish(store.Update{ReceivedFrom: store.Locator{Tracker: "trk1", TaskID: "1"}})
	waitUntilForwarded(trk1.chn, mtrk.chn, 1)
	trk2.Publish(store.Update{ReceivedFrom: store.Locator{Tracker: "trk2", TaskID: "1"}})
	waitUntilForwarded(trk2.chn, mtrk.chn, 2)
	trk1.Publish(store.Update{ReceivedFrom: store.Locator{Tracker: "trk1", TaskID: "2"}})
	waitUntilForwarded(trk1.chn, mtrk.chn, 3)

	assert.Equal(t, store.Update{ReceivedFrom: store.Locator{Tracker: "trk1", TaskID: "1"}}, <-updates)
	assert.Equal(t, store.Update{ReceivedFrom: store.Locator{Tracker: "trk2", TaskID: "1"}}, <-updates)
	assert.Equal(t, store.Update{ReceivedFrom: store.Locator{Tracker: "trk1", TaskID: "2"}}, <-updates)

	cancel()
	err = mtrk.Close(context.Background())
	assert.NoError(t, err)

	assert.Len(t, trk1.CloseCalls(), 1)
	assert.Len(t, trk2.CloseCalls(), 1)

	waitTimeout(t, &closed, defaultTestTimeout, "close run")
}

func waitTimeout(t *testing.T, done *int32, timeout time.Duration, msgs ...interface{}) {
	tm := time.NewTimer(timeout)
	for {
		select {
		case <-tm.C:
			assert.FailNow(t, "timed out", msgs...)
		default:
		}

		if done := atomic.LoadInt32(done); done == atomicTrue {
			tm.Stop()
			return
		}
	}
}

const (
	defaultTestTimeout = 5 * time.Second
	atomicTrue         = int32(1)
	atomicFalse        = int32(0)
)

func waitUntilForwarded(from, to chan store.Update, numOfQueued int) {
	for {
		if len(from) == 0 && len(to) == numOfQueued {
			return
		}
	}
}

type trackerMock struct {
	InterfaceMock
	chn  chan store.Update
	once sync.Once
	name string
}

func (t *trackerMock) Name() string {
	return t.name
}

func (t *trackerMock) Updates() <-chan store.Update {
	t.once.Do(func() { t.chn = make(chan store.Update) })
	return t.chn
}

func (t *trackerMock) Publish(upd store.Update) {
	t.once.Do(func() { t.chn = make(chan store.Update) })
	t.chn <- upd
}

func (t *trackerMock) Close(ctx context.Context) error {
	close(t.chn)
	if t.InterfaceMock.CloseFunc == nil {
		t.InterfaceMock.CloseFunc = func(_ context.Context) error { return nil }
	}
	return t.InterfaceMock.Close(ctx)
}

func TestMultiTracker_Call(t *testing.T) {
	trk1 := &trackerMock{name: "trk1", InterfaceMock: InterfaceMock{
		CallFunc: func(_ context.Context, req Request) (Response, error) {
			assert.Equal(t, "trk1/method1", req.Method)
			return Response{}, nil
		}},
	}
	trk2 := &trackerMock{name: "trk2", InterfaceMock: InterfaceMock{
		CallFunc: func(_ context.Context, req Request) (Response, error) {
			assert.Equal(t, "trk2/method2", req.Method)
			return Response{}, nil
		}},
	}
	mtrk, err := NewMultiTracker(log.Default(), []Interface{trk1, trk2})
	require.NoError(t, err)

	_, err = mtrk.Call(context.Background(), Request{Method: "trk1/method1"})
	assert.NoError(t, err)

	assert.Len(t, trk1.CallCalls(), 1)

	_, err = mtrk.Call(context.Background(), Request{Method: "trk2/method2"})
	assert.NoError(t, err)

	assert.Len(t, trk2.CallCalls(), 1)

	_, err = mtrk.Call(context.Background(), Request{Method: "trk1/method1"})
	assert.NoError(t, err)

	assert.Len(t, trk1.CallCalls(), 2)

	err = mtrk.Close(context.Background())
	assert.NoError(t, err)
}

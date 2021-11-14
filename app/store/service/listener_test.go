package service

import (
	"context"
	"github.com/cappuccinotm/dastracker/app/store"
	"github.com/cappuccinotm/dastracker/app/tracker"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

// FIXME(semior): test is unstable
func TestListener_Run(t *testing.T) {
	ch := make(chan store.Update)
	trk := &tracker.InterfaceMock{
		UpdatesFunc: func() <-chan store.Update { return ch },
	}

	expectedUpd := store.Update{TriggerName: "trigger", URL: "some-url"}

	called := signaler()
	l := NewListener(trk,
		HandlerFunc(func(ctx context.Context, upd store.Update) {
			assert.Equal(t, expectedUpd, upd)
			called.done()
		}),
	)

	ctx, cancel := context.WithCancel(context.Background())
	stopped := signaler()
	go func() {
		assert.ErrorIs(t, l.Listen(ctx), context.Canceled)
		stopped.done()
	}()

	ch <- expectedUpd
	waitTimeout(t, called, time.Second, "handler not called")

	close(ch)
	cancel()
	waitTimeout(t, stopped, time.Second, "listener not stopped after cancel")
	assert.Len(t, trk.UpdatesCalls(), 1)
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

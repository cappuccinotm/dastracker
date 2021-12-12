package service

import (
	"context"
	"github.com/cappuccinotm/dastracker/app/tracker"
	"github.com/cappuccinotm/dastracker/pkg/sign"
	"github.com/stretchr/testify/assert"
	"reflect"
	"runtime"
	"sync"
	"testing"
	"time"
)

func TestActor_Listen(t *testing.T) {
	getFuncName := func(i interface{}) string {
		return runtime.FuncForPC(reflect.ValueOf(i).Pointer()).Name()
	}

	listenCalled := sign.Signal()

	var handleUpdate tracker.Handler
	hlock := &mutex{}
	a := &Actor{
		Tracker: &tracker.InterfaceMock{
			ListenFunc: func(ctx context.Context, h tracker.Handler) error {
				listenCalled.Done()
				hlock.WithLock(func() { handleUpdate = h })
				<-ctx.Done()
				return ctx.Err()
			},
		},
	}

	ctx, stop := context.WithCancel(context.Background())
	closed := sign.Signal()
	var closeErr error

	go func() {
		closeErr = a.Listen(ctx)
		closed.Done()
	}()

	assert.NoError(t, listenCalled.WaitTimeout(time.Second), "listen call")
	hlock.WithLock(func() {
		assert.Equal(t, getFuncName(a.handleUpdate), getFuncName(handleUpdate))
	})
	stop()

	assert.NoError(t, closed.WaitTimeout(time.Second), "stop")
	assert.ErrorIs(t, closeErr, context.Canceled)
}

type mutex sync.Mutex

func (l *mutex) WithLock(fn func()) {
	(*sync.Mutex)(l).Lock()
	fn()
	(*sync.Mutex)(l).Unlock()
}

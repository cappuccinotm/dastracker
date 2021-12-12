package sign

import (
	"errors"
	"time"
)

// ErrTimedOut shows that the signal wasn't fired during the
// duration, given to the WaitTimeout.
var ErrTimedOut = errors.New("timed out")

// Sign is a helper structure which holds a simple signal,
// for instance, if a goroutine is stopped.
type Sign chan struct{}

// Done signs that the sign is done.
func (d Sign) Done() {
	select {
	case <-d:
		return
	default:
		close(d)
	}
}

// Signaled immediately checks if the signal was already given.
func (d Sign) Signaled() bool {
	select {
	case <-d:
		return true
	default:
		return false
	}
}

// WaitTimeout waits until the signal is fired or timed out.
func (d Sign) WaitTimeout(timeout time.Duration) error {
	tm := time.NewTimer(timeout)
	for {
		select {
		case <-tm.C:
			return ErrTimedOut
		case <-d:
			tm.Stop()
			return nil
		}
	}
}

// Signal makes new instance of Sign.
func Signal() Sign { return make(chan struct{}) }

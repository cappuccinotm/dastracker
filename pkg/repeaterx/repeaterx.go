package repeaterx

import (
	"context"
	"errors"
	"github.com/go-pkgz/repeater/strategy"
	"time"
)

// AllowedErrors wraps the strategy.Interface with general Do method.
// It returns error if the error, returned by lambda is NOT in the list
// of provided errors.
type AllowedErrors struct {
	strategy.Interface
}

// NewAllowedErrors makes new instance of AllowedErrors.
// If strategy=nil initializes with FixedDelay 5sec, 10 times.
func NewAllowedErrors(strtg strategy.Interface) *AllowedErrors {
	if strtg == nil {
		strtg = &strategy.FixedDelay{Repeats: 10, Delay: time.Second * 5}
	}
	result := AllowedErrors{Interface: strtg}
	return &result
}

// Do repeats fun till error returned by it either is not in the list or nil.
// It considers every non-listed errors as allowed for repeating and non-critical.
func (r AllowedErrors) Do(ctx context.Context, fun func() error, errs ...error) (err error) {
	ctx, cancelFunc := context.WithCancel(ctx)
	defer cancelFunc() // ensure strategy's channel termination

	inErrors := func(err error) bool {
		for _, e := range errs {
			if errors.Is(err, e) {
				return true
			}
		}
		return false
	}

	ch := r.Start(ctx)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case _, ok := <-ch:
			if !ok { // closed channel indicates completion or early termination, set by strategy
				return err
			}
			if err = fun(); err == nil {
				return nil
			}
			if err != nil && !inErrors(err) {
				return err
			}
		}
	}
}

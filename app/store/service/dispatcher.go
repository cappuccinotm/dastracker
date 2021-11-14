package service

import (
	"context"
	"fmt"
	"github.com/cappuccinotm/dastracker/app/store"
	"github.com/cappuccinotm/dastracker/app/tracker"
	"golang.org/x/sync/errgroup"
	"log"
	"strings"
	"time"
)

// Dispatcher wraps all Interface implementations with dispatch logic
// inside it.
// It interprets Request.Method as a route in form of "tracker/method" and
// dispatches all requests to the desired trackers.
type Dispatcher struct {
	trackers map[string]tracker.Interface
	chn      chan store.Update
	log      *log.Logger

	opts
}

type opts struct {
	lostTimeout time.Duration
}

// NewDispatcher makes new instance of Dispatcher.
func NewDispatcher(lg *log.Logger, trackers []tracker.Interface, opts ...DispatcherOption) (*Dispatcher, error) {
	svc := &Dispatcher{
		trackers: map[string]tracker.Interface{},
		chn:      make(chan store.Update, maxConcurrentUpdates),
		log:      lg,
	}

	for _, opt := range opts {
		opt(&svc.opts)
	}

	for _, trk := range trackers {
		name := trk.Name()
		if _, present := svc.trackers[name]; present {
			return nil, fmt.Errorf("tracker with name %q appears twice", name)
		}
		svc.trackers[name] = trk
	}

	return svc, nil
}

// Name returns the list of the wrapped trackers.
func (m *Dispatcher) Name() string {
	names := make([]string, 0, len(m.trackers))
	for name := range m.trackers {
		names = append(names, name)
	}
	return fmt.Sprintf("Dispatcher[%s]", strings.Join(names, ", "))
}

// Call dispatches the call to the desired task tracker.
func (m *Dispatcher) Call(ctx context.Context, req tracker.Request) (tracker.Response, error) {
	trackerName, _, err := req.ParseMethod()
	if err != nil {
		return tracker.Response{}, fmt.Errorf("extract method: %w", err)
	}

	trk, present := m.trackers[trackerName]
	if !present {
		return tracker.Response{}, ErrTrackerNotRegistered(trackerName)
	}

	resp, err := trk.Call(ctx, req)
	if err != nil {
		return tracker.Response{}, fmt.Errorf("tracker %q failed to call: %w", trackerName, err)
	}

	return resp, nil
}

// Subscribe dispatches the subscription call to the desired task tracker.
func (m *Dispatcher) Subscribe(ctx context.Context, req tracker.SubscribeReq) error {
	trk, present := m.trackers[req.Tracker]
	if !present {
		return ErrTrackerNotRegistered(req.Tracker)
	}

	if err := trk.Subscribe(ctx, req); err != nil {
		return fmt.Errorf("tracker %q failed to subscribe: %w", req.Tracker, err)
	}

	return nil
}

// Updates returns the merged updates channel.
func (m *Dispatcher) Updates() <-chan store.Update { return m.chn }

// Run merges updates channel and creates a listener for updates.
// Always returns non-nil error. Blocking call.
func (m *Dispatcher) Run(ctx context.Context) error {
	ewg, ctx := errgroup.WithContext(ctx)

	for name, trk := range m.trackers {
		trk := trk
		name := name
		ewg.Go(func() error {
			if err := Listen(ctx, trk, HandlerFunc(m.forwardUpdate)); err != nil {
				return fmt.Errorf("listener for tracker %q stopped, reason: %w", name, err)
			}

			return nil
		})
	}

	if err := ewg.Wait(); err != nil {
		return fmt.Errorf("run stopped, reason: %w", err)
	}

	return nil
}

func (m *Dispatcher) forwardUpdate(ctx context.Context, upd store.Update) {
	if m.lostTimeout != 0 {
		var cancel func()
		ctx, cancel = context.WithTimeout(ctx, m.lostTimeout)
		defer cancel()
	}

	select {
	case <-ctx.Done():
		m.log.Printf("[WARN] lost update from %q about %s", upd.TriggerName, upd.URL)
	case m.chn <- upd:
	}
}

// DispatcherOption specifies options, that might be applied to the Dispatcher.
type DispatcherOption func(*opts)

// LostTimeout sets the timeout for updates, so, in case
// if the queue is full, the update might be lost.
func LostTimeout(timeout time.Duration) DispatcherOption {
	return func(o *opts) { o.lostTimeout = timeout }
}

// ErrTrackerNotRegistered indicates about the call to the tracker, that was
// not registered by the Dispatcher.
type ErrTrackerNotRegistered string

// Error returns the string representation of the error.
func (e ErrTrackerNotRegistered) Error() string {
	return fmt.Sprintf("tracker %q is not registered", string(e))
}

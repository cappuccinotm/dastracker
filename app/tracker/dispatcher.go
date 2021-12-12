package tracker

import (
	"context"
	"fmt"
	"golang.org/x/sync/errgroup"
	"log"
	"strings"
)

// Dispatcher wraps all Interface implementations with dispatch logic
// inside it.
// It interprets Request.MethodURI as a route in form of "tracker/method" and
// dispatches all requests to the desired trackers.
type Dispatcher struct {
	trackers map[string]Interface
	log      *log.Logger
}

// NewDispatcher makes new instance of Dispatcher.
func NewDispatcher(lg *log.Logger, trackers []Interface) (*Dispatcher, error) {
	svc := &Dispatcher{
		trackers: map[string]Interface{},
		log:      lg,
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
func (m *Dispatcher) Call(ctx context.Context, req Request) (Response, error) {
	trackerName, _, err := req.ParseMethodURI()
	if err != nil {
		return Response{}, fmt.Errorf("extract method: %w", err)
	}

	trk, present := m.trackers[trackerName]
	if !present {
		return Response{}, ErrTrackerNotRegistered(trackerName)
	}

	resp, err := trk.Call(ctx, req)
	if err != nil {
		return Response{}, fmt.Errorf("tracker %q failed to call: %w", trackerName, err)
	}

	return resp, nil
}

// Subscribe dispatches the subscription call to the desired task tracker.
func (m *Dispatcher) Subscribe(ctx context.Context, req SubscribeReq) error {
	trk, present := m.trackers[req.Tracker]
	if !present {
		return ErrTrackerNotRegistered(req.Tracker)
	}

	if err := trk.Subscribe(ctx, req); err != nil {
		return fmt.Errorf("tracker %q failed to subscribe: %w", req.Tracker, err)
	}

	return nil
}

// Listen merges updates channel and creates a listener for updates.
// Always returns non-nil error. Blocking call.
func (m *Dispatcher) Listen(ctx context.Context, h Handler) error {
	m.log.Printf("[INFO] running dispatcher %s", m.Name())

	ewg, ctx := errgroup.WithContext(ctx)

	for name, trk := range m.trackers {
		trk := trk
		name := name
		ewg.Go(func() error {
			if err := trk.Listen(ctx, h); err != nil {
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

// ErrTrackerNotRegistered indicates about the call to the tracker, that was
// not registered by the Dispatcher.
type ErrTrackerNotRegistered string

// Error returns the string representation of the error.
func (e ErrTrackerNotRegistered) Error() string {
	return fmt.Sprintf("tracker %q is not registered", string(e))
}

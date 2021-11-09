package tracker

import (
	"context"
	"fmt"
	"github.com/cappuccinotm/dastracker/app/store"
	"golang.org/x/sync/errgroup"
	"log"
	"strings"
)

const maxConcurrentUpdates = 15

//go:generate rm -f tracker_mock.go
//go:generate moq -out tracker_mock.go -fmt goimports . Interface

// Interface defines methods that each task tracker must implement.
type Interface interface {
	// Name returns the name of the tracker to match in the configuration.
	Name() string

	// Call makes a request to the tracker with specified method name,
	// variables and dastracker's TaskID. Response should contain the
	// TaskID of the ticket in the tracker.
	Call(ctx context.Context, req Request) (Response, error)

	// Subscribe makes a trigger with specified parameters and returns the
	// channel, to which updates will be published.
	Subscribe(ctx context.Context, req SubscribeReq) error

	// Updates returns the channel, where the updates will appear.
	// Note: the channel must be unique per each implementation of an Interface.
	Updates() <-chan store.Update

	// Close closes the connection to the tracker.
	Close(ctx context.Context) error
}

// Request describes a requests to tracker's action.
type Request struct {
	Method string
	Vars   store.Vars
	TaskID string // might be empty, in case if task is not registered yet
}

// ParseMethod parses the Method field of the request, assuming that
// the method is composed in form of "tracker/method". If the assumption does
// not hold, it returns empty strings instead.
func (r Request) ParseMethod() (tracker, method string) {
	dividerIdx := strings.IndexRune(r.Method, '/')
	if dividerIdx == -1 || dividerIdx == len(r.Method)-1 || dividerIdx == 0 {
		return "", ""
	}

	return r.Method[:dividerIdx], r.Method[dividerIdx+1:]
}

// Response describes possible return values of the Interface.Call
type Response struct {
	TaskID string // id of the created task in the tracker.
}

// SubscribeReq describes parameters of the subscription for task updates.
type SubscribeReq struct {
	TriggerName string
	Tracker     string
	Vars        store.Vars
}

// MultiTracker wraps all Interface implementations with dispatch logic
// inside it.
// It interprets Request.Method as a route in form of "tracker/method" and
// dispatches all requests to the desired trackers.
type MultiTracker struct {
	trackers map[string]Interface
	chn      chan store.Update
	cancel   func()
}

// NewMultiTracker makes new instance of MultiTracker. It also runs a listener
// for updates.
func NewMultiTracker(ctx context.Context, trackers []Interface) (*MultiTracker, error) {
	svc := &MultiTracker{
		trackers: map[string]Interface{},
		chn:      make(chan store.Update, maxConcurrentUpdates),
	}

	for _, tracker := range trackers {
		name := tracker.Name()
		if _, present := svc.trackers[name]; present {
			return nil, fmt.Errorf("tracker with name %q appears twice", name)
		}
		svc.trackers[name] = tracker
	}

	svc.run(ctx)

	return svc, nil
}

// Name returns the list of the wrapped trackers.
func (m *MultiTracker) Name() string {
	names := make([]string, 0, len(m.trackers))
	for name := range m.trackers {
		names = append(names, name)
	}
	return fmt.Sprintf("MultiTracker[%v]", names)
}

// Call dispatches the call to the desired task tracker.
func (m *MultiTracker) Call(ctx context.Context, req Request) (Response, error) {
	trackerName, _ := req.ParseMethod()
	trk, present := m.trackers[trackerName]
	if !present {
		return Response{}, fmt.Errorf("tracker %q is not registered", trackerName)
	}

	resp, err := trk.Call(ctx, req)
	if err != nil {
		return Response{}, fmt.Errorf("tracker %q failed to call: %w", trackerName, err)
	}

	return resp, nil
}

// Subscribe dispatches the subscription call to the desired task tracker.
func (m *MultiTracker) Subscribe(ctx context.Context, req SubscribeReq) error {
	trk, present := m.trackers[req.Tracker]
	if !present {
		return fmt.Errorf("tracker %q is not registered", req.Tracker)
	}

	if err := trk.Subscribe(ctx, req); err != nil {
		return fmt.Errorf("tracker %q failed to subscribe: %w", req.Tracker, err)
	}

	return nil
}

// Updates returns the merged updates channel.
func (m *MultiTracker) Updates() <-chan store.Update { return m.chn }

// Close closes all wrapped trackers and the updates channel.
func (m *MultiTracker) Close(ctx context.Context) error {
	if m.cancel != nil {
		m.cancel()
	}
	close(m.chn)

	ewg := &errgroup.Group{}
	for _, trk := range m.trackers {
		trk := trk
		ewg.Go(func() error {
			if err := trk.Close(ctx); err != nil {
				return fmt.Errorf("close %q: %w", trk.Name(), err)
			}
			return nil
		})
	}

	if err := ewg.Wait(); err != nil {
		return fmt.Errorf("closing trackers: %w", err)
	}

	return nil
}

// run merges updates channel and creates a listener for updates.
func (m *MultiTracker) run(ctx context.Context) {
	ctx, m.cancel = context.WithCancel(ctx)

	listener := func(name string, ch <-chan store.Update) {
		for {
			select {
			case upd := <-ch:
				m.chn <- upd
			case <-ctx.Done():
				log.Printf("[WARN] closing updates listener for %q", name)
				return
			}
		}
	}

	for name, trk := range m.trackers {
		go listener(name, trk.Updates())
	}
}

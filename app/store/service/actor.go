package service

import (
	"context"
	"errors"
	"fmt"
	"github.com/cappuccinotm/dastracker/app/flow"
	"github.com/cappuccinotm/dastracker/app/store"
	"github.com/cappuccinotm/dastracker/app/store/engine"
	"github.com/cappuccinotm/dastracker/app/tracker"
	"github.com/go-pkgz/syncs"
	"log"
	"time"
)

// maxConcurrentUpdates defines the maximum number of goroutines, that may
// process updates concurrently at the same time
const maxConcurrentUpdates = 15
const defaultUpdateTimeout = 1 * time.Minute

// Actor receives updates from the Tracker and follows the calls the actions in the
// wrapped tracker according to the job provided by the Flow.
type Actor struct {
	Tracker       tracker.Interface
	Engine        engine.Interface
	Flow          flow.Interface
	Log           *log.Logger
	UpdateTimeout time.Duration
	cancel        func()
}

// Run runs the updates' listener.
// Always returns non-nil error.
// Blocking call.
func (s *Actor) Run(ctx context.Context) error {
	if s.UpdateTimeout == 0 {
		s.UpdateTimeout = defaultUpdateTimeout
	}

	ctx, s.cancel = context.WithCancel(ctx)

	s.Log.Printf("[INFO] started updates listener")

	updates := s.Tracker.Updates()
	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("run stopped, reason: %w", ctx.Err())
		case upd := <-updates:
			s.handleUpdate(ctx, upd)
		}
	}
}

// Close closes the actor's listener.
func (s *Actor) Close(ctx context.Context) error {
	s.cancel()
	return s.Tracker.Close(ctx)
}

// handleUpdate runs the jobs concurrently over the given update
// produces goroutines
func (s *Actor) handleUpdate(ctx context.Context, upd store.Update) {
	// do not run the update handler if the context is already done
	if ctxDone(ctx) {
		return
	}

	ctx, cancel := context.WithTimeout(ctx, s.UpdateTimeout)
	defer cancel()

	jobs, err := s.Flow.GetSubscribedJobs(ctx, upd.TriggerName)
	if err != nil {
		s.Log.Printf("[WARN] failed to get subscribed jobs for trigger %q on update from %s: %v",
			upd.TriggerName, upd.ReceivedFrom, err)
		return
	}

	swg := syncs.NewSizedGroup(maxConcurrentUpdates, syncs.Context(ctx))
	for _, job := range jobs {
		job := job
		swg.Go(func(ctx context.Context) {
			if err := s.runJob(ctx, job, upd); err != nil {
				s.Log.Printf("[WARN] failed to handle update %v for job %q: %v", upd, job.Name, err)
				return
			}
		})
	}
	swg.Wait()
}

// runJob goes through the job's flow
// thread-safe
func (s *Actor) runJob(ctx context.Context, job store.Job, upd store.Update) error {
	ticket, err := s.Engine.Get(ctx, engine.GetRequest{Locator: upd.ReceivedFrom})
	switch {
	case errors.Is(err, engine.ErrNotFound):
		ticket = store.Ticket{TrackerIDs: map[string]string{}}
	case err != nil:
		return fmt.Errorf("get ticket by locator %s: %w", upd.ReceivedFrom, err)
	}

	ticket.Patch(upd)

	for _, act := range job.Actions {
		// TODO(semior): add support of detached calls
		vars, err := act.With.Evaluate(upd)
		if err != nil {
			return fmt.Errorf("evaluate variables for %q action: %w", act.Name, err)
		}

		resp, err := s.Tracker.Call(ctx, tracker.Request{Method: act.Name, Vars: vars, Ticket: ticket})
		if err != nil {
			return fmt.Errorf("call to %s: %w", act.Name, err)
		}

		ticket.TrackerIDs[resp.Tracker] = resp.TaskID
	}

	if ticket.ID == "" {
		if ticket.ID, err = s.Engine.Create(ctx, ticket); err != nil {
			return fmt.Errorf("create ticket: %w", err)
		}

		return nil
	}

	if err = s.Engine.Update(ctx, ticket); err != nil {
		return fmt.Errorf("update ticket from %s: %w", upd.ReceivedFrom, err)
	}

	return nil
}

func ctxDone(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return true
	default:
		return false
	}
}

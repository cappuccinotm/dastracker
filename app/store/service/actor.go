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

// Actor receives updates from the Tracker and follows the calls the actions in the
// wrapped tracker according to the job provided by the Flow.
type Actor struct {
	Tracker       tracker.Interface
	Engine        engine.Interface
	Flow          flow.Interface
	Log           *log.Logger
	UpdateTimeout time.Duration
}

// Listen runs the updates' listener.
// Always returns non-nil error.
// Blocking call.
func (s *Actor) Listen(ctx context.Context) error {
	if err := s.Tracker.Listen(ctx, tracker.HandlerFunc(s.handleUpdate)); err != nil {
		return fmt.Errorf("updates listener stopped, reason: %w", err)
	}

	return nil
}

// handleUpdate runs the jobs concurrently over the given update
func (s *Actor) handleUpdate(ctx context.Context, upd store.Update) {
	if s.UpdateTimeout != 0 {
		var cancel func()
		ctx, cancel = context.WithTimeout(ctx, s.UpdateTimeout)
		defer cancel()
	}

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
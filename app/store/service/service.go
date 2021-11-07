package service

import (
	"context"
	"errors"
	"fmt"
	"github.com/cappuccinotm/dastracker/app/flow"
	"github.com/cappuccinotm/dastracker/app/store"
	"github.com/cappuccinotm/dastracker/app/store/engine"
	"github.com/cappuccinotm/dastracker/app/tracker"
)

// DataStore wraps engine.Interface implementations with methods, common
// for each engine implementation.
type DataStore struct {
	eng      engine.Interface
	trackers map[string]tracker.Interface
}

// Run sets up the triggers, describes in the given jobs.
func (s *DataStore) Run(ctx context.Context, jobs []flow.Job) error {
	for _, job := range jobs {
		trk := s.trackers[job.Trigger.Tracker]

		if err := trk.Subscribe(ctx, job.Trigger.With, s.taskUpdatedCb(job)); err != nil {
			return fmt.Errorf("set up trigger in %q for the job %q: %w", job.Trigger.Tracker, job.Name, err)
		}
	}
	return nil
}

// taskUpdatedCb is a callback for tracker's triggers.
func (s *DataStore) taskUpdatedCb(job flow.Job) tracker.Subscriber {
	return subscriberFunc(func(ctx context.Context, update store.Update) error {
		return s.onTaskUpdated(ctx, job, update)
	})
}

func (s *DataStore) onTaskUpdated(ctx context.Context, job flow.Job, upd store.Update) error {
	ticket, err := s.eng.Get(ctx, engine.GetRequest{Locator: upd.ReceivedFrom})
	switch {
	case errors.Is(err, engine.ErrNotFound):
		ticket = store.Ticket{TrackerIDs: map[string]string{}}
	case err != nil:
		return fmt.Errorf("get ticket by locator %s: %w", upd.ReceivedFrom, err)
	}

	ticket.Patch(upd)

	for _, act := range job.Actions {
		trkName, mtd := act.Path()
		trk := s.trackers[trkName]

		vars, err := act.With.Evaluate(upd)
		if err != nil {
			return fmt.Errorf("evaluate variables for %q action: %w", act.Name, err)
		}

		taskID, taskRegistered := ticket.TrackerIDs[trkName]

		resp, err := trk.Call(ctx, tracker.Request{
			Method: mtd, Vars: vars, TaskID: taskID,
		})
		if err != nil {
			return fmt.Errorf("call to %s: %w", act.Name, err)
		}

		if !taskRegistered {
			ticket.TrackerIDs[trkName] = resp.TaskID
		}
	}

	if ticket.ID == "" {
		if ticket.ID, err = s.eng.Create(ctx, ticket); err != nil {
			return fmt.Errorf("create ticket: %w", err)
		}

		return nil
	}

	if err = s.eng.Update(ctx, ticket); err != nil {
		return fmt.Errorf("update ticket from %s: %w", upd.ReceivedFrom, err)
	}

	return nil
}

type subscriberFunc func(context.Context, store.Update) error

// TaskUpdated calls the wrapped function.
func (f subscriberFunc) TaskUpdated(ctx context.Context, upd store.Update) error {
	return f(ctx, upd)
}

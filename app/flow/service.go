package flow

import (
	"context"
	"errors"
	"fmt"
	"github.com/cappuccinotm/dastracker/app/provider"
	"github.com/cappuccinotm/dastracker/app/store"
	"github.com/cappuccinotm/dastracker/app/store/engine"
	"github.com/cappuccinotm/dastracker/app/tracker"
)

// maxConcurrentUpdates defines the maximum number of goroutines, that may
// process updates concurrently at the same time
const maxConcurrentUpdates = 15

// DataStore wraps engine.Interface implementations with methods, common
// for each engine implementation.
type DataStore struct {
	Trackers map[string]tracker.Interface
	Engine   engine.Interface
	Provider provider.Interface
}

// HandleUpdate processes new update by calling jobs in it.
func (s *DataStore) HandleUpdate(ctx context.Context, jobName string, upd store.Update) error {
	job, err := s.Provider.GetJob(ctx, jobName)
	if err != nil {
		return fmt.Errorf("get job %q: %w", jobName, err)
	}

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
		trkName, mtd := act.Path()
		trk := s.Trackers[trkName]

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

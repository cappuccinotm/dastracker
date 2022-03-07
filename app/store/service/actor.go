package service

import (
	"context"
	"errors"
	"fmt"
	"github.com/cappuccinotm/dastracker/app/errs"
	"github.com/cappuccinotm/dastracker/app/flow"
	"github.com/cappuccinotm/dastracker/app/store"
	"github.com/cappuccinotm/dastracker/app/store/engine"
	"github.com/cappuccinotm/dastracker/app/tracker"
	"github.com/cappuccinotm/dastracker/pkg/logx"
	"github.com/go-pkgz/syncs"
	"golang.org/x/sync/errgroup"
	"strings"
	"time"
)

// maxConcurrentUpdates defines the maximum number of goroutines, that may
// process updates concurrently at the same time
const maxConcurrentUpdates = 15

// Actor receives updates from the Tracker and follows the calls the actions in the
// wrapped tracker according to the job provided by the Flow.
type Actor struct {
	Trackers      map[string]tracker.Interface
	TicketsStore  engine.Tickets
	Flow          flow.Interface
	Log           logx.Logger
	UpdateTimeout time.Duration
}

// Listen runs the updates' listener. Always returns non-nil error.
// Blocking call.
func (s *Actor) Listen(ctx context.Context) error {
	if err := s.registerTriggers(ctx); err != nil {
		return fmt.Errorf("register triggers: %w", err)
	}

	ewg, ctx := errgroup.WithContext(ctx)

	for name, trk := range s.Trackers {
		trk := trk
		name := name
		ewg.Go(func() error {
			s.Log.Printf("[INFO] starting listener for %s", name)

			if err := trk.Listen(ctx, tracker.HandlerFunc(s.handleUpdate)); err != nil {
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

// handleUpdate runs the jobs concurrently over the given update
func (s *Actor) handleUpdate(ctx context.Context, upd store.Update) {
	s.Log.Printf("[DEBUG] received update %+v", upd)

	if s.UpdateTimeout != 0 {
		var cancel func()
		ctx, cancel = context.WithTimeout(ctx, s.UpdateTimeout)
		defer cancel()
	}

	jobs, err := s.Flow.ListSubscribedJobs(ctx, upd.TriggerName)
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
	s.Log.Printf("[DEBUG] running job %s, triggered by %s", job.Name, job.TriggerName)

	ticket, err := s.TicketsStore.Get(ctx, engine.GetRequest{Locator: upd.ReceivedFrom})
	if err != nil && !errors.Is(err, errs.ErrNotFound) {
		return fmt.Errorf("get ticket by locator %s: %w", upd.ReceivedFrom, err)
	}

	ticket.Patch(upd)

	for _, act := range job.Actions {
		s.Log.Printf("[DEBUG] running action %s with vars %+v", act.Name, act.With)

		// TODO(semior): add support of detached calls
		vars, err := store.Evaluate(act.With, upd)
		if err != nil {
			return fmt.Errorf("evaluate variables for %q action: %w", act.Name, err)
		}

		trkName, method := parseMethodURI(act.Name)

		resp, err := s.Trackers[trkName].Call(ctx, tracker.Request{
			Method: method,
			Ticket: ticket,
			Vars:   vars,
		})
		if err != nil {
			return fmt.Errorf("call to %s: %w", act.Name, err)
		}

		ticket.TrackerIDs.Set(trkName, resp.TaskID)
	}

	if ticket.ID == "" {
		if ticket.ID, err = s.TicketsStore.Create(ctx, ticket); err != nil {
			return fmt.Errorf("create ticket: %w", err)
		}

		return nil
	}

	if err = s.TicketsStore.Update(ctx, ticket); err != nil {
		return fmt.Errorf("update ticket from %s: %w", upd.ReceivedFrom, err)
	}

	return nil
}

func (s *Actor) registerTriggers(ctx context.Context) error {
	triggers, err := s.Flow.ListTriggers(ctx)
	if err != nil {
		return fmt.Errorf("list triggers: %w", err)
	}

	ewg, ctx := errgroup.WithContext(ctx)

	for _, trigger := range triggers {
		trigger := trigger
		ewg.Go(func() error {
			vars, err := store.Evaluate(trigger.With, store.Update{})
			if err != nil {
				return fmt.Errorf("evaluate variables for %q trigger: %w", trigger.Name, err)
			}

			err = s.Trackers[trigger.Tracker].Subscribe(ctx, tracker.SubscribeReq{
				TriggerName: trigger.Name,
				Vars:        vars,
			})
			if err != nil {
				return fmt.Errorf("subscribe %q to %q: %w", trigger.Name, trigger.Tracker, err)
			}
			return nil
		})
	}

	if err = ewg.Wait(); err != nil {
		// todo unsubscribe from already registered triggers
		return fmt.Errorf("one of trackers refused to register triggers: %w", err)
	}

	// todo unregister triggers on shutdown

	return nil
}

func parseMethodURI(s string) (tracker, method string) {
	dividerIdx := strings.IndexRune(s, '/')
	return s[:dividerIdx], s[dividerIdx+1:]
}

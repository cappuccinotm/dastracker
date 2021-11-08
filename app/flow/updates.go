package flow

import (
	"context"
	"fmt"
	"github.com/cappuccinotm/dastracker/app/store"
	"github.com/cappuccinotm/dastracker/app/tracker"
	"github.com/go-pkgz/syncs"
	"golang.org/x/sync/errgroup"
	"log"
	"time"
)

// UpdatesDispatcher aggregates trigger channels and checks for updates in them.
// Each time update received it calls Handler's callback.
type UpdatesDispatcher struct {
	Trackers          map[string]tracker.Interface
	Handler           updatesHandler
	ProcessingTimeout time.Duration

	triggers map[string]*triggerNode
}

type triggerNode struct {
	updateCh       <-chan store.Update
	subscribedJobs []string
}

//go:generate rm -f updates_handler_mock.go
//go:generate moq -out updates_handler_mock.go -fmt goimports . updatesHandler

type updatesHandler interface {
	HandleUpdate(ctx context.Context, jobName string, upd store.Update) error
}

// Close closes all registered trackers.
func (l *UpdatesDispatcher) Close(ctx context.Context) error {
	wg := &errgroup.Group{}
	for _, trk := range l.Trackers {
		trk := trk
		wg.Go(func() error { return trk.Close(ctx) })
	}
	return wg.Wait()
}

// Run sets up the triggers, describes in the given jobs.
func (l *UpdatesDispatcher) Run(ctx context.Context, triggers []store.Trigger, jobs []store.Job) error {
	// preparing jobs subscriptions
	l.triggers = map[string]*triggerNode{}
	for _, job := range jobs {
		node, present := l.triggers[job.TriggerName]
		if !present {
			node = &triggerNode{}
		}

		node.subscribedJobs = append(node.subscribedJobs, job.Name)
		l.triggers[job.TriggerName] = node
	}

	// setting up triggers
	for _, trigger := range triggers {
		trk := l.Trackers[trigger.Tracker]
		updCh, err := trk.Subscribe(ctx, trigger.With)
		if err != nil {
			return fmt.Errorf("set up trigger %q in %q: %w", trigger.Name, trigger.Tracker, err)
		}

		node := l.triggers[trigger.Name]
		node.updateCh = updCh
	}

	if err := l.listenForUpdates(ctx); err != nil {
		return fmt.Errorf("listen for updates: %w", err)
	}

	return nil
}

func (l *UpdatesDispatcher) listenForUpdates(ctx context.Context) error {
	for {
		if err := ctxDone(ctx); err != nil {
			return fmt.Errorf("context is done: %w", err)
		}

		l.lookupUpdates(ctx)
	}
}

func (l *UpdatesDispatcher) lookupUpdates(ctx context.Context) {
	swg := syncs.NewSizedGroup(maxConcurrentUpdates, syncs.Context(ctx), syncs.Preemptive)
	for _, trigger := range l.triggers {
		trigger := trigger

		upd, hasUpdate := checkForUpdate(trigger.updateCh)
		if !hasUpdate {
			continue
		}

		for _, job := range trigger.subscribedJobs {
			job := job

			swg.Go(func(ctx context.Context) {
				updCtx, cancel := context.WithTimeout(ctx, l.ProcessingTimeout)
				defer cancel()

				if err := l.Handler.HandleUpdate(updCtx, job, upd); err != nil {
					log.Printf("[WARN] failed to process task update on job %s with update %v: %v",
						job, upd, err)
				}
			})
		}
	}
	swg.Wait()
}

func checkForUpdate(ch <-chan store.Update) (upd store.Update, ok bool) {
	select {
	case upd := <-ch:
		return upd, true
	default:
		return store.Update{}, false
	}
}

func ctxDone(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}

package service

import (
	"context"
	"github.com/cappuccinotm/dastracker/app/errs"
	"github.com/cappuccinotm/dastracker/app/flow"
	"github.com/cappuccinotm/dastracker/app/store"
	"github.com/cappuccinotm/dastracker/app/store/engine"
	"github.com/cappuccinotm/dastracker/app/tracker"
	"github.com/cappuccinotm/dastracker/pkg/sign"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"log"
	"reflect"
	"runtime"
	"sync"
	"testing"
	"time"
)

func TestActor_Listen(t *testing.T) {
	getFuncName := func(i interface{}) string {
		return runtime.FuncForPC(reflect.ValueOf(i).Pointer()).Name()
	}

	listenCalled := sign.Signal()

	var handleUpdate tracker.Handler
	hlock := &mutex{}
	a := &Actor{
		Trackers: map[string]tracker.Interface{
			"blah": &tracker.InterfaceMock{
				ListenFunc: func(ctx context.Context, h tracker.Handler) error {
					listenCalled.Done()
					hlock.WithLock(func() { handleUpdate = h })
					<-ctx.Done()
					return ctx.Err()
				},
				SubscribeFunc: func(ctx context.Context, req tracker.SubscribeReq) error {
					assert.Equal(t, tracker.SubscribeReq{
						TriggerName: "someTrigger",
						Vars:        map[string]string{"foo": "bar"},
					}, req)
					return nil
				},
			},
		},
		Flow: &flow.InterfaceMock{
			ListTriggersFunc: func(ctx context.Context) ([]store.Trigger, error) {
				return []store.Trigger{{
					Name:    "someTrigger",
					Tracker: "blah",
					With:    map[string]string{"foo": "bar"},
				}}, nil
			},
		},
		Log: log.Default(),
	}

	ctx, stop := context.WithCancel(context.Background())
	closed := sign.Signal()
	var closeErr error

	go func() {
		closeErr = a.Listen(ctx)
		closed.Done()
	}()

	assert.NoError(t, listenCalled.WaitTimeout(time.Second), "listen call")
	hlock.WithLock(func() {
		assert.Equal(t, getFuncName(a.handleUpdate), getFuncName(handleUpdate))
	})
	stop()

	assert.NoError(t, closed.WaitTimeout(time.Second), "stop")
	assert.ErrorIs(t, closeErr, context.Canceled)
}

type mutex sync.Mutex

func (l *mutex) WithLock(fn func()) {
	(*sync.Mutex)(l).Lock()
	fn()
	(*sync.Mutex)(l).Unlock()
}

func TestActor_runJob(t *testing.T) {
	t.Run("update of the existing ticket", func(t *testing.T) {
		initialTicket := store.Ticket{
			ID: "ticket-id",
			TrackerIDs: map[string]string{
				"other-tracker": "other-id",
				"tracker":       "task-id",
			},
			Content: store.Content{
				Body:   "ticket-body",
				Title:  "ticket-title",
				Fields: map[string]string{"field": "value"},
			},
		}
		expectedTrackerReq := tracker.Request{
			Method: "create-or-update",
			Ticket: store.Ticket{
				ID: "ticket-id",
				TrackerIDs: map[string]string{
					"other-tracker": "other-id",
					"tracker":       "task-id",
				},
				Content: store.Content{
					Body:   "updated-body",
					Title:  "updated-title",
					Fields: map[string]string{"field": "updated-value"},
				},
			},
			Vars: map[string]string{
				"msg":  "Task with id task-id has been updated",
				"body": "Body: updated-body",
				"url":  "https://blah.com",
			},
		}
		svc := &Actor{
			Log: log.Default(),
			TicketsStore: &engine.TicketsMock{
				GetFunc: func(_ context.Context, req engine.GetRequest) (store.Ticket, error) {
					assert.Equal(t, engine.GetRequest{
						Locator: store.Locator{Tracker: "tracker", ID: "task-id"},
					}, req)
					return initialTicket, nil
				},
				UpdateFunc: func(_ context.Context, ticket store.Ticket) error {
					expectedTicket := initialTicket
					expectedTicket.TrackerIDs["new-tracker"] = "new-task-id"
					expectedTicket.Content.Body = "updated-body"
					expectedTicket.Content.Title = "updated-title"
					expectedTicket.Content.Fields["field"] = "updated-value"
					assert.Equal(t, expectedTicket, ticket)
					return nil
				},
			},
			Trackers: map[string]tracker.Interface{
				"tracker": &tracker.InterfaceMock{
					CallFunc: func(_ context.Context, req tracker.Request) (tracker.Response, error) {
						assert.Equal(t, expectedTrackerReq, req)
						return tracker.Response{TaskID: "new-task-id"}, nil
					},
				},
			},
		}

		err := svc.runJob(
			context.Background(),
			store.Job{
				Name:        "test-job",
				TriggerName: "test-trigger",
				Actions: []store.Action{
					{Name: "tracker/create-or-update", With: map[string]string{
						"msg":  "Task with id {{.Update.ReceivedFrom.ID}} has been updated",
						"body": "Body: {{.Update.Content.Body}}",
						"url":  "{{.Update.URL}}",
					}},
				},
			},
			store.Update{
				URL:          "https://blah.com",
				ReceivedFrom: store.Locator{Tracker: "tracker", ID: "task-id"},
				Content: store.Content{
					Body:   "updated-body",
					Title:  "updated-title",
					Fields: map[string]string{"field": "updated-value"},
				},
			},
		)
		require.NoError(t, err)
	})

	t.Run("creating new ticket", func(t *testing.T) {
		initialTicket := store.Ticket{
			Content: store.Content{
				Body:   "ticket-body",
				Title:  "ticket-title",
				Fields: map[string]string{"field": "value"},
			},
		}
		expectedTrackerReq := tracker.Request{
			Method: "create-or-update",
			Ticket: store.Ticket{
				TrackerIDs: map[string]string{"tracker": "task-id"},
				Content: store.Content{
					Body:   "updated-body",
					Title:  "updated-title",
					Fields: map[string]string{"field": "updated-value"},
				},
			},
			Vars: map[string]string{
				"msg":  "Task with id task-id has been updated",
				"body": "Body: updated-body",
				"url":  "https://blah.com",
			},
		}
		svc := &Actor{
			Log: log.Default(),
			TicketsStore: &engine.TicketsMock{
				GetFunc: func(_ context.Context, req engine.GetRequest) (store.Ticket, error) {
					assert.Equal(t, engine.GetRequest{
						Locator: store.Locator{Tracker: "tracker", ID: "task-id"},
					}, req)
					return store.Ticket{}, errs.ErrNotFound
				},
				CreateFunc: func(_ context.Context, ticket store.Ticket) (string, error) {
					expectedTicket := initialTicket
					expectedTicket.TrackerIDs = map[string]string{"new-tracker": "new-task-id", "tracker": "task-id"}
					expectedTicket.Content.Body = "updated-body"
					expectedTicket.Content.Title = "updated-title"
					expectedTicket.Content.Fields["field"] = "updated-value"
					assert.Equal(t, expectedTicket, ticket)
					return "ticket-id", nil
				},
			},
			Trackers: map[string]tracker.Interface{
				"new-tracker": &tracker.InterfaceMock{
					CallFunc: func(_ context.Context, req tracker.Request) (tracker.Response, error) {
						assert.Equal(t, expectedTrackerReq, req)
						return tracker.Response{TaskID: "new-task-id"}, nil
					},
				},
			},
		}

		err := svc.runJob(
			context.Background(),
			store.Job{
				Name:        "test-job",
				TriggerName: "test-trigger",
				Actions: []store.Action{
					{Name: "new-tracker/create-or-update", With: map[string]string{
						"msg":  "Task with id {{.Update.ReceivedFrom.ID}} has been updated",
						"body": "Body: {{.Update.Content.Body}}",
						"url":  "{{.Update.URL}}",
					}},
				},
			},
			store.Update{
				URL:          "https://blah.com",
				ReceivedFrom: store.Locator{Tracker: "tracker", ID: "task-id"},
				Content: store.Content{
					Body:   "updated-body",
					Title:  "updated-title",
					Fields: map[string]string{"field": "updated-value"},
				},
			},
		)
		require.NoError(t, err)
	})
}

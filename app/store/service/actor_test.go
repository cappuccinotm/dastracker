package service

import (
	"context"
	"errors"
	"github.com/cappuccinotm/dastracker/app/errs"
	"github.com/cappuccinotm/dastracker/app/store"
	"github.com/cappuccinotm/dastracker/app/store/engine"
	"github.com/cappuccinotm/dastracker/app/tracker"
	"github.com/cappuccinotm/dastracker/lib"
	"github.com/cappuccinotm/dastracker/pkg/logx"
	"github.com/cappuccinotm/dastracker/pkg/sign"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

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
			TaskID: "task-id",
			Vars: map[string]string{
				"msg":  "Task with id task-id has been updated",
				"body": "Body: updated-body",
				"url":  "https://blah.com",
			},
		}
		svc := &Actor{
			Log: logx.Nop(),
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
			Vars: map[string]string{
				"msg":  "Task with id task-id has been updated",
				"body": "Body: updated-body",
				"url":  "https://blah.com",
			},
		}
		svc := &Actor{
			Log: logx.Nop(),
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

func TestActor_Listen(t *testing.T) {
	flow := &engine.FlowMock{}
	subsStore := &engine.SubscriptionsMock{}
	trk1, trk2 := &tracker.InterfaceMock{}, &tracker.InterfaceMock{}
	svc := &Actor{
		Flow: flow,
		SubscriptionsManager: &SubscriptionsManager{
			Store:   subsStore,
			BaseURL: "https://localhost",
		},
		Trackers: map[string]tracker.Interface{
			"tracker-1": trk1,
			"tracker-2": trk2,
		},
		Log: logx.Nop(),
	}

	const varsEvaluated = "variable evaluated"
	require.NoError(t, os.Setenv("VE", varsEvaluated))

	// register triggers
	{
		flow.ListTriggersFunc = func(ctx context.Context) ([]store.Trigger, error) {
			return []store.Trigger{
				{Name: "trigger-1", Tracker: "tracker-1", With: lib.Vars{"test": `{{ env "VE" }}`}},
				{Name: "trigger-2", Tracker: "tracker-2", With: lib.Vars{"test1": `{{ env "VE" }}`}},
			}, nil
		}
		subsStore.CreateFunc = func(ctx context.Context, sub store.Subscription) (string, error) {
			switch sub.TriggerName {
			case "trigger-1":
				assert.Equal(t, store.Subscription{
					TrackerName: "tracker-1",
					TriggerName: "trigger-1",
					BaseURL:     "https://localhost",
				}, sub)
				return "subscription-1", nil
			case "trigger-2":
				assert.Equal(t, store.Subscription{
					TrackerName: "tracker-2",
					TriggerName: "trigger-2",
					BaseURL:     "https://localhost",
				}, sub)
				return "subscription-2", nil
			}
			assert.Fail(t, "no match for create subscription", sub)
			return "", nil
		}
		trk1.SubscribeFunc = func(ctx context.Context, req tracker.SubscribeReq) (tracker.SubscribeResp, error) {
			assert.Equal(t, tracker.SubscribeReq{
				Vars:       lib.Vars{"test": varsEvaluated},
				WebhookURL: "https://localhost/subscription-1",
			}, req)
			return tracker.SubscribeResp{TrackerRef: "tracker-ref-1"}, nil
		}
		trk2.SubscribeFunc = func(ctx context.Context, req tracker.SubscribeReq) (tracker.SubscribeResp, error) {
			assert.Equal(t, tracker.SubscribeReq{
				Vars:       lib.Vars{"test1": varsEvaluated},
				WebhookURL: "https://localhost/subscription-2",
			}, req)
			return tracker.SubscribeResp{TrackerRef: "tracker-ref-2"}, nil
		}
		subsStore.GetFunc = func(ctx context.Context, id string) (store.Subscription, error) {
			switch id {
			case "subscription-1":
				return store.Subscription{
					ID:          "subscription-1",
					TrackerName: "tracker-1",
					TriggerName: "trigger-1",
					BaseURL:     "https://localhost",
				}, nil
			case "subscription-2":
				return store.Subscription{
					ID:          "subscription-2",
					TrackerName: "tracker-2",
					TriggerName: "trigger-2",
					BaseURL:     "https://localhost",
				}, nil
			}
			assert.Fail(t, "no match for get subscription", id)
			return store.Subscription{}, nil
		}
		subsStore.UpdateFunc = func(ctx context.Context, sub store.Subscription) error {
			switch sub.ID {
			case "subscription-1":
				assert.Equal(t, store.Subscription{
					ID:          "subscription-1",
					TrackerName: "tracker-1",
					TriggerName: "trigger-1",
					BaseURL:     "https://localhost",
					TrackerRef:  "tracker-ref-1",
				}, sub)
			case "subscription-2":
				assert.Equal(t, store.Subscription{
					ID:          "subscription-2",
					TrackerName: "tracker-2",
					TriggerName: "trigger-2",
					BaseURL:     "https://localhost",
					TrackerRef:  "tracker-ref-2",
				}, sub)
			default:
				require.Fail(t, "no match for update subscription", sub.ID)
			}
			return nil
		}
	}

	trk1Started, trk2Started := sign.Signal(), sign.Signal()
	trk1Closed, trk2Closed := sign.Signal(), sign.Signal()

	// run listeners
	{
		trk1.ListenFunc = func(ctx context.Context, h tracker.Handler) error {
			assert.NotNil(t, h)
			trk1Started.Done()
			<-ctx.Done()
			trk1Closed.Done()
			return ctx.Err()
		}
		trk2.ListenFunc = func(ctx context.Context, h tracker.Handler) error {
			assert.NotNil(t, h)
			trk2Started.Done()
			<-ctx.Done()
			trk2Closed.Done()
			return ctx.Err()
		}
	}

	// unregister triggers
	{
		subsStore.ListFunc = func(ctx context.Context, trackerID string) ([]store.Subscription, error) {
			assert.Empty(t, trackerID)
			return []store.Subscription{
				{
					ID:          "subscription-1",
					TrackerName: "tracker-1",
					TriggerName: "trigger-1",
					BaseURL:     "https://localhost",
					TrackerRef:  "tracker-ref-1",
				},
				{
					ID:          "subscription-2",
					TrackerName: "tracker-2",
					TriggerName: "trigger-2",
					BaseURL:     "https://localhost",
					TrackerRef:  "tracker-ref-2",
				},
			}, nil
		}
		trk1.UnsubscribeFunc = func(ctx context.Context, req tracker.UnsubscribeReq) error {
			assert.Equal(t, tracker.UnsubscribeReq{
				TrackerRef: "tracker-ref-1",
			}, req)
			return nil
		}
		trk2.UnsubscribeFunc = func(ctx context.Context, req tracker.UnsubscribeReq) error {
			assert.Equal(t, tracker.UnsubscribeReq{
				TrackerRef: "tracker-ref-2",
			}, req)
			return nil
		}
		subsStore.DeleteFunc = func(ctx context.Context, subID string) error {
			switch subID {
			case "subscription-1", "subscription-2":
			default:
				require.FailNow(t, "no case for delete subscription", subID)
			}
			return nil
		}
	}

	ctx, cancel := context.WithCancel(context.Background())

	svcClosed := sign.Signal()
	go func() {
		err := svc.Listen(ctx)
		svcClosed.Done()
		if !errors.Is(err, context.Canceled) {
			require.FailNow(t, "listen stopped with other err", err)
		}
	}()
	require.NoError(t, trk1Started.WaitTimeout(3*time.Second))
	require.NoError(t, trk2Started.WaitTimeout(3*time.Second))

	cancel()

	require.NoError(t, trk1Closed.WaitTimeout(3*time.Second))
	require.NoError(t, trk2Closed.WaitTimeout(3*time.Second))
	require.NoError(t, svcClosed.WaitTimeout(3*time.Second))
}

func TestActor_HandleWebhook(t *testing.T) {
	trk := &tracker.InterfaceMock{
		HandleWebhookFunc: func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(`called`))
			w.WriteHeader(http.StatusOK)
		},
	}
	svc := &Actor{
		Log:      logx.Nop(),
		Trackers: map[string]tracker.Interface{"trk": trk},
	}

	ts := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			r = r.WithContext(store.PutSubscription(r.Context(),
				store.Subscription{TrackerName: "trk"}))
			svc.HandleWebhook(w, r)
		},
	))
	defer ts.Close()

	resp, err := ts.Client().Get(ts.URL)
	require.NoError(t, err)

	b, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Equal(t, []byte(`called`), b)

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

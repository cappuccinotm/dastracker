package service

import (
	"context"
	"github.com/cappuccinotm/dastracker/app/store"
	"github.com/cappuccinotm/dastracker/app/store/engine"
	"github.com/cappuccinotm/dastracker/pkg/sign"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestSubscriptionsManager_Create(t *testing.T) {
	svc := &SubscriptionsManager{
		BaseURL: "https://localhost",
		Store: &engine.SubscriptionsMock{
			CreateFunc: func(_ context.Context, sub store.Subscription) (string, error) {
				assert.Equal(t, store.Subscription{
					TrackerName: "tracker",
					TriggerName: "trigger",
					BaseURL:     "https://localhost",
				}, sub)
				return "id", nil
			},
		},
	}

	sub, err := svc.Create(context.Background(), "tracker", "trigger")
	require.NoError(t, err)
	assert.Equal(t, store.Subscription{
		ID:          "id",
		TrackerRef:  "", // not filled yet
		TrackerName: "tracker",
		TriggerName: "trigger",
		BaseURL:     "https://localhost",
	}, sub)
}

func TestSubscriptionsManager_SetTrackerRef(t *testing.T) {
	svc := &SubscriptionsManager{
		Store: &engine.SubscriptionsMock{
			GetFunc: func(ctx context.Context, subID string) (store.Subscription, error) {
				assert.Equal(t, "sub-id", subID)
				return store.Subscription{
					ID:          "sub-id",
					TrackerRef:  "",
					TrackerName: "tracker-name",
					TriggerName: "trigger-name",
					BaseURL:     "https://localhost",
				}, nil
			},
			UpdateFunc: func(ctx context.Context, sub store.Subscription) error {
				assert.Equal(t, store.Subscription{
					ID:          "sub-id",
					TrackerRef:  "tracker-ref",
					TrackerName: "tracker-name",
					TriggerName: "trigger-name",
					BaseURL:     "https://localhost",
				}, sub)
				return nil
			},
		},
	}
	err := svc.SetTrackerRef(context.Background(), "sub-id", "tracker-ref")
	require.NoError(t, err)
}

func TestSubscriptionsManager_Delete(t *testing.T) {
	svc := &SubscriptionsManager{
		Store: &engine.SubscriptionsMock{
			DeleteFunc: func(ctx context.Context, subID string) error {
				assert.Equal(t, "sub-id", subID)
				return nil
			},
		},
	}
	err := svc.Delete(context.Background(), "sub-id")
	require.NoError(t, err)
}

func TestSubscriptionsManager_List(t *testing.T) {
	svc := &SubscriptionsManager{
		Store: &engine.SubscriptionsMock{
			ListFunc: func(ctx context.Context, trackerID string) ([]store.Subscription, error) {
				assert.Equal(t, "tracker-id", trackerID)
				return []store.Subscription{
					{ID: "sub-id-1"},
					{ID: "sub-id-2"},
					{ID: "sub-id-3"},
					{ID: "sub-id-4"},
				}, nil
			},
		},
	}
	subs, err := svc.List(context.Background(), "tracker-id")
	require.NoError(t, err)
	assert.Equal(t, []store.Subscription{
		{ID: "sub-id-1"},
		{ID: "sub-id-2"},
		{ID: "sub-id-3"},
		{ID: "sub-id-4"},
	}, subs)
}

func TestSubscriptionsManager_Listen(t *testing.T) {
	svc := &SubscriptionsManager{
		Store: &engine.SubscriptionsMock{
			GetFunc: func(ctx context.Context, subID string) (store.Subscription, error) {
				assert.Equal(t, "sub-id", subID)
				return store.Subscription{
					ID:          "sub-id",
					TrackerRef:  "tracker-ref",
					TrackerName: "tracker-name",
					TriggerName: "trigger-name",
					BaseURL:     "https://localhost",
				}, nil
			},
		},
		Addr:   ":9099",
		Router: mux.NewRouter(),
	}

	ctx, cancel := context.WithCancel(context.Background())

	started := sign.Signal()
	stopped := sign.Signal()
	go func() {
		started.Done()
		err := svc.Listen(ctx,
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				b, err := io.ReadAll(r.Body)
				require.NoError(t, err)
				assert.Equal(t, []byte(`{}`), b)
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`read`))

				sub, err := store.GetSubscription(r.Context())
				require.NoError(t, err)
				assert.Equal(t, store.Subscription{
					ID:          "sub-id",
					TrackerRef:  "tracker-ref",
					TrackerName: "tracker-name",
					TriggerName: "trigger-name",
					BaseURL:     "https://localhost",
				}, sub)
			}))
		assert.Error(t, context.Canceled, err)
		stopped.Done()
	}()

	require.NoError(t, started.WaitTimeout(5*time.Second))

	resp, err := http.Post("http://localhost:9099/sub-id",
		"application/json", strings.NewReader("{}"))
	require.NoError(t, err)
	b, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Equal(t, []byte(`read`), b)

	cancel()
	require.NoError(t, stopped.WaitTimeout(5*time.Second))
}

package webhook

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/cappuccinotm/dastracker/app/errs"
	"github.com/cappuccinotm/dastracker/app/store"
	"github.com/cappuccinotm/dastracker/app/store/engine"
	"github.com/cappuccinotm/dastracker/pkg/logx"
	"github.com/cappuccinotm/dastracker/pkg/sign"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestManager_Register(t *testing.T) {
	t.Run("tracker is already registered", func(t *testing.T) {
		err := (&Manager{registeredHandlers: []string{"trk1"}}).
			Register("trk1", http.Handler(nil))
		var eTrkRegistered errs.ErrTrackerRegistered
		assert.ErrorAs(t, err, &eTrkRegistered)
		assert.Equal(t, "trk1", string(eTrkRegistered))
	})

	t.Run("successful registration", func(t *testing.T) {
		type body struct {
			SomeStr string `json:"some_str"`
		}
		m := &Manager{registeredHandlers: []string{"trk1"}, r: mux.NewRouter()}

		err := m.Register("trk", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var b body
			assert.NoError(t, json.NewDecoder(r.Body).Decode(&b))
			assert.Equal(t, "blah", b.SomeStr)
			_, err := w.Write([]byte("blahblah"))
			require.NoError(t, err)
			w.WriteHeader(200)
		}))
		assert.NoError(t, err)

		ts := httptest.NewServer(m.r)
		defer ts.Close()

		b, err := json.Marshal(body{SomeStr: "blah"})
		require.NoError(t, err)

		resp, err := ts.Client().Post(ts.URL+"/trk/123", "application/json", bytes.NewReader(b))
		assert.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, 200, resp.StatusCode)
		b, err = io.ReadAll(resp.Body)
		require.NoError(t, err)
		assert.Equal(t, "blahblah", string(b))
	})
}

func TestNewManager(t *testing.T) {
	expectedWh := store.Webhook{
		ID:          "id",
		TrackerRef:  "tracker-id",
		TrackerName: "tracker-name",
		TriggerName: "trigger-name",
		BaseURL:     "base-url",
	}

	eng := &engine.WebhooksMock{
		GetFunc: func(_ context.Context, id string) (store.Webhook, error) {
			assert.Equal(t, "id", id)
			return expectedWh, nil
		},
	}

	called := sign.Signal()

	m := NewManager(
		"blah",
		mux.NewRouter(),
		eng,
		&logx.LoggerMock{
			PrintfFunc: func(f string, args ...interface{}) {
				t.Logf(f, args...)
				assert.FailNow(t, "log must not called")
			},
		},
	)
	m.r.PathPrefix("/{whID}").Methods("GET").
		HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called.Done()
			wh, err := GetWebhook(r.Context())
			require.NoError(t, err)
			require.Equal(t, expectedWh, wh)
			w.WriteHeader(200)
			_, err = w.Write([]byte(`something`))
			require.NoError(t, err)
		})

	ts := httptest.NewServer(m.r)
	defer ts.Close()

	resp, err := ts.Client().Get(ts.URL + "/id")
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
	assert.True(t, called.Signaled())
}

func TestManager_SetTrackerID(t *testing.T) {
	eng := &engine.WebhooksMock{
		GetFunc: func(_ context.Context, s string) (store.Webhook, error) {
			assert.Equal(t, "id", s)
			return store.Webhook{ID: "id"}, nil
		},
		UpdateFunc: func(_ context.Context, wh store.Webhook) error {
			assert.Equal(t, store.Webhook{ID: "id", TrackerRef: "tracker-id"}, wh)
			return nil
		},
	}

	err := (&Manager{store: eng}).SetTrackerID(context.Background(), "id", "tracker-id")
	assert.NoError(t, err)
}

func TestManager_Create(t *testing.T) {
	t.Run("tracker not registered", func(t *testing.T) {
		wh, err := (&Manager{}).Create(context.Background(), "trk1", "trigger")
		var eTrkNotRegistered errs.ErrTrackerNotRegistered
		assert.ErrorAs(t, err, &eTrkNotRegistered)
		assert.Equal(t, "trk1", string(eTrkNotRegistered))
		assert.Empty(t, wh)
	})

	t.Run("success", func(t *testing.T) {
		wh, err := (&Manager{
			store: &engine.WebhooksMock{CreateFunc: func(ctx context.Context, wh store.Webhook) (string, error) {
				assert.Equal(t, store.Webhook{
					TrackerName: "tracker",
					TriggerName: "trigger",
					BaseURL:     "base-url",
				}, wh)
				return "id", nil
			}},
			baseURL:            "base-url",
			registeredHandlers: []string{"tracker"},
		}).Create(context.Background(), "tracker", "trigger")
		assert.NoError(t, err)
		assert.Equal(t, store.Webhook{
			ID:          "id",
			TrackerName: "tracker",
			TriggerName: "trigger",
			BaseURL:     "base-url",
		}, wh)
	})
}

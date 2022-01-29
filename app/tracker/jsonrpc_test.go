package tracker

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/cappuccinotm/dastracker/app/store"
	"github.com/cappuccinotm/dastracker/app/webhook"
	"github.com/cappuccinotm/dastracker/pkg/rpcx"
	"github.com/cappuccinotm/dastracker/pkg/sign"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewJSONRPC(t *testing.T) {
	_, err := NewJSONRPC(rpcx.Client(nil), "jsonrpc", &webhook.InterfaceMock{
		RegisterFunc: func(name string, handler http.Handler) error {
			assert.Equal(t, "jsonrpc", name)
			assert.NotNil(t, handler)
			return nil
		},
	})
	require.NoError(t, err)
}

func TestJSONRPC_Name(t *testing.T) {
	assert.Equal(t, "jsonrpc", (&JSONRPC{name: "jsonrpc"}).Name())
}

func TestJSONRPC_Call(t *testing.T) {
	svc := &JSONRPC{name: "jrpc", cl: &rpcx.ClientMock{
		CallFunc: func(ctx context.Context, serviceMethod string, args, reply interface{}) error {
			resp, ok := reply.(*Response)
			assert.True(t, ok)
			req, ok := args.(Request)
			assert.True(t, ok)

			assert.Equal(t, "jrpc.some-method", serviceMethod)
			*resp = Response{Tracker: "remote-tracker", TaskID: "task-id"}
			assert.Equal(t, Request{
				MethodURI: "some-tracker/some-method",
				Ticket: store.Ticket{
					ID:         "ticket-id",
					TrackerIDs: map[string]string{"tracker": "tracker-id"},
					Content:    store.Content{Body: "body", Title: "title"},
				},
				Vars: store.Vars{},
			}, req)
			return nil
		},
	}}

	resp, err := svc.Call(context.Background(), Request{
		MethodURI: "some-tracker/some-method",
		Ticket: store.Ticket{
			ID:         "ticket-id",
			TrackerIDs: map[string]string{"tracker": "tracker-id"},
			Content:    store.Content{Body: "body", Title: "title"},
		},
		Vars: store.Vars{},
	})
	require.NoError(t, err)
	assert.Equal(t, Response{Tracker: "remote-tracker", TaskID: "task-id"}, resp)
}

func TestJSONRPC_Subscribe(t *testing.T) {
	svc := &JSONRPC{
		name: "jrpc",
		whm: &webhook.InterfaceMock{
			CreateFunc: func(ctx context.Context, tracker string, trigger string) (store.Webhook, error) {
				assert.Equal(t, "jrpc", tracker)
				assert.Equal(t, "trigger", trigger)
				return store.Webhook{
					ID:          "trigger-id",
					TrackerName: "jrpc",
					TriggerName: "trigger",
					BaseURL:     "https://blah.com/webhooks",
				}, nil
			},
		},
		cl: &rpcx.ClientMock{
			CallFunc: func(ctx context.Context, serviceMethod string, args, reply interface{}) error {
				req, ok := args.(SubscribeReq)
				assert.True(t, ok)

				assert.Equal(t, "jrpc.Subscribe", serviceMethod)
				assert.Equal(t, SubscribeReq{
					TriggerName: "trigger",
					Tracker:     "jrpc",
					Vars: map[string]string{
						"blah": "blah",
						"_url": "https://blah.com/webhooks/jrpc/trigger-id",
					},
				}, req)
				return nil
			},
		},
	}

	err := svc.Subscribe(context.Background(), SubscribeReq{
		TriggerName: "trigger",
		Tracker:     "jrpc",
		Vars:        map[string]string{"blah": "blah"},
	})
	require.NoError(t, err)
}

func TestJSONRPC_whHandler(t *testing.T) {
	called := sign.Signal()
	svc := &JSONRPC{
		handler: HandlerFunc(func(ctx context.Context, update store.Update) {
			assert.Equal(t, store.Update{
				URL:          "url",
				TriggerName:  "wh-trigger-name",
				ReceivedFrom: store.Locator{TaskID: "task-id", Tracker: "wh-tracker-name"},
				Content: store.Content{
					Body:   "body",
					Title:  "title",
					Fields: map[string]string{"field": "value"},
				},
			}, update)
			called.Done()
		}),
	}

	ts := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			r = r.WithContext(webhook.PutWebhook(r.Context(), store.Webhook{
				ID:          "wh-id",
				TrackerID:   "wh-tracker-id",
				TrackerName: "wh-tracker-name",
				TriggerName: "wh-trigger-name",
				BaseURL:     "wh-base-url",
			}))
			svc.whHandler(w, r)
		}))
	defer ts.Close()

	b, err := json.Marshal(store.Update{
		URL:          "url",
		ReceivedFrom: store.Locator{TaskID: "task-id"},
		Content: store.Content{
			Body:   "body",
			Title:  "title",
			Fields: map[string]string{"field": "value"},
		},
	})
	require.NoError(t, err)

	resp, err := ts.Client().Post(ts.URL, "application/json", bytes.NewReader(b))
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	assert.True(t, called.Signaled())
}

func TestJSONRPC_Listen(t *testing.T) {
	svc := &JSONRPC{}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	handlerProvided := sign.Signal()

	err := svc.Listen(ctx, HandlerFunc(func(_ context.Context, _ store.Update) { handlerProvided.Done() }))
	assert.Equal(t, context.Canceled, err)
	svc.handler.Handle(context.Background(), store.Update{})

	assert.True(t, handlerProvided.Signaled())
}

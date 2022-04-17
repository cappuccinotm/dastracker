package tracker

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"bytes"
	"encoding/json"
	"github.com/cappuccinotm/dastracker/app/store"
	"github.com/cappuccinotm/dastracker/lib"
	"github.com/cappuccinotm/dastracker/pkg/logx"
	"github.com/cappuccinotm/dastracker/pkg/rpcx"
	"github.com/cappuccinotm/dastracker/pkg/sign"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"time"
)

func TestNewJSONRPC(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer ts.Close()

	u, err := url.Parse(ts.URL)
	require.NoError(t, err)

	_, err = NewJSONRPC("jsonrpc", logx.Nop(), lib.Vars{"address": u.Host})
	require.NoError(t, err)
}

func TestJSONRPC_Name(t *testing.T) {
	assert.Equal(t, "jsonrpc", (&JSONRPC{name: "jsonrpc"}).Name())
}

func TestJSONRPC_Call(t *testing.T) {
	svc := &JSONRPC{name: "jrpc", cl: &rpcx.ClientMock{
		CallFunc: func(ctx context.Context, serviceMethod string, args, reply interface{}) error {
			resp, ok := reply.(*lib.Response)
			assert.True(t, ok)
			req, ok := args.(lib.Request)
			assert.True(t, ok)

			assert.Equal(t, "plugin.some-method", serviceMethod)
			*resp = lib.Response{Task: lib.Task{
				ID:     "task-id",
				URL:    "task-url",
				Title:  "task-title",
				Body:   "task-body",
				Fields: map[string]string{"field": "value"},
			}}
			assert.Equal(t, lib.Request{
				TaskID: "task-id",
				Vars:   lib.Vars{},
			}, req)
			return nil
		},
	}}

	resp, err := svc.Call(context.Background(), Request{
		Method: "some-method",
		TaskID: "task-id",
		Vars:   lib.Vars{},
	})
	require.NoError(t, err)
	assert.Equal(t, Response{Task: store.Task{
		ID: "task-id",
		Content: store.Content{
			Body:   "task-body",
			Title:  "task-title",
			Fields: map[string]string{"field": "value"},
		},
	}}, resp)
}

func TestJSONRPC_Subscribe(t *testing.T) {
	svc := &JSONRPC{
		name: "jrpc",
		cl: &rpcx.ClientMock{
			CallFunc: func(ctx context.Context, serviceMethod string, args, reply interface{}) error {
				req, ok := args.(lib.SubscribeReq)
				assert.True(t, ok)

				assert.Equal(t, "plugin.Subscribe", serviceMethod)
				assert.Equal(t, lib.SubscribeReq{
					WebhookURL: "https://blah.com/webhooks/jrpc/trigger-id",
					Vars:       map[string]string{"blah": "blah"},
				}, req)
				reply.(*lib.SubscribeResp).TrackerRef = "tracker-ref"
				return nil
			},
		},
	}

	resp, err := svc.Subscribe(context.Background(), SubscribeReq{
		WebhookURL: "https://blah.com/webhooks/jrpc/trigger-id",
		Vars:       lib.Vars{"blah": "blah"},
	})
	require.NoError(t, err)
	assert.Equal(t, SubscribeResp{TrackerRef: "tracker-ref"}, resp)
}

func TestJSONRPC_Unsubscribe(t *testing.T) {
	svc := &JSONRPC{
		name: "jrpc",
		cl: &rpcx.ClientMock{
			CallFunc: func(ctx context.Context, serviceMethod string, args, reply interface{}) error {
				req, ok := args.(lib.UnsubscribeReq)
				assert.True(t, ok)

				assert.Equal(t, "plugin.Unsubscribe", serviceMethod)
				assert.Equal(t, lib.UnsubscribeReq{TrackerRef: "tracker-ref"}, req)
				return nil
			},
		},
	}

	err := svc.Unsubscribe(context.Background(), UnsubscribeReq{TrackerRef: "tracker-ref"})
	require.NoError(t, err)
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

func TestJSONRPC_HandleWebhook(t *testing.T) {
	called := sign.Signal()
	svc := &JSONRPC{
		name: "tracker",
		handler: HandlerFunc(func(ctx context.Context, update store.Update) {
			assert.Equal(t, store.Update{
				URL: "update-url",
				ReceivedFrom: store.Locator{
					Tracker: "tracker",
					ID:      "task-id",
				},
				Content: store.Content{
					Body:   "body",
					Title:  "title",
					Fields: map[string]string{"somefield": "somevalue"},
				},
			}, update)
			called.Done()
		}),
	}

	ts := httptest.NewServer(http.HandlerFunc(svc.HandleWebhook))
	defer ts.Close()

	b, err := json.Marshal(lib.Task{
		URL:    "update-url",
		ID:     "task-id",
		Title:  "title",
		Body:   "body",
		Fields: map[string]string{"somefield": "somevalue"},
	})
	require.NoError(t, err)

	resp, err := ts.Client().Post(ts.URL, "application/json", bytes.NewReader(b))
	require.NoError(t, err)
	assert.Equal(t, resp.StatusCode, http.StatusOK)
	require.NoError(t, called.WaitTimeout(5*time.Second))
}

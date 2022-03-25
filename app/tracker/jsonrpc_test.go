package tracker

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/cappuccinotm/dastracker/app/store"
	"github.com/cappuccinotm/dastracker/lib"
	"github.com/cappuccinotm/dastracker/pkg/logx"
	"github.com/cappuccinotm/dastracker/pkg/rpcx"
	"github.com/cappuccinotm/dastracker/pkg/sign"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
			*resp = lib.Response{TaskID: "task-id"}
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
	assert.Equal(t, Response{TaskID: "task-id"}, resp)
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

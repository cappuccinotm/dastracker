package tracker

import (
	"context"
	"fmt"
	"net/http"

	"encoding/json"
	"time"

	"github.com/cappuccinotm/dastracker/app/store"
	"github.com/cappuccinotm/dastracker/lib"
	"github.com/cappuccinotm/dastracker/pkg/logx"
	"github.com/cappuccinotm/dastracker/pkg/rpcx"
	"github.com/go-pkgz/repeater/strategy"
)

// JSONRPC implements Interface in order to allow external services,
// described in the configuration file, extend the functionality of
// the dastracker.
type JSONRPC struct {
	cl      rpcx.Client
	name    string
	l       logx.Logger
	handler Handler
}

// NewJSONRPC makes new instance of JSONRPC.
func NewJSONRPC(name string, l logx.Logger, vars lib.Vars) (*JSONRPC, error) {
	dialer, err := rpcx.NewRedialer(
		rpcx.JSONRPC(),
		&strategy.FixedDelay{Repeats: 3, Delay: time.Second},
		"tcp",
		vars.Get("address"),
	)
	if err != nil {
		return nil, fmt.Errorf("initialize new dialer for %s tracker: %w", name, err)
	}

	svc := &JSONRPC{cl: dialer, name: name, l: l}

	return svc, nil
}

// Name returns the name of the JSONRPC plugin tracker.
func (rpc *JSONRPC) Name() string { return rpc.name }

// Call makes a call to the remote JSONRPC server with given Request.
func (rpc *JSONRPC) Call(ctx context.Context, req Request) (Response, error) {
	var resp lib.Response

	rpcReq := lib.Request{TaskID: req.TaskID, Vars: req.Vars}
	if err := rpc.cl.Call(ctx, "plugin."+req.Method, rpcReq, &resp); err != nil {
		return Response{}, fmt.Errorf("call remote method %s: %w", req.Method, err)
	}

	return Response{TaskID: resp.TaskID}, nil
}

// Subscribe sends subscribe call to the remote JSONRPC server.
func (rpc *JSONRPC) Subscribe(ctx context.Context, req SubscribeReq) (SubscribeResp, error) {
	var resp lib.SubscribeResp
	if err := rpc.cl.Call(ctx, "plugin.Subscribe", lib.SubscribeReq{
		WebhookURL: req.WebhookURL,
		Vars:       req.Vars,
	}, &resp); err != nil {
		return SubscribeResp{}, fmt.Errorf("call remote Subscribe: %w", err)
	}

	return SubscribeResp{TrackerRef: resp.TrackerRef}, nil
}

// Unsubscribe sends unsubscribe call to the remote JSONRPC server.
func (rpc *JSONRPC) Unsubscribe(ctx context.Context, req UnsubscribeReq) error {
	panic("implement me")
}

// HandleWebhook handles webhook call from the remote JSONRPC server.
func (rpc *JSONRPC) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var upd store.Update
	if err := json.NewDecoder(r.Body).Decode(&upd); err != nil {
		rpc.l.Printf("[WARN] failed to decode webhook update: %v", err)
		return
	}

	upd.ReceivedFrom.Tracker = rpc.name

	rpc.handler.Handle(ctx, upd)
	w.WriteHeader(http.StatusOK)
}

// Listen starts updates listener.
func (rpc *JSONRPC) Listen(ctx context.Context, h Handler) error {
	rpc.handler = h
	<-ctx.Done()
	return ctx.Err()
}

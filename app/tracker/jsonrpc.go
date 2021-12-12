package tracker

import (
	"context"
	"fmt"
	"net/http"

	"encoding/json"
	"github.com/cappuccinotm/dastracker/app/store"
	"github.com/cappuccinotm/dastracker/app/webhook"
	"github.com/cappuccinotm/dastracker/pkg/logx"
	"github.com/cappuccinotm/dastracker/pkg/rpcx"
)

// JSONRPC implements Interface in order to allow external services,
// described in the configuration file, extend the functionality of
// the dastracker.
type JSONRPC struct {
	cl      rpcx.Client
	name    string
	l       logx.Logger
	whm     webhook.Interface
	handler Handler
}

// NewJSONRPC makes new instance of JSONRPC.
func NewJSONRPC(cl rpcx.Client, name string, whm webhook.Interface) (*JSONRPC, error) {
	svc := &JSONRPC{cl: cl, name: name, whm: whm}

	if err := whm.Register(name, http.HandlerFunc(svc.whHandler)); err != nil {
		return nil, fmt.Errorf("register webhooks handler: %w", err)
	}

	return svc, nil
}

// Name returns the name of the JSONRPC plugin tracker.
func (rpc *JSONRPC) Name() string { return rpc.name }

// Call makes a call to the remote JSONRPC server with given Request.
func (rpc *JSONRPC) Call(ctx context.Context, req Request) (Response, error) {
	var resp Response
	if err := rpc.cl.Call(ctx, fmt.Sprintf("%s.%s", rpc.name, req.Method), req, &resp); err != nil {
		return Response{}, fmt.Errorf("call remote method %s: %w", req.Method, err)
	}
	return resp, nil
}

// Subscribe sends subscribe call to the remote JSONRPC server.
func (rpc *JSONRPC) Subscribe(ctx context.Context, req SubscribeReq) error {
	wh, err := rpc.whm.Create(ctx, rpc.name, req.TriggerName)
	if err != nil {
		return fmt.Errorf("create webhook: %w", err)
	}

	url, err := wh.URL()
	if err != nil {
		return fmt.Errorf("make url from webhook %q: %w", wh.ID, err)
	}

	req.Vars.Set("_url", url)

	var resp struct{}
	if err := rpc.cl.Call(ctx, fmt.Sprintf("%s.Subscribe", rpc.name), req, &resp); err != nil {
		return fmt.Errorf("call remote Subscribe: %w", err)
	}

	return nil
}

func (rpc *JSONRPC) whHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	wh, err := webhook.GetWebhook(ctx)
	if err != nil {
		rpc.l.Printf("[WARN] failed to get webhook information from request: %v", err)
		return
	}

	var upd store.Update
	if err := json.NewDecoder(r.Body).Decode(&upd); err != nil {
		rpc.l.Printf("[WARN] failed to decode webhook update: %v", err)
		return
	}

	upd.ReceivedFrom.Tracker = wh.TrackerName
	upd.TriggerName = wh.TriggerName

	rpc.handler.Handle(ctx, upd)
	w.WriteHeader(http.StatusOK)
}

// Listen starts updates listener.
func (rpc *JSONRPC) Listen(ctx context.Context, h Handler) error {
	rpc.handler = h
	<-ctx.Done()
	return ctx.Err()
}

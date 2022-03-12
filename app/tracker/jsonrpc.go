package tracker

import (
	"context"
	"fmt"
	"net/http"

	"encoding/json"
	"time"

	"github.com/cappuccinotm/dastracker/app/store"
	"github.com/cappuccinotm/dastracker/app/webhook"
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
	whm     webhook.Interface
	handler Handler
}

// NewJSONRPC makes new instance of JSONRPC.
func NewJSONRPC(name string, l logx.Logger, whm webhook.Interface, vars lib.Vars) (*JSONRPC, error) {
	dialer, err := rpcx.NewRedialer(
		rpcx.JSONRPC(),
		&strategy.FixedDelay{Repeats: 3, Delay: time.Second},
		"tcp",
		vars.Get("address"),
	)
	if err != nil {
		return nil, fmt.Errorf("initialize new dialer for %s tracker: %w", name, err)
	}

	svc := &JSONRPC{cl: dialer, name: name, whm: whm, l: l}

	if err := whm.Register(name, http.HandlerFunc(svc.whHandler)); err != nil {
		return nil, fmt.Errorf("register webhooks handler: %w", err)
	}

	return svc, nil
}

// Name returns the name of the JSONRPC plugin tracker.
func (rpc *JSONRPC) Name() string { return rpc.name }

// Call makes a call to the remote JSONRPC server with given Request.
func (rpc *JSONRPC) Call(ctx context.Context, req Request) (Response, error) {
	var resp lib.Response
	if err := rpc.cl.Call(ctx, "plugin."+req.Method, rpc.transformRPCRequest(req), &resp); err != nil {
		return Response{}, fmt.Errorf("call remote method %s: %w", req.Method, err)
	}
	return rpc.transformRPCResponse(resp), nil
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

	req.Vars.Set(lib.URLKey, url)

	var resp struct{}
	if err := rpc.cl.Call(ctx, "plugin.Subscribe", req, &resp); err != nil {
		return fmt.Errorf("call remote Subscribe: %w", err)
	}

	return nil
}

// Unsubscribe sends unsubscribe call to the remote JSONRPC server.
func (rpc *JSONRPC) Unsubscribe(ctx context.Context, req SubscribeReq) error {
	panic("unimplemented")
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

func (rpc *JSONRPC) transformRPCRequest(req Request) lib.Request {
	return lib.Request{
		Ticket: lib.Ticket{
			ID:     req.Ticket.ID,
			TaskID: req.Ticket.TrackerIDs.Get(rpc.name),
			Title:  req.Ticket.Title,
			Body:   req.Ticket.Body,
			Fields: req.Ticket.Fields,
		},
		Vars: req.Vars,
	}
}

func (rpc *JSONRPC) transformRPCResponse(resp lib.Response) Response {
	return Response{TaskID: resp.TaskID}
}

package tracker

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/cappuccinotm/dastracker/app/store"
	"github.com/cappuccinotm/dastracker/lib"
)

// RPC implements Interface and delegates all calls
// to the remote JRPC server.
type RPC struct {
	address string
	cl      RPCClient

	log     *log.Logger
	Webhook WebhookProps
	Tracker Props
}

// NewRPC makes new instance of RPC.
func NewRPC(vars lib.Vars, dl RPCDialer) (*RPC, error) {
	res := &RPC{}
	var ok bool
	var err error

	if res.address, ok = vars["address"]; !ok {
		return nil, ErrInvalidConf("rpc serever address is not present")
	}

	res.cl, err = dl.Dial("tcp", res.address)
	if err != nil {
		return nil, fmt.Errorf("dial %s: %w", res.address, err)
	}

	return res, nil
}

// Close does no-op
func (rc *RPC) Close(_ context.Context) error { return nil }

func (rc *RPC) Call(_ context.Context, call lib.Request) (lib.Response, error) {
	resp := lib.Response{}
	if err := rc.cl.Call(call.Method, call, &resp); err != nil {
		return lib.Response{}, fmt.Errorf("call rpc method %s: %w", call.Method, err)
	}
	return resp, nil
}

func (rc *RPC) SetUpTrigger(_ context.Context, vars lib.Vars, cb Callback) error {
	url := rc.Webhook.newWebHook(func(w http.ResponseWriter, r *http.Request) {
		upd := store.Update{}

		if err := json.NewDecoder(r.Body).Decode(&upd); err != nil {
			rc.log.Printf("[WARN] failed to decode webhook for rpc/%s: %v", rc.Tracker.Name, err)
			return
		}

		if err := cb.Do(r.Context(), upd); err != nil {
			rc.log.Printf("[WARN] callback returned error for rpc/%s: %v", rc.Tracker.Name, err)
			return
		}
	})

	var resp lib.SetUpTriggerResp

	err := rc.cl.Call("set_up_trigger", lib.SetUpTriggerReq{URL: url, Vars: vars}, &resp)
	if err != nil {
		return fmt.Errorf("call set_up_trigger: %w", err)
	}

	return nil
}

// RPCDialer is a maker interface dialing to rpc server and returning new RPCClient
type RPCDialer interface {
	Dial(network, address string) (RPCClient, error)
}

// RPCDialerFunc is an adapter to allow the use of an ordinary functions as the RPCDialer.
type RPCDialerFunc func(network, address string) (RPCClient, error)

// Dial rpc server.
func (f RPCDialerFunc) Dial(network, address string) (RPCClient, error) { return f(network, address) }

// RPCClient defines interface for remote calls
type RPCClient interface {
	Call(serviceMethod string, args interface{}, reply interface{}) error
}

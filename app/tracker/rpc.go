package tracker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/rpc"
	"time"

	"github.com/cappuccinotm/dastracker/app/store"
	"github.com/cappuccinotm/dastracker/lib"
)

// RPC implements Interface and delegates all calls
// to the remote JRPC server.
type RPC struct {
	address string
	cl      RPCClient

	RPCParams
}

// RPCParams describes parameters needed to initialize RPC client.
type RPCParams struct {
	Dialer  RPCDialer
	Logger  *log.Logger
	Webhook WebhookProps
	Tracker Props
}

// NewRPC makes new instance of RPC.
func NewRPC(params RPCParams) (*RPC, error) {
	res := &RPC{RPCParams: params}
	var ok bool
	var err error

	if res.address, ok = params.Tracker.Variables["address"]; !ok {
		return nil, ErrInvalidConf("rpc serever address is not present")
	}

	res.cl, err = redialer(params.Dialer, "tcp", res.address, redialFixedDelayStrategy{
		MaxConnTry: 3,
		ReconDelay: 3 * time.Second,
	})
	if err != nil {
		return nil, fmt.Errorf("dial %s: %w", res.address, err)
	}

	log.Printf("[INFO] initialized RPC client with params %+v", params)

	return res, nil
}

// Close does no-op
func (rc *RPC) Close(_ context.Context) error { return nil }

func (rc *RPC) Call(_ context.Context, call lib.Request) (lib.Response, error) {
	resp := lib.Response{}
	if err := rc.cl.Call(rc.Tracker.Name+"."+call.Method, call, &resp); err != nil {
		return lib.Response{}, fmt.Errorf("call rpc method %s: %w", call.Method, err)
	}
	return resp, nil
}

func (rc *RPC) SetUpTrigger(_ context.Context, vars lib.Vars, cb Callback) error {
	url := rc.Webhook.newWebHook(func(w http.ResponseWriter, r *http.Request) {
		upd := store.Update{}

		if err := json.NewDecoder(r.Body).Decode(&upd); err != nil {
			rc.Logger.Printf("[WARN] failed to decode webhook for rpc/%s: %v", rc.Tracker.Name, err)
			return
		}

		if err := cb.Do(r.Context(), upd); err != nil {
			rc.Logger.Printf("[WARN] callback returned error for rpc/%s: %v", rc.Tracker.Name, err)
			return
		}
	})

	var resp lib.SetUpTriggerResp

	err := rc.cl.Call(rc.Tracker.Name+".SetUpTriggerCall", lib.SetUpTriggerReq{URL: url, Vars: vars}, &resp)
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

type redialFixedDelayStrategy struct {
	MaxConnTry int
	ReconDelay time.Duration
}

// redialer dials a new connection in case if RPCClient.Call returns an error
func redialer(dial RPCDialer, network, addr string, strategy redialFixedDelayStrategy) (RPCClient, error) {
	cl, err := dial.Dial(network, addr)
	if err != nil {
		return nil, fmt.Errorf("dial: %w", err)
	}

	return RPCClientFunc(func(serviceMethod string, args interface{}, reply interface{}) error {
		if err = cl.Call(serviceMethod, args, reply); !errors.Is(err, rpc.ErrShutdown) {
			return err
		}

		for reconnTry := 0; reconnTry < strategy.MaxConnTry; reconnTry++ {
			time.Sleep(strategy.ReconDelay)
			if cl, err = dial.Dial(network, addr); err == nil {
				return cl.Call(serviceMethod, args, reply)
			}
		}

		return err
	}), nil
}

// RPCClient defines interface for remote calls
type RPCClient interface {
	Call(serviceMethod string, args interface{}, reply interface{}) error
}

// RPCClientFunc is an adapter to allow to use ordinary functions as the RPCClient.
type RPCClientFunc func(serviceMethod string, args interface{}, reply interface{}) error

// Call implements RPCClient.
func (f RPCClientFunc) Call(serviceMethod string, args interface{}, reply interface{}) error {
	return f(serviceMethod, args, reply)
}

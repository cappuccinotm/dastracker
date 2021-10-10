package tracker

import (
	"context"
	"fmt"
)

// RPC implements Interface and delegates all calls
// to the remote JRPC server.
type RPC struct {
	address string
	cl      RPCClient
}

// NewRPC makes new instance of RPC.
func NewRPC(vars Vars, dl RPCDialer) (*RPC, error) {
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
func (r *RPC) Close(_ context.Context) error {}

func (r *RPC) Call(ctx context.Context, call Request) (Response, error) {
	resp := Response{}
	if err := r.cl.Call(call.Method, call, &resp); err != nil {
		return Response{}, fmt.Errorf("call rpc method %s: %w", call.Method, err)
	}
	return resp, nil
}

func (r *RPC) SetUpTrigger(ctx context.Context, vars Vars, cb Callback) error {
	panic("not implemented")
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

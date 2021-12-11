package rpcx

import (
	"context"
	"net/rpc/jsonrpc"
)

// Dialer is a maker interface dialing to rpc server and returning new Client
type Dialer interface {
	Dial(network, address string) (RPCClient, error)
}

// DialerFunc is an adapter to use ordinary functions as Dialer.
type DialerFunc func(network, address string) (RPCClient, error)

// Dial calls the wrapped function.
func (f DialerFunc) Dial(network, addr string) (RPCClient, error) { return f(network, addr) }

// Client defines interface for a client to make contextual remote calls.
type Client interface {
	Call(ctx context.Context, serviceMethod string, args interface{}, reply interface{}) error
	Close() error
}

// RPCClient describes a pure standard library rpc client.
type RPCClient interface {
	Call(serviceMethod string, args interface{}, reply interface{}) error
	Close() error
}

// JSONRPC returns the jsonrpc.Dial dialer.
func JSONRPC() Dialer {
	return DialerFunc(func(network, addr string) (RPCClient, error) {
		return jsonrpc.Dial(network, addr)
	})
}

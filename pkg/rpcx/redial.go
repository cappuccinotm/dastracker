package rpcx

import (
	"context"
	"errors"
	"fmt"
	"github.com/cappuccinotm/dastracker/pkg/logx"
	"github.com/cappuccinotm/dastracker/pkg/repeaterx"
	"github.com/go-pkgz/repeater/strategy"
	"net/rpc"
)

// Redialer redials the RPC server with the provided repeating
// strategy, in case if Call has been failed.
type Redialer struct {
	dialer   Dialer
	repeater *repeaterx.AllowedErrors
	network  string
	addr     string
	cl       RPCClient
	log      logx.Logger
}

// NewRedialer makes new instance of Redialer.
func NewRedialer(dialer Dialer, strtg strategy.Interface, network, addr string) (*Redialer, error) {
	svc := &Redialer{
		dialer:   dialer,
		repeater: repeaterx.NewAllowedErrors(strtg),
		network:  network,
		addr:     addr,
	}

	err := svc.repeater.Do(context.Background(), func() error {
		cl, err := svc.dialer.Dial(svc.network, svc.addr)
		if err != nil {
			return err
		}
		svc.cl = cl
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("dial: %w", err)
	}

	return svc, nil
}

// Call makes a call to the remote RPC server,
// tries to redial if the RPC connection is shut down.
func (r *Redialer) Call(ctx context.Context, serviceMethod string, args interface{}, reply interface{}) error {
	err := r.repeater.Do(ctx, func() error {
		switch err := r.cl.Call(serviceMethod, args, reply); {
		case errors.Is(err, rpc.ErrShutdown):
			if r.cl, err = r.dialer.Dial(r.network, r.addr); err != nil {
				r.log.Printf("[WARN] failed to dial %s://%s: %v", r.network, r.addr, err)
				return rpc.ErrShutdown
			}
		case err != nil:
			return err
		}
		return nil
	}, rpc.ErrShutdown)
	return err
}

// Close proxies the Close call to the embedded rpc client.
func (r *Redialer) Close() error {
	return r.cl.Close()
}

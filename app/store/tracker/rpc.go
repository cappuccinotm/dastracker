package tracker

import "context"

// RPC implements Interface via RPC-provided plugins.
type RPC struct {
}

func (r *RPC) Call(ctx context.Context, call Request) error {
	panic("implement me")
}

func (r *RPC) SetUpTrigger(ctx context.Context, vars Vars, cb Callback) error {
	panic("implement me")
}

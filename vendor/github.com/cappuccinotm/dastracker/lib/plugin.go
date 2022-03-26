package lib

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/rpc"
	"net/rpc/jsonrpc"
)

// Plugin provides methods to start listener and register RPC methods.
type Plugin struct {
	Address            string
	SubscribeHandler   func(req SubscribeReq) (SubscribeResp, error)
	UnsubscribeHandler func(req UnsubscribeReq) error
}

// Listen starts RPC server and listens for incoming connections.
func (p *Plugin) Listen(ctx context.Context, rcvr interface{}) (err error) {
	ctxCancel, cancel := context.WithCancel(ctx)
	defer cancel()

	if err = rpc.RegisterName("plugin", rcvr); err != nil {
		return fmt.Errorf("can't register plugin: %v", err)
	}
	log.Printf("[INFO] register rpc on address %s", p.Address)

	return p.listen(ctxCancel)
}

// Subscribe implements Subscribe RPC handler and calls the handler for
// subscription, if it is set.
func (p *Plugin) Subscribe(req SubscribeReq, resp *SubscribeResp) error {
	if p.SubscribeHandler == nil {
		return nil
	}

	mtdResp, err := p.SubscribeHandler(req)
	resp = &mtdResp
	return err
}

// Unsubscribe implements Unsubscribe RPC handler and calls the handler for
// subscription, if it is set.
func (p *Plugin) Unsubscribe(req UnsubscribeReq, _ *struct{}) error {
	if p.UnsubscribeHandler == nil {
		return nil
	}

	return p.UnsubscribeHandler(req)
}

func (p *Plugin) listen(ctx context.Context) error {
	listener, err := net.Listen("tcp", p.Address)
	if err != nil {
		return fmt.Errorf("can't listen on %s: %v", p.Address, err)
	}

	go func() {
		<-ctx.Done()
		if err := listener.Close(); err != nil {
			log.Printf("[WARN] can't close plugin listener")
		}
	}()

	for {
		conn, err := listener.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
				return fmt.Errorf("accept failed for: %v", err)
			}
		}
		go func() {
			log.Printf("[INFO] accepted connection from %s", conn.RemoteAddr().String())
			jsonrpc.ServeConn(conn)
			log.Printf("[INFO] stopped to serve connection from %s", conn.RemoteAddr().String())
		}()
	}
}

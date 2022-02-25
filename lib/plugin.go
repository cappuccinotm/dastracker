package lib

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/rpc"
)

// Plugin provides methods to start listener and register RPC methods.
type Plugin struct {
	Name             string
	Address          string
	SubscribeHandler func(req SubscribeReq) error
}

// Listen starts RPC server and listens for incoming connections.
func (p *Plugin) Listen(ctx context.Context, rcvr interface{}) (err error) {
	ctxCancel, cancel := context.WithCancel(ctx)
	defer cancel()

	if err = rpc.RegisterName(p.Name, rcvr); err != nil {
		return fmt.Errorf("can't register plugin %s: %v", p.Name, err)
	}
	log.Printf("[INFO] register rpc %s:%s", p.Name, p.Address)

	return p.listen(ctxCancel)
}

// Subscribe implements Subscribe RPC handler and calls the handler for
// subscription, if it is set.
func (p *Plugin) Subscribe(req SubscribeReq, _ *struct{}) error {
	if p.SubscribeHandler == nil {
		return nil
	}

	return p.SubscribeHandler(req)
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
		log.Printf("[DEBUG] plugin listener for %s:%s activated", p.Name, p.Address)
		conn, err := listener.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
				return fmt.Errorf("accept failed for %s: %v", p.Name, err)
			}
		}
		go rpc.ServeConn(conn)
	}
}

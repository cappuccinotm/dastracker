package lib

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/rpc"
	"net/rpc/jsonrpc"
)

// Plugin provides methods to start listener and register RPC methods.
type Plugin struct {
	Address string
	Logger  Logger // no-op by default
}

// Listen starts RPC server and listens for incoming connections.
func (p *Plugin) Listen(ctx context.Context, rcvr interface{}) (err error) {
	ctxCancel, cancel := context.WithCancel(ctx)
	defer cancel()

	if p.Logger == nil {
		p.Logger = log.New(ioutil.Discard, "", 0)
	}

	if err = rpc.RegisterName("plugin", rcvr); err != nil {
		return fmt.Errorf("can't register plugin: %v", err)
	}
	p.Logger.Printf("[INFO] register rpc on address %s", p.Address)

	return p.listen(ctxCancel)
}

func (p *Plugin) listen(ctx context.Context) error {
	listener, err := net.Listen("tcp", p.Address)
	if err != nil {
		return fmt.Errorf("can't listen on %s: %v", p.Address, err)
	}

	go func() {
		<-ctx.Done()
		if err := listener.Close(); err != nil {
			p.Logger.Printf("[WARN] can't close plugin listener")
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
			p.Logger.Printf("[INFO] accepted connection from %s", conn.RemoteAddr().String())
			jsonrpc.ServeConn(conn)
			p.Logger.Printf("[INFO] stopped to serve connection from %s", conn.RemoteAddr().String())
		}()
	}
}

// Logger is a logger interface.
type Logger interface {
	Printf(format string, v ...interface{})
}

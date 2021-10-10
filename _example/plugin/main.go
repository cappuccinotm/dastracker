package main

import (
	"context"
	"log"

	"github.com/cappuccinotm/dastracker/lib"
)

type Handler struct{}

func (h Handler) Print(req lib.Request, resp *lib.Response) error {
	msg := req.Vars.Get("message")
	log.Printf("Received Print call with msg: %s", msg)
	return nil
}

func main() {
	pl := lib.Plugin{Name: "customrpc", Address: ":9000"}
	if err := pl.Listen(context.Background(), &Handler{}); err != nil {
		log.Printf("[WARN] listener stopped, reason: %v", err)
	}
}

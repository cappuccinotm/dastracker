package main

import (
	"context"
	"log"

	"github.com/cappuccinotm/dastracker/lib"
)

type Handler struct{}

func (Handler) Subscribe(req lib.SubscribeReq, _ *lib.SubscribeResp) error {
	log.Printf("Subscribe called: %+v", req)
	return nil
}

func (Handler) Unsubscribe(req lib.UnsubscribeReq, _ *struct{}) error {
	log.Printf("Unsubscribe called: %+v", req)
	return nil
}

func (Handler) Print(req lib.Request, _ *lib.Response) error {
	msg := req.Vars.Get("message")
	log.Printf("Received Print call with msg: %s", msg)
	return nil
}

func main() {
	pl := lib.Plugin{
		Address: ":9000",
		Logger:  log.Default(),
	}
	if err := pl.Listen(context.Background(), Handler{}); err != nil {
		log.Printf("[WARN] listener stopped, reason: %v", err)
	}
}

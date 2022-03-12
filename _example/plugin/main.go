package main

import (
	"context"
	"log"

	"github.com/cappuccinotm/dastracker/lib"
)

type Handler struct{}

func (h *Handler) Print(req lib.Request, _ *lib.Response) error {
	msg := req.Vars.Get("message")
	log.Printf("Received Print call with msg: %s", msg)
	return nil
}

func main() {
	pl := lib.Plugin{
		Address: ":9000",
		SubscribeHandler: func(req lib.SubscribeReq) error {
			log.Printf("[INFO] requested subscription with webhook on %s", req.WebhookURL())
			return nil
		},
	}
	if err := pl.Listen(context.Background(), &Handler{}); err != nil {
		log.Printf("[WARN] listener stopped, reason: %v", err)
	}
}

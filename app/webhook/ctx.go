package webhook

import (
	"context"
	"github.com/cappuccinotm/dastracker/app/store"
)

type whKey struct{}

// GetWebhook extracts a webhook from the given context.
func GetWebhook(ctx context.Context) (store.Webhook, error) {
	i := ctx.Value(whKey{})
	if i == nil {
		return store.Webhook{}, ErrNoWebhook
	}

	if wh, ok := i.(store.Webhook); ok {
		return wh, nil
	}
	return store.Webhook{}, ErrNoWebhook
}

// PutWebhook puts the provided webhook information to the given context.
// Should not be used out of tests outside of webhook package.
func PutWebhook(ctx context.Context, wh store.Webhook) context.Context {
	return context.WithValue(ctx, whKey{}, wh)
}

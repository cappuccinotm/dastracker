package store

import (
	"context"
	"errors"
)

type subKey struct{}

// GetSubscription extracts a subscription from the given context.
func GetSubscription(ctx context.Context) (Subscription, error) {
	if ctx != nil {
		if i := ctx.Value(subKey{}); i != nil {
			if sub, ok := i.(Subscription); ok {
				return sub, nil
			}
		}
	}

	return Subscription{}, ErrNoSubscription
}

// PutSubscription puts the provided subscription information to the given context.
// Should not be used out of tests outside of subscription package.
func PutSubscription(ctx context.Context, wh Subscription) context.Context {
	return context.WithValue(ctx, subKey{}, wh)
}

// ErrNoSubscription indicates that the subscription in the provided context was not found.
var ErrNoSubscription = errors.New("no subscription in the provided context")

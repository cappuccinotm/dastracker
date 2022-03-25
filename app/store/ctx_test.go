package store

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestGetSubscription(t *testing.T) {
	t.Run("basic case", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), subKey{}, Subscription{ID: "sub-id"})
		sub, err := GetSubscription(ctx)
		require.NoError(t, err)
		assert.Equal(t, Subscription{ID: "sub-id"}, sub)
	})
	t.Run("context value is not subscription", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), subKey{}, []int{})
		sub, err := GetSubscription(ctx)
		assert.Empty(t, sub)
		assert.ErrorIs(t, err, ErrNoSubscription)
	})
	t.Run("context nil", func(t *testing.T) {
		sub, err := GetSubscription(nil)
		assert.Empty(t, sub)
		assert.ErrorIs(t, err, ErrNoSubscription)
	})
	t.Run("context doesn't contain sub", func(t *testing.T) {
		sub, err := GetSubscription(context.Background())
		assert.Empty(t, sub)
		assert.ErrorIs(t, err, ErrNoSubscription)
	})
}

func TestPutSubscription(t *testing.T) {
	ctx := PutSubscription(context.Background(), Subscription{ID: "sub-id"})
	val, ok := ctx.Value(subKey{}).(Subscription)
	require.True(t, ok)
	assert.Equal(t, Subscription{ID: "sub-id"}, val)
}

package store

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestSubscription_URL(t *testing.T) {
	assert.Equal(t, "https://localhost/wh-id", Subscription{BaseURL: "https://localhost", ID: "wh-id"}.URL())
}

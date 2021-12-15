//go:build !race
// +build !race

// bolt itself thread-safe so there is no need in race-detector
package bolt

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/cappuccinotm/dastracker/app/store"
	"github.com/cappuccinotm/dastracker/pkg/logx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	bolt "go.etcd.io/bbolt"
	"io/ioutil"
	"os"
	"path"
	"testing"
	"time"
)

func TestWebhook_Create(t *testing.T) {
	svc := prepareWebhook(t)
	id, err := svc.Create(context.Background(), store.Webhook{
		TrackerID:   "tracker-id",
		TrackerName: "tracker-name",
		TriggerName: "trigger-name",
		BaseURL:     "base-url",
	})
	require.NoError(t, err)
	assert.NotEmpty(t, id)

	err = svc.db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket([]byte(webhooksBktName))
		whBts := bkt.Get([]byte(id))
		var wh store.Webhook
		assert.NoError(t, json.Unmarshal(whBts, &wh))
		assert.Equal(t, store.Webhook{
			ID:          id,
			TrackerID:   "tracker-id",
			TrackerName: "tracker-name",
			TriggerName: "trigger-name",
			BaseURL:     "base-url",
		}, wh)

		bkt = tx.Bucket([]byte(trackerToWhRefsBktName)).Bucket([]byte("tracker-id"))
		assert.NotNil(t, bkt)

		whBts = bkt.Get([]byte(id))
		assert.NotEmpty(t, whBts)

		return nil
	})
	require.NoError(t, err)
}

func TestWebhook_Update(t *testing.T) {
	svc := prepareWebhook(t)
	err := svc.Update(context.Background(), store.Webhook{
		ID:          "id",
		TrackerID:   "tracker-id",
		TrackerName: "tracker-name",
		TriggerName: "trigger-name",
		BaseURL:     "base-url",
	})
	require.NoError(t, err)

	err = svc.db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket([]byte(webhooksBktName))
		whBts := bkt.Get([]byte("id"))
		var wh store.Webhook
		assert.NoError(t, json.Unmarshal(whBts, &wh))
		assert.Equal(t, store.Webhook{
			ID:          "id",
			TrackerID:   "tracker-id",
			TrackerName: "tracker-name",
			TriggerName: "trigger-name",
			BaseURL:     "base-url",
		}, wh)

		bkt = tx.Bucket([]byte(trackerToWhRefsBktName)).Bucket([]byte("tracker-id"))
		assert.NotNil(t, bkt)

		whBts = bkt.Get([]byte("id"))
		assert.NotEmpty(t, whBts)

		return nil
	})
	require.NoError(t, err)
}

func TestNewWebhook(t *testing.T) {
	svc := prepareWebhook(t)
	err := svc.db.View(func(tx *bolt.Tx) error {
		assert.NotNil(t, tx.Bucket([]byte(webhooksBktName)))
		assert.NotNil(t, tx.Bucket([]byte(trackerToWhRefsBktName)))
		return nil
	})
	require.NoError(t, err)
}

func TestWebhook_List(t *testing.T) {
	svc := prepareWebhook(t)

	generateWebhooks := func(amount int) (res []store.Webhook) {
		for i := 1; i <= amount; i++ {
			res = append(res, store.Webhook{
				ID:          fmt.Sprintf("%d-id", i),
				TrackerID:   fmt.Sprintf("%d-tracker-id", i),
				TrackerName: "tracker-name",
				TriggerName: fmt.Sprintf("%d-trigger-name", i),
				BaseURL:     fmt.Sprintf("%d-base-url", i),
			})
		}
		return res
	}

	err := svc.db.Update(func(tx *bolt.Tx) error {
		whs := generateWebhooks(5)
		bkt := tx.Bucket([]byte(webhooksBktName))
		refsBkt, err := tx.Bucket([]byte(trackerToWhRefsBktName)).CreateBucketIfNotExists([]byte("tracker-name"))
		require.NoError(t, err)

		for _, wh := range whs {
			bts, err := json.Marshal(wh)
			require.NoError(t, err)

			err = bkt.Put([]byte(wh.ID), bts)
			require.NoError(t, err)

			err = refsBkt.Put([]byte(wh.ID), []byte(time.Now().Format(time.RFC3339Nano)))
			require.NoError(t, err)
		}
		return nil
	})
	require.NoError(t, err)

	whs, err := svc.List(context.Background(), "tracker-name")
	require.NoError(t, err)

	assert.ElementsMatch(t, generateWebhooks(5), whs)
}

func TestWebhook_Delete(t *testing.T) {
	svc := prepareWebhook(t)

	err := svc.db.Update(func(tx *bolt.Tx) error {
		bts, err := json.Marshal(store.Webhook{
			ID:          "id",
			TrackerID:   "tracker-id",
			TrackerName: "tracker-name",
			TriggerName: "trigger-name",
			BaseURL:     "base-url",
		})
		require.NoError(t, err)
		err = tx.Bucket([]byte(webhooksBktName)).Put([]byte("id"), bts)
		require.NoError(t, err)

		bkt, err := tx.Bucket([]byte(trackerToWhRefsBktName)).CreateBucketIfNotExists([]byte("tracker-name"))
		require.NoError(t, err)
		err = bkt.Put([]byte("id"), []byte("2006-01-02T15:04:05.999999999Z07:00"))
		require.NoError(t, err)
		return nil
	})
	require.NoError(t, err)

	err = svc.Delete(context.Background(), "id")
	require.NoError(t, err)

	err = svc.db.View(func(tx *bolt.Tx) error {
		assert.Nil(t, tx.Bucket([]byte(webhooksBktName)).Get([]byte("id")))
		assert.Nil(t, tx.Bucket([]byte(trackerToWhRefsBktName)).
			Bucket([]byte("tracker-name")).
			Get([]byte("id")),
		)
		return nil
	})
	require.NoError(t, err)
}

func prepareWebhook(t *testing.T) *Webhooks {
	loc, err := ioutil.TempDir("", "test_dastracker")
	require.NoError(t, err, "failed to make temp dir")

	svc, err := NewWebhook(path.Join(loc, "dastracker_webhooks_test.db"), bolt.Options{}, logx.NopLogger())
	require.NoError(t, err)

	t.Cleanup(func() { assert.NoError(t, os.RemoveAll(loc)) })

	return svc
}

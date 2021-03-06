//go:build !race
// +build !race

// bolt itself thread-safe so there is no need in race-detector
package bolt

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"testing"
	"time"

	"github.com/cappuccinotm/dastracker/app/store"
	"github.com/cappuccinotm/dastracker/pkg/logx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	bolt "go.etcd.io/bbolt"
)

func TestSubscription_Create(t *testing.T) {
	svc := prepareSubscription(t)
	id, err := svc.Create(context.Background(), store.Subscription{
		TrackerRef:  "tracker-id",
		TrackerName: "tracker-name",
		TriggerName: "trigger-name",
		BaseURL:     "base-url",
	})
	require.NoError(t, err)
	assert.NotEmpty(t, id)

	err = svc.db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket([]byte(subscriptionsBktName))
		subBts := bkt.Get([]byte(id))
		var sub store.Subscription
		assert.NoError(t, json.Unmarshal(subBts, &sub))
		assert.Equal(t, store.Subscription{
			ID:          id,
			TrackerRef:  "tracker-id",
			TrackerName: "tracker-name",
			TriggerName: "trigger-name",
			BaseURL:     "base-url",
		}, sub)

		bkt = tx.Bucket([]byte(trackerToSubsBktName)).Bucket([]byte("tracker-name"))
		assert.NotNil(t, bkt)

		subBts = bkt.Get([]byte(id))
		assert.NotEmpty(t, subBts)

		b := tx.Bucket([]byte(trackerTriggerRefToSubsBktName)).Get([]byte("tracker-name:trigger-name"))
		assert.Equal(t, []byte(id), b)

		return nil
	})
	require.NoError(t, err)
}

func TestSubscription_Update(t *testing.T) {
	svc := prepareSubscription(t)
	err := svc.Update(context.Background(), store.Subscription{
		ID:          "id",
		TrackerRef:  "tracker-id",
		TrackerName: "tracker-name",
		TriggerName: "trigger-name",
		BaseURL:     "base-url",
	})
	require.NoError(t, err)

	err = svc.db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket([]byte(subscriptionsBktName))
		subBts := bkt.Get([]byte("id"))
		var sub store.Subscription
		assert.NoError(t, json.Unmarshal(subBts, &sub))
		assert.Equal(t, store.Subscription{
			ID:          "id",
			TrackerRef:  "tracker-id",
			TrackerName: "tracker-name",
			TriggerName: "trigger-name",
			BaseURL:     "base-url",
		}, sub)

		bkt = tx.Bucket([]byte(trackerToSubsBktName)).Bucket([]byte("tracker-name"))
		assert.NotNil(t, bkt)

		subBts = bkt.Get([]byte("id"))
		assert.NotEmpty(t, subBts)

		b := tx.Bucket([]byte(trackerTriggerRefToSubsBktName)).Get([]byte("tracker-name:trigger-name"))
		assert.Equal(t, []byte("id"), b)

		return nil
	})
	require.NoError(t, err)
}

func TestNewSubscription(t *testing.T) {
	svc := prepareSubscription(t)
	err := svc.db.View(func(tx *bolt.Tx) error {
		assert.NotNil(t, tx.Bucket([]byte(subscriptionsBktName)))
		assert.NotNil(t, tx.Bucket([]byte(trackerToSubsBktName)))
		return nil
	})
	require.NoError(t, err)
}

func TestSubscription_List(t *testing.T) {
	t.Run("subscriptions of particular tracker", func(t *testing.T) {
		svc := prepareSubscription(t)

		generateSubscriptions := func(amount int) (res []store.Subscription) {
			for i := 1; i <= amount; i++ {
				res = append(res, store.Subscription{
					ID:          fmt.Sprintf("%d-id", i),
					TrackerRef:  fmt.Sprintf("%d-tracker-id", i),
					TrackerName: "tracker-name",
					TriggerName: fmt.Sprintf("%d-trigger-name", i),
					BaseURL:     fmt.Sprintf("%d-base-url", i),
				})
			}
			return res
		}

		err := svc.db.Update(func(tx *bolt.Tx) error {
			subs := generateSubscriptions(5)
			bkt := tx.Bucket([]byte(subscriptionsBktName))
			refsBkt, err := tx.Bucket([]byte(trackerToSubsBktName)).CreateBucketIfNotExists([]byte("tracker-name"))
			require.NoError(t, err)

			for _, sub := range subs {
				bts, err := json.Marshal(sub)
				require.NoError(t, err)

				err = bkt.Put([]byte(sub.ID), bts)
				require.NoError(t, err)

				err = refsBkt.Put([]byte(sub.ID), []byte(time.Now().Format(time.RFC3339Nano)))
				require.NoError(t, err)
			}
			return nil
		})
		require.NoError(t, err)

		subs, err := svc.List(context.Background(), "tracker-name")
		require.NoError(t, err)

		assert.ElementsMatch(t, generateSubscriptions(5), subs)
	})

	t.Run("list all", func(t *testing.T) {
		svc := prepareSubscription(t)

		generateSubscriptions := func(amount int) (res []store.Subscription) {
			for i := 1; i <= amount; i++ {
				res = append(res, store.Subscription{
					ID:          fmt.Sprintf("%d-id", i),
					TrackerRef:  fmt.Sprintf("%d-tracker-id", i),
					TrackerName: fmt.Sprintf("%d-tracker-name", i),
					TriggerName: fmt.Sprintf("%d-trigger-name", i),
					BaseURL:     fmt.Sprintf("%d-base-url", i),
				})
			}
			return res
		}

		err := svc.db.Update(func(tx *bolt.Tx) error {
			subs := generateSubscriptions(5)
			bkt := tx.Bucket([]byte(subscriptionsBktName))

			for _, sub := range subs {
				bts, err := json.Marshal(sub)
				require.NoError(t, err)

				err = bkt.Put([]byte(sub.ID), bts)
				require.NoError(t, err)
			}
			return nil
		})
		require.NoError(t, err)

		subs, err := svc.List(context.Background(), "")
		require.NoError(t, err)

		assert.ElementsMatch(t, generateSubscriptions(5), subs)
	})
}

func TestSubscription_Delete(t *testing.T) {
	svc := prepareSubscription(t)

	err := svc.db.Update(func(tx *bolt.Tx) error {
		bts, err := json.Marshal(store.Subscription{
			ID:          "id",
			TrackerRef:  "tracker-id",
			TrackerName: "tracker-name",
			TriggerName: "trigger-name",
			BaseURL:     "base-url",
		})
		require.NoError(t, err)
		err = tx.Bucket([]byte(subscriptionsBktName)).Put([]byte("id"), bts)
		require.NoError(t, err)

		bkt, err := tx.Bucket([]byte(trackerToSubsBktName)).CreateBucketIfNotExists([]byte("tracker-name"))
		require.NoError(t, err)
		err = bkt.Put([]byte("id"), []byte("2006-01-02T15:04:05.999999999Z07:00"))
		require.NoError(t, err)

		err = tx.Bucket([]byte(trackerTriggerRefToSubsBktName)).Put([]byte("tracker-name:tracker-id"), []byte("id"))
		require.NoError(t, err)
		return nil
	})
	require.NoError(t, err)

	err = svc.Delete(context.Background(), "id")
	require.NoError(t, err)

	err = svc.db.View(func(tx *bolt.Tx) error {
		assert.Nil(t, tx.Bucket([]byte(subscriptionsBktName)).Get([]byte("id")))
		assert.Nil(t, tx.Bucket([]byte(trackerToSubsBktName)).
			Bucket([]byte("tracker-name")).
			Get([]byte("id")),
		)
		return nil
	})
	require.NoError(t, err)
}

func prepareSubscription(t *testing.T) *Subscriptions {
	loc, err := ioutil.TempDir("", "test_dastracker")
	require.NoError(t, err, "failed to make temp dir")

	svc, err := NewSubscription(path.Join(loc, "dastracker_subscriptions_test.db"), bolt.Options{}, logx.Nop())
	require.NoError(t, err)

	t.Cleanup(func() { assert.NoError(t, os.RemoveAll(loc)) })

	return svc
}

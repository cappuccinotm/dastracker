package bolt

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/cappuccinotm/dastracker/app/errs"
	"github.com/cappuccinotm/dastracker/app/store"
	"github.com/google/uuid"
	bolt "go.etcd.io/bbolt"
)

const (
	subscriptionsBktName   = "subscriptions"
	trackerToWhRefsBktName = "tracker_wh_refs"
)

// Subscriptions implements engine.Subscriptions over the BoltDB storage.
// subscriptions: key - subscriptionID, val - subscription
// refs: key - reference, val - nested bucket with keys as subscriptionIDs and values as ts
type Subscriptions struct {
	fileName string
	db       *bolt.DB
}

// NewSubscription creates buckets and initial data processing
func NewSubscription(fileName string, options bolt.Options) (*Subscriptions, error) {
	db, err := bolt.Open(fileName, 0600, &options)
	if err != nil {
		return nil, fmt.Errorf("failed to make boltdb for %s: %w", fileName, err)
	}

	err = db.Update(func(tx *bolt.Tx) error {
		if _, err := tx.CreateBucketIfNotExists([]byte(subscriptionsBktName)); err != nil {
			return fmt.Errorf("failed to create top-level bucket %s: %w", subscriptionsBktName, err)
		}

		if _, err := tx.CreateBucketIfNotExists([]byte(trackerToWhRefsBktName)); err != nil {
			return fmt.Errorf("failed to create top-level bucket %s: %w", trackerToWhRefsBktName, err)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize boltdb buckets for %s: %w", fileName, err)
	}

	return &Subscriptions{db: db, fileName: fileName}, nil
}

// Create creates a subscription in the storage.
func (b *Subscriptions) Create(ctx context.Context, wh store.Subscription) (string, error) {
	wh.ID = uuid.NewString()

	if err := b.Update(ctx, wh); err != nil {
		return "", fmt.Errorf("put subscription into storage: %w", err)
	}

	return wh.ID, nil
}

// Get subscription by id.
func (b *Subscriptions) Get(_ context.Context, id string) (store.Subscription, error) {
	var wh store.Subscription
	err := b.db.View(func(tx *bolt.Tx) error {
		var err error
		if wh, err = b.get(tx, id); err != nil {
			return fmt.Errorf("get from bucket: %w", err)
		}
		return nil
	})
	if err != nil {
		return store.Subscription{}, fmt.Errorf("view storage: %b", err)
	}

	return wh, nil
}

// Delete subscription by id.
func (b *Subscriptions) Delete(_ context.Context, whID string) error {
	err := b.db.Update(func(tx *bolt.Tx) error {
		wh, err := b.get(tx, whID)
		if err != nil {
			return fmt.Errorf("get subscription: %w", err)
		}

		if err = tx.Bucket([]byte(subscriptionsBktName)).Delete([]byte(whID)); err != nil {
			return fmt.Errorf("delete subscription itself: %w", err)
		}

		trkBkt := tx.Bucket([]byte(trackerToWhRefsBktName)).Bucket([]byte(wh.TrackerName))
		if trkBkt == nil {
			return fmt.Errorf("bucket with %q tracker not found: %w", wh.TrackerName, errs.ErrNotFound)
		}

		if err = trkBkt.Delete([]byte(whID)); err != nil {
			return fmt.Errorf("delete %s reference in %s tracker's bucket: %w", whID, wh.TrackerName, err)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("update storage: %w", err)
	}

	return nil
}

// Update totally rewrites the provided subscription entry.
func (b *Subscriptions) Update(_ context.Context, wh store.Subscription) error {
	bts, err := json.Marshal(wh)
	if err != nil {
		return fmt.Errorf("marshal subscription: %b", err)
	}

	err = b.db.Update(func(tx *bolt.Tx) error {
		if err = tx.Bucket([]byte(subscriptionsBktName)).Put([]byte(wh.ID), bts); err != nil {
			return fmt.Errorf("put subscription to storage: %w", err)
		}

		if wh.TrackerRef == "" {
			return nil
		}

		bkt, err := tx.Bucket([]byte(trackerToWhRefsBktName)).CreateBucketIfNotExists([]byte(wh.TrackerRef))
		if err != nil {
			return fmt.Errorf("create refs bucket for tracker %s: %w", wh.TrackerRef, err)
		}

		if err = bkt.Put([]byte(wh.ID), []byte(time.Now().Format(time.RFC3339Nano))); err != nil {
			return fmt.Errorf("put %s subscription reference into %s tracker's bucket: %w",
				wh.ID, wh.TrackerRef, err)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("update storage: %w", err)
	}
	return nil
}

// List lists the subscriptions registered on the given tracker.
func (b *Subscriptions) List(_ context.Context, trackerName string) ([]store.Subscription, error) {
	var subscriptions []store.Subscription
	err := b.db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket([]byte(trackerToWhRefsBktName)).Bucket([]byte(trackerName))
		if bkt == nil {
			return nil
		}

		err := bkt.ForEach(func(whID, _ []byte) error {
			wh, err := b.get(tx, string(whID))
			if err != nil {
				return fmt.Errorf("get subscription %s: %w", whID, err)
			}

			subscriptions = append(subscriptions, wh)
			return nil
		})
		if err != nil {
			return fmt.Errorf("iterate over each reference: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("view storage: %w", err)
	}
	return subscriptions, nil
}

func (b *Subscriptions) get(tx *bolt.Tx, whID string) (store.Subscription, error) {
	bts := tx.Bucket([]byte(subscriptionsBktName)).Get([]byte(whID))
	if bts == nil {
		return store.Subscription{}, errs.ErrNotFound
	}

	var wh store.Subscription
	if err := json.Unmarshal(bts, &wh); err != nil {
		return store.Subscription{}, fmt.Errorf("unmarshal subscription: %w", err)
	}
	return wh, nil
}
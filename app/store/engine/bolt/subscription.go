package bolt

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/cappuccinotm/dastracker/app/errs"
	"github.com/cappuccinotm/dastracker/app/store"
	"github.com/cappuccinotm/dastracker/pkg/logx"
	"github.com/google/uuid"
	bolt "go.etcd.io/bbolt"
)

const (
	subscriptionsBktName           = "subscriptions"
	trackerToSubsBktName           = "tracker_refs_to_subs"
	trackerTriggerRefToSubsBktName = "tracker_trigger_to_subs"
)

// Subscriptions implements engine.Subscriptions over the BoltDB storage.
// It contains three top-level buckets:
// subscriptions: key - subscriptionID, val - subscription
// tracker_refs_to_subs: k: tracker name, v: nested bucket with
//							k: subscriptionID, v: timestamp in RFC3339 nano
// tracker_trigger_to_subs: k - reference in "trackerName:triggerName" form, v: subscriptionID
type Subscriptions struct {
	fileName string
	l        logx.Logger
	db       *bolt.DB
}

// NewSubscription creates buckets and initial data processing
func NewSubscription(fileName string, options bolt.Options, log logx.Logger) (*Subscriptions, error) {
	db, err := bolt.Open(fileName, 0600, &options)
	if err != nil {
		return nil, fmt.Errorf("failed to make boltdb for %s: %w", fileName, err)
	}

	err = db.Update(func(tx *bolt.Tx) error {
		for _, bktName := range []string{subscriptionsBktName, trackerToSubsBktName, trackerTriggerRefToSubsBktName} {
			if _, err := tx.CreateBucketIfNotExists([]byte(bktName)); err != nil {
				return fmt.Errorf("failed to create bucket %s: %w", bktName, err)
			}
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize boltdb buckets for %s: %w", fileName, err)
	}

	return &Subscriptions{db: db, fileName: fileName, l: log}, nil
}

// Create creates a subscription in the storage.
func (b *Subscriptions) Create(_ context.Context, sub store.Subscription) (string, error) {
	sub.ID = uuid.NewString()

	err := b.db.Update(func(tx *bolt.Tx) error {
		ref := b.trackerTriggerRef(sub)
		if bts := tx.Bucket([]byte(trackerTriggerRefToSubsBktName)).Get([]byte(ref)); bts != nil {
			return fmt.Errorf("subscription %s is already assigned to %q tracker:trigger pair: %w", sub.ID, ref, errs.ErrExists)
		}

		if err := b.put(tx, sub); err != nil {
			return fmt.Errorf("put subscription %s: %w", sub.ID, err)
		}
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("update storage: %w", err)
	}

	return sub.ID, nil
}

// Update totally rewrites the provided subscription entry.
func (b *Subscriptions) Update(_ context.Context, sub store.Subscription) error {
	if err := b.db.Update(func(tx *bolt.Tx) error { return b.put(tx, sub) }); err != nil {
		return fmt.Errorf("update storage: %w", err)
	}
	return nil
}

// Get subscription by id.
func (b *Subscriptions) Get(_ context.Context, id string) (store.Subscription, error) {
	var sub store.Subscription
	err := b.db.View(func(tx *bolt.Tx) error {
		var err error
		if sub, err = b.get(tx, id); err != nil {
			return fmt.Errorf("get from bucket: %w", err)
		}
		return nil
	})
	if err != nil {
		return store.Subscription{}, fmt.Errorf("view storage: %b", err)
	}

	return sub, nil
}

// Delete subscription by id.
func (b *Subscriptions) Delete(_ context.Context, subID string) error {
	err := b.db.Update(func(tx *bolt.Tx) error {
		sub, err := b.get(tx, subID)
		if err != nil {
			return fmt.Errorf("get subscription: %w", err)
		}

		if trkBkt := tx.Bucket([]byte(trackerToSubsBktName)).Bucket([]byte(sub.TrackerName)); trkBkt != nil {
			if err = trkBkt.Delete([]byte(subID)); err != nil {
				return fmt.Errorf("delete %s reference in %s tracker's bucket: %w", subID, sub.TrackerName, err)
			}
		}

		ref := b.trackerTriggerRef(sub)
		if err = tx.Bucket([]byte(trackerTriggerRefToSubsBktName)).Delete([]byte(ref)); err != nil {
			return fmt.Errorf("delete %s reference fo tracker:trigger %s: %w", subID, ref, err)
		}

		if err = tx.Bucket([]byte(subscriptionsBktName)).Delete([]byte(subID)); err != nil {
			return fmt.Errorf("delete subscription itself: %w", err)
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
		if trackerName == "" {
			err := tx.Bucket([]byte(subscriptionsBktName)).ForEach(func(k, v []byte) error {
				sub, err := b.get(tx, string(k))
				if err != nil {
					return fmt.Errorf("get subscription: %w", err)
				}
				subscriptions = append(subscriptions, sub)
				return nil
			})
			if err != nil {
				return fmt.Errorf("iterate over subscriptions bucket: %w", err)
			}
			return nil
		}

		bkt := tx.Bucket([]byte(trackerToSubsBktName)).Bucket([]byte(trackerName))
		if bkt == nil {
			return nil
		}

		err := bkt.ForEach(func(subID, _ []byte) error {
			sub, err := b.get(tx, string(subID))
			if err != nil {
				return fmt.Errorf("get subscription %s: %w", subID, err)
			}

			subscriptions = append(subscriptions, sub)
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

func (b *Subscriptions) get(tx *bolt.Tx, subID string) (store.Subscription, error) {
	bts := tx.Bucket([]byte(subscriptionsBktName)).Get([]byte(subID))
	if bts == nil {
		return store.Subscription{}, errs.ErrNotFound
	}

	var sub store.Subscription
	if err := json.Unmarshal(bts, &sub); err != nil {
		return store.Subscription{}, fmt.Errorf("unmarshal subscription: %w", err)
	}
	return sub, nil
}

func (b *Subscriptions) put(tx *bolt.Tx, sub store.Subscription) error {
	bts, err := json.Marshal(sub)
	if err != nil {
		return fmt.Errorf("marshal subscription %s: %w", sub.ID, err)
	}

	if err = tx.Bucket([]byte(subscriptionsBktName)).Put([]byte(sub.ID), bts); err != nil {
		return fmt.Errorf("put subscription %s to storage: %w", sub.ID, err)
	}

	bkt, err := tx.Bucket([]byte(trackerToSubsBktName)).CreateBucketIfNotExists([]byte(sub.TrackerName))
	if err != nil {
		return fmt.Errorf("get %s tracker's refs bucket: %w", sub.TrackerName, err)
	}

	if err = bkt.Put([]byte(sub.ID), []byte(time.Now().Format(time.RFC3339Nano))); err != nil {
		return fmt.Errorf("put %s subscription reference into %s tracker's bucket: %w",
			sub.ID, sub.TrackerName, err)
	}

	ref := b.trackerTriggerRef(sub)
	if err = tx.Bucket([]byte(trackerTriggerRefToSubsBktName)).Put([]byte(ref), []byte(sub.ID)); err != nil {
		return fmt.Errorf("put %s's tracker:trigger %s reference: %w",
			sub.ID, ref, err)
	}

	return nil
}

func (b *Subscriptions) trackerTriggerRef(sub store.Subscription) string {
	return fmt.Sprintf("%s:%s", sub.TrackerName, sub.TriggerName)
}

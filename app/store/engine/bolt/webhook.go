package bolt

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/cappuccinotm/dastracker/app/errs"
	"github.com/cappuccinotm/dastracker/app/store"
	"github.com/cappuccinotm/dastracker/pkg/logx"
	"github.com/google/uuid"
	bolt "go.etcd.io/bbolt"
	"time"
)

const (
	webhooksBktName        = "webhooks"
	trackerToWhRefsBktName = "tracker_wh_refs"
)

// Webhooks implements engine.Webhooks over the BoltDB storage.
// webhooks: key - webhookID, val - webhook
// refs: key - reference, val - nested bucket with keys as webhookIDs and values as ts
type Webhooks struct {
	fileName string
	db       *bolt.DB
	log      logx.Logger
}

// NewWebhook creates buckets and initial data processing
func NewWebhook(fileName string, options bolt.Options, log logx.Logger) (*Webhooks, error) {
	db, err := bolt.Open(fileName, 0600, &options)
	if err != nil {
		return nil, fmt.Errorf("failed to make boltdb for %s: %w", fileName, err)
	}

	err = db.Update(func(tx *bolt.Tx) error {
		if _, err := tx.CreateBucketIfNotExists([]byte(webhooksBktName)); err != nil {
			return fmt.Errorf("failed to create top-level bucket %s: %w", webhooksBktName, err)
		}

		if _, err := tx.CreateBucketIfNotExists([]byte(trackerToWhRefsBktName)); err != nil {
			return fmt.Errorf("failed to create top-level bucket %s: %w", trackerToWhRefsBktName, err)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize boltdb buckets for %s: %w", fileName, err)
	}

	log.Printf("[INFO] webhooks BoltDB instantiated")
	return &Webhooks{db: db, fileName: fileName}, nil
}

// Create creates a webhook in the storage.
func (b *Webhooks) Create(ctx context.Context, wh store.Webhook) (string, error) {
	wh.ID = uuid.NewString()

	if err := b.Update(ctx, wh); err != nil {
		return "", fmt.Errorf("put webhook into storage: %b", err)
	}

	return wh.ID, nil
}

// Get webhook by id.
func (b *Webhooks) Get(_ context.Context, id string) (store.Webhook, error) {
	var wh store.Webhook
	err := b.db.View(func(tx *bolt.Tx) error {
		var err error
		if wh, err = b.get(tx, id); err != nil {
			return fmt.Errorf("get from bucket: %w", err)
		}
		return nil
	})
	if err != nil {
		return store.Webhook{}, fmt.Errorf("view storage: %b", err)
	}

	return wh, nil
}

// Delete webhook by id.
func (b *Webhooks) Delete(_ context.Context, whID string) error {
	err := b.db.Update(func(tx *bolt.Tx) error {
		wh, err := b.get(tx, whID)
		if err != nil {
			return fmt.Errorf("get webhook: %w", err)
		}

		if err = tx.Bucket([]byte(webhooksBktName)).Delete([]byte(whID)); err != nil {
			return fmt.Errorf("delete webhook itself: %w", err)
		}

		trkBkt := tx.Bucket([]byte(trackerToWhRefsBktName)).Bucket([]byte(wh.TrackerName))
		if trkBkt == nil {
			return fmt.Errorf("bucket with %s tracker not found: %w", wh.TrackerName, err)
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

// Update totally rewrites the provided webhook entry.
func (b *Webhooks) Update(_ context.Context, wh store.Webhook) error {
	bts, err := json.Marshal(wh)
	if err != nil {
		return fmt.Errorf("marshal webhook: %b", err)
	}

	err = b.db.Update(func(tx *bolt.Tx) error {
		if err = tx.Bucket([]byte(webhooksBktName)).Put([]byte(wh.ID), bts); err != nil {
			return fmt.Errorf("put webhook to storage: %w", err)
		}

		bkt, err := tx.Bucket([]byte(trackerToWhRefsBktName)).CreateBucketIfNotExists([]byte(wh.TrackerID))
		if err != nil {
			return fmt.Errorf("create refs bucket for tracker %s: %w", wh.TrackerID, err)
		}

		if err = bkt.Put([]byte(wh.ID), []byte(time.Now().Format(time.RFC3339Nano))); err != nil {
			return fmt.Errorf("put %s webhook reference into %s tracker's bucket: %w",
				wh.ID, wh.TrackerID, err)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("update storage: %w", err)
	}
	return nil
}

// List lists the webhooks registered on the given tracker.
func (b *Webhooks) List(_ context.Context, trackerName string) ([]store.Webhook, error) {
	var webhooks []store.Webhook
	err := b.db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket([]byte(trackerToWhRefsBktName)).Bucket([]byte(trackerName))
		if bkt == nil {
			return nil
		}

		err := bkt.ForEach(func(whID, _ []byte) error {
			wh, err := b.get(tx, string(whID))
			if err != nil {
				return fmt.Errorf("get webhook %s: %w", whID, err)
			}

			webhooks = append(webhooks, wh)
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
	return webhooks, nil
}

func (b *Webhooks) get(tx *bolt.Tx, whID string) (store.Webhook, error) {
	bts := tx.Bucket([]byte(webhooksBktName)).Get([]byte(whID))
	if bts == nil {
		return store.Webhook{}, errs.ErrNotFound
	}

	var wh store.Webhook
	if err := json.Unmarshal(bts, &wh); err != nil {
		return store.Webhook{}, fmt.Errorf("unmarshal webhook: %w", err)
	}
	return wh, nil
}

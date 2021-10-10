package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/cappuccinotm/dastracker/app/store"
	"github.com/google/uuid"
	bolt "go.etcd.io/bbolt"
)

const (
	ticketsBktName = "tickets"
	refsBktName    = "refs"
)

// Bolt implements Interface over BoltDB.
// tickets: key - ticketID, val - ticket
// refs: key - reference, val - ticketID
type Bolt struct {
	fileName string
	db       *bolt.DB
}

// NewBolt creates buckets and initial data processing
func NewBolt(fileName string, options bolt.Options) (*Bolt, error) {
	db, err := bolt.Open(fileName, 0600, &options)
	if err != nil {
		return nil, fmt.Errorf("failed to make boltdb for %s: %w", fileName, err)
	}

	err = db.Update(func(tx *bolt.Tx) error {
		if _, err := tx.CreateBucketIfNotExists([]byte(ticketsBktName)); err != nil {
			return fmt.Errorf("failed to create top-level bucket %s: %w", ticketsBktName, err)
		}

		if _, err := tx.CreateBucketIfNotExists([]byte(refsBktName)); err != nil {
			return fmt.Errorf("failed to create top-level bucket %s: %w", refsBktName, err)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize boltdb buckets for %s: %w", fileName, err)
	}

	log.Printf("[INFO] Users BoltDB instantiated")
	return &Bolt{
		db:       db,
		fileName: fileName,
	}, nil
}

func (b *Bolt) Create(_ context.Context, ticket store.Ticket) (ticketID string, err error) {
	err = b.db.Update(func(tx *bolt.Tx) error {
		ticket.ID = uuid.NewString()

		bts, err := json.Marshal(ticket)
		if err != nil {
			return fmt.Errorf("marshal ticket: %w", err)
		}

		if err = tx.Bucket([]byte(ticketsBktName)).Put([]byte(ticket.ID), bts); err != nil {
			return fmt.Errorf("put ticket to storage: %w", err)
		}

		for trackerName, trackerTaskID := range ticket.TrackerIDs {
			ref := taskRef(trackerName, trackerTaskID)
			if err = tx.Bucket([]byte(refsBktName)).Put([]byte(ref), []byte(ticket.ID)); err != nil {
				return fmt.Errorf("put ref %s: %w", ref, err)
			}
		}

		return nil
	})
	if err != nil {
		return "", fmt.Errorf("update storage: %w", err)
	}
	return ticket.ID, nil
}

func (b *Bolt) Update(_ context.Context, ticket store.Ticket) (err error) {
	err = b.db.Update(func(tx *bolt.Tx) error {
		bts, err := json.Marshal(ticket)
		if err != nil {
			return fmt.Errorf("marshal ticket: %w", err)
		}

		if err = tx.Bucket([]byte(ticketsBktName)).Put([]byte(ticket.ID), bts); err != nil {
			return fmt.Errorf("put ticket to storage: %w", err)
		}

		for trackerName, trackerTaskID := range ticket.TrackerIDs {
			ref := taskRef(trackerName, trackerTaskID)
			if err = tx.Bucket([]byte(refsBktName)).Put([]byte(ref), []byte(ticket.ID)); err != nil {
				return fmt.Errorf("put ref %s: %w", ref, err)
			}
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("update storage: %w", err)
	}
	return nil
}

func (b *Bolt) Get(_ context.Context, trackerName, trackerTaskID string) (store.Ticket, error) {
	ref := taskRef(trackerName, trackerTaskID)
	var ticketBts []byte

	err := b.db.View(func(tx *bolt.Tx) error {
		ticketID := tx.Bucket([]byte(refsBktName)).Get([]byte(ref))
		if ticketID == nil {
			return fmt.Errorf("ref not found: %w", ErrNotFound)
		}

		if ticketBts = tx.Bucket([]byte(ticketsBktName)).Get(ticketID); ticketBts == nil {
			return fmt.Errorf("ticket %s not found: %w", ticketID, ErrNotFound)
		}

		return nil
	})
	if err != nil {
		return store.Ticket{}, fmt.Errorf("view storage: %w", err)
	}

	var ticket store.Ticket

	if err = json.Unmarshal(ticketBts, &ticket); err != nil {
		return store.Ticket{}, fmt.Errorf("unmarshal: %w", err)
	}

	return ticket, nil
}

func taskRef(trackerName, trackerTaskID string) string {
	return fmt.Sprintf("%s!!%s", trackerName, trackerTaskID)
}

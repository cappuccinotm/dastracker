package bolt

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/cappuccinotm/dastracker/app/errs"
	"github.com/cappuccinotm/dastracker/app/store"
	"github.com/cappuccinotm/dastracker/app/store/engine"
	"github.com/google/uuid"
	bolt "go.etcd.io/bbolt"
	"log"
)

const (
	ticketsBktName = "tickets"
	refsBktName    = "refs"
)

// Tickets implements engine.Tickets over BoltDB.
// tickets: key - ticketID, val - ticket
// refs: key - reference, val - ticketID
type Tickets struct {
	fileName string
	db       *bolt.DB
}

// NewTickets creates buckets and initial data processing
func NewTickets(fileName string, options bolt.Options) (*Tickets, error) {
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

	log.Printf("[INFO] tickets BoltDB instantiated")
	return &Tickets{db: db, fileName: fileName}, nil
}

// Create creates ticket in the storage.
func (b *Tickets) Create(ctx context.Context, ticket store.Ticket) (string, error) {
	ticket.ID = uuid.NewString()

	if err := b.Update(ctx, ticket); err != nil {
		return "", fmt.Errorf("put ticket into storage: %w", err)
	}

	return ticket.ID, nil
}

// Update updates ticket by its ID.
func (b *Tickets) Update(_ context.Context, ticket store.Ticket) error {
	bts, err := json.Marshal(ticket)
	if err != nil {
		return fmt.Errorf("marshal ticket: %w", err)
	}

	err = b.db.Update(func(tx *bolt.Tx) error {
		if err = tx.Bucket([]byte(ticketsBktName)).Put([]byte(ticket.ID), bts); err != nil {
			return fmt.Errorf("put ticket to storage: %w", err)
		}

		if err = b.putTicketRefs(tx, ticket.ID, ticket.TrackerIDs); err != nil {
			return fmt.Errorf("put ticket refs: %w", err)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("update storage: %w", err)
	}
	return nil
}

// Get returns ticket by the given engine.GetRequest.
func (b *Tickets) Get(_ context.Context, req engine.GetRequest) (store.Ticket, error) {
	ticketID := req.TicketID

	var ticketBts []byte
	err := b.db.View(func(tx *bolt.Tx) error {
		var err error

		if ticketID == "" && !req.Locator.Empty() {
			if ticketID, err = b.getTicketID(tx, req.Locator); err != nil {
				return fmt.Errorf("get ticket id by ref %q: %w",
					req.Locator.String(), err)
			}
		}

		if ticketBts = tx.Bucket([]byte(ticketsBktName)).Get([]byte(ticketID)); ticketBts == nil {
			return fmt.Errorf("ticket %s not found: %w", ticketID, errs.ErrNotFound)
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

func (b *Tickets) putTicketRefs(tx *bolt.Tx, ticketID string, trackerIDs store.TrackerIDs) error {
	locators := trackerIDs.Locators()
	for _, locator := range locators {
		if err := tx.Bucket([]byte(refsBktName)).Put([]byte(taskRef(locator)), []byte(ticketID)); err != nil {
			return fmt.Errorf("put ref for %q on %s: %w", locator.String(), ticketID, err)
		}
	}
	return nil
}

func (b *Tickets) getTicketID(tx *bolt.Tx, locator store.Locator) (string, error) {
	ref := taskRef(locator)
	ticketID := tx.Bucket([]byte(refsBktName)).Get([]byte(ref))
	if ticketID == nil {
		return "", errs.ErrNotFound
	}
	return string(ticketID), nil
}

func taskRef(locator store.Locator) string {
	return fmt.Sprintf("%s!!%s", locator.Tracker, locator.TaskID)
}

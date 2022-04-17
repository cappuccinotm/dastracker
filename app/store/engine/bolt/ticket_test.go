//go:build !race
// +build !race

// bolt itself thread-safe so there is no need in race-detector
package bolt

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"os"
	"path"
	"testing"

	"github.com/cappuccinotm/dastracker/app/store"
	"github.com/cappuccinotm/dastracker/app/store/engine"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	bolt "go.etcd.io/bbolt"
)

func TestTickets_Create(t *testing.T) {
	svc := prepareTickets(t)

	tkt := store.Ticket{
		Variations: map[string]store.Task{
			"tracker-1": {ID: "task-id-1", Content: store.Content{Body: "body-1", Title: "title-1"}},
			"tracker-2": {ID: "task-id-2", Content: store.Content{Body: "body-2", Title: "title-2"}},
			"tracker-3": {ID: "task-id-3", Content: store.Content{Body: "body-3", Title: "title-3"}},
			"tracker-4": {ID: "task-id-4", Content: store.Content{Body: "body-4", Title: "title-4"}},
		},
	}

	id, err := svc.Create(context.Background(), tkt)
	require.NoError(t, err)
	assert.NotEmpty(t, id)

	err = svc.db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket([]byte(ticketsBktName))
		tktBts := bkt.Get([]byte(id))
		var tkt store.Ticket
		assert.NoError(t, json.Unmarshal(tktBts, &tkt))
		expectedTkt := tkt
		expectedTkt.ID = id
		assert.Equal(t, expectedTkt, tkt)

		refsBkt := tx.Bucket([]byte(ticketRefsBktName))

		locators := expectedTkt.Variations.Locators()
		for _, locator := range locators {
			refTicketID := refsBkt.Get([]byte(taskRef(locator)))
			assert.Equal(t, []byte(id), refTicketID)
		}

		return nil
	})
	require.NoError(t, err)
}

func TestTickets_Update(t *testing.T) {
	svc := prepareTickets(t)

	expectedTkt := store.Ticket{
		ID: "id",
		Variations: map[string]store.Task{
			"tracker-1": {ID: "task-id-1", Content: store.Content{Body: "body-1", Title: "title-1"}},
			"tracker-2": {ID: "task-id-2", Content: store.Content{Body: "body-2", Title: "title-2"}},
			"tracker-3": {ID: "task-id-3", Content: store.Content{Body: "body-3", Title: "title-3"}},
			"tracker-4": {ID: "task-id-4", Content: store.Content{Body: "body-4", Title: "title-4"}},
		},
	}

	err := svc.Update(context.Background(), expectedTkt)
	require.NoError(t, err)

	err = svc.db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket([]byte(ticketsBktName))
		tktBts := bkt.Get([]byte("id"))
		var tkt store.Ticket
		assert.NoError(t, json.Unmarshal(tktBts, &tkt))
		assert.Equal(t, expectedTkt, tkt)

		refsBkt := tx.Bucket([]byte(ticketRefsBktName))

		locators := expectedTkt.Variations.Locators()
		for _, locator := range locators {
			refTicketID := refsBkt.Get([]byte(taskRef(locator)))
			assert.Equal(t, []byte("id"), refTicketID)
		}

		return nil
	})
	require.NoError(t, err)
}

func TestTickets_Get(t *testing.T) {
	expectedTkt := store.Ticket{
		ID: "id",
		Variations: map[string]store.Task{
			"tracker-1": {ID: "task-id-1", Content: store.Content{Body: "body-1", Title: "title-1"}},
			"tracker-2": {ID: "task-id-2", Content: store.Content{Body: "body-2", Title: "title-2"}},
			"tracker-3": {ID: "task-id-3", Content: store.Content{Body: "body-3", Title: "title-3"}},
			"tracker-4": {ID: "task-id-4", Content: store.Content{Body: "body-4", Title: "title-4"}},
		},
	}

	expectedTktBts, err := json.Marshal(expectedTkt)
	require.NoError(t, err)

	t.Run("direct ticket id provided", func(t *testing.T) {
		svc := prepareTickets(t)

		err = svc.db.Update(func(tx *bolt.Tx) error {
			err = tx.Bucket([]byte(ticketsBktName)).Put([]byte("id"), expectedTktBts)
			require.NoError(t, err)
			return nil
		})
		require.NoError(t, err)

		tkt, err := svc.Get(context.Background(), engine.GetRequest{
			TicketID: "id",
		})
		require.NoError(t, err)
		assert.Equal(t, expectedTkt, tkt)
	})

	t.Run("locator provided", func(t *testing.T) {
		svc := prepareTickets(t)

		err = svc.db.Update(func(tx *bolt.Tx) error {
			err = tx.Bucket([]byte(ticketsBktName)).Put([]byte("id"), expectedTktBts)
			require.NoError(t, err)

			err = tx.Bucket([]byte(ticketRefsBktName)).
				Put([]byte(taskRef(store.Locator{Tracker: "tracker-2", ID: "task-id-2"})), []byte("id"))
			require.NoError(t, err)

			return nil
		})
		require.NoError(t, err)

		tkt, err := svc.Get(context.Background(), engine.GetRequest{
			Locator: store.Locator{
				Tracker: "tracker-2",
				ID:      "task-id-2",
			},
		})
		require.NoError(t, err)
		assert.Equal(t, expectedTkt, tkt)
	})
}

func TestNewTickets(t *testing.T) {
	svc := prepareTickets(t)
	err := svc.db.View(func(tx *bolt.Tx) error {
		assert.NotNil(t, tx.Bucket([]byte(ticketsBktName)))
		assert.NotNil(t, tx.Bucket([]byte(ticketRefsBktName)))
		return nil
	})
	require.NoError(t, err)
}

func prepareTickets(t *testing.T) *Tickets {
	loc, err := ioutil.TempDir("", "test_dastracker")
	require.NoError(t, err, "failed to make temp dir")

	svc, err := NewTickets(path.Join(loc, "dastracker_tickets_test.db"), bolt.Options{})
	require.NoError(t, err)

	t.Cleanup(func() { assert.NoError(t, os.RemoveAll(loc)) })

	return svc
}

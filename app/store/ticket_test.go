package store

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestTicket_Patch(t *testing.T) {
	tkt := Ticket{
		ID:         "ticket-id",
		TrackerIDs: map[string]string{"trk1": "id1"},
		Content: Content{
			Body:   "original-body",
			Title:  "original-title",
			Fields: map[string]string{"o-f1": "o-v1", "o-f2": "o-v2"},
		},
	}
	tkt.Patch(Update{
		ReceivedFrom: Locator{Tracker: "tracker", TaskID: "task-id"},
		Content: Content{
			Body:   "body",
			Title:  "title",
			Fields: map[string]string{"f1": "v1", "f2": "v2"},
		},
	})
	assert.Equal(t, Ticket{
		ID:         "ticket-id",
		TrackerIDs: map[string]string{"trk1": "id1", "tracker": "task-id"},
		Content: Content{
			Body:   "body",
			Title:  "title",
			Fields: map[string]string{"f1": "v1", "f2": "v2"},
		},
	}, tkt)

	tkt.Patch(Update{
		Content: Content{
			Body:   "body",
			Title:  "title",
			Fields: map[string]string{"f1": "v1", "f2": "v2"},
		},
	})
	assert.Equal(t, Ticket{
		ID:         "ticket-id",
		TrackerIDs: map[string]string{"trk1": "id1", "tracker": "task-id"},
		Content: Content{
			Body:   "body",
			Title:  "title",
			Fields: map[string]string{"f1": "v1", "f2": "v2"},
		},
	}, tkt)
}

func TestLocator_String(t *testing.T) {
	assert.Equal(t, "tracker/task-id", Locator{Tracker: "tracker", TaskID: "task-id"}.String())
}

func TestLocator_Empty(t *testing.T) {
	assert.True(t, Locator{}.Empty())
	assert.True(t, Locator{TaskID: "task-id"}.Empty())
	assert.True(t, Locator{Tracker: "tracker"}.Empty())
	assert.False(t, Locator{Tracker: "tracker", TaskID: "task-id"}.Empty())
}

func TestTrackerIDs_Set(t *testing.T) {
	t.Run("map is nil", func(t *testing.T) {
		m := TrackerIDs(nil)
		m.Set("tracker", "task-id")
		assert.Equal(t, TrackerIDs(map[string]string{"tracker": "task-id"}), m)
	})

	t.Run("map is not nil", func(t *testing.T) {
		m := TrackerIDs{"other-tracker": "other-task-id"}
		m.Set("tracker", "task-id")
		assert.Equal(t, TrackerIDs(map[string]string{
			"tracker":       "task-id",
			"other-tracker": "other-task-id",
		}), m)
	})
}

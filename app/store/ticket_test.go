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
	tkt.Patch(Update{Content: Content{
		Body:   "body",
		Title:  "title",
		Fields: map[string]string{"f1": "v1", "f2": "v2"},
	}})
	assert.Equal(t, Ticket{
		ID:         "ticket-id",
		TrackerIDs: map[string]string{"trk1": "id1"},
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

package store

import (
	"fmt"

	"github.com/cappuccinotm/dastracker/app/errs"
)

// Content contains the data in the ticket itself.
type Content struct {
	Body   string       `json:"body"`
	Title  string       `json:"title"`
	Fields TicketFields `json:"fields"`
}

// TicketFields is an alias for a map of fields in form
// of map[fieldName]fieldValue.
type TicketFields map[string]string

// Ticket describes a basic task/ticket in task tracker.
type Ticket struct {
	ID         string     `json:"id"`
	TrackerIDs TrackerIDs `json:"tracker_ids"`

	Content
}

// Patch updates ticket fields with given update values.
func (t *Ticket) Patch(upd Update) {
	t.Content = upd.Content
	if !upd.ReceivedFrom.Empty() {
		t.TrackerIDs.Set(upd.ReceivedFrom.Tracker, upd.ReceivedFrom.ID)
	}
}

// TrackerIDs is a wrapper for a set of trackerIDs.
// Key - tracker name, value - task ID.
type TrackerIDs map[string]string

// Add checks that the given tracker doesn't already have an assigned task
// and if it has, returns error, otherwise adds the task ID to the list of ids.
func (m *TrackerIDs) Add(trackerName, taskID string) error {
	if m == nil {
		*m = map[string]string{}
	}
	if existingTaskID, ok := (*m)[trackerName]; ok {
		return fmt.Errorf("tracker %q already has a task with id %q, id %q is ambiguous: %w",
			trackerName, existingTaskID, taskID, errs.ErrExists)
	}
	(*m)[trackerName] = taskID
	return nil
}

// Set sets the task ID in the list of trackers.
func (m *TrackerIDs) Set(trackerName, taskID string) {
	if *m == nil {
		*m = map[string]string{}
	}
	(*m)[trackerName] = taskID
}

// Locators returns a list of locators from the map of trackerIDs.
func (m TrackerIDs) Locators() []Locator {
	res := make([]Locator, 0, len(m))
	for name, id := range m {
		res = append(res, Locator{Tracker: name, ID: id})
	}
	return res
}

// Get returns the tracker ID for the given tracker name.
func (m TrackerIDs) Get(name string) string {
	if m == nil {
		return ""
	}
	return m[name]
}

// Update describes a ticket update.
type Update struct {
	TriggerName  string  `json:"trigger_name"`
	URL          string  `json:"url"`
	ReceivedFrom Locator `json:"received_from"`

	Content
}

// Locator describes the path to the entity in the specific tracker.
type Locator struct {
	Tracker string `json:"tracker"`
	ID      string `json:"id"`
}

// String returns the location of the task in string representation.
func (l Locator) String() string { return l.Tracker + "/" + l.ID }

// Empty returns true if the locator is not specified.
func (l Locator) Empty() bool { return l.Tracker == "" || l.ID == "" }

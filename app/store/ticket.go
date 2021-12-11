package store

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
	ID         string            `json:"id"`
	TrackerIDs map[string]string `json:"tracker_ids"`

	Content
}

// Patch updates ticket fields with given update values.
func (t *Ticket) Patch(upd Update) { t.Content = upd.Content }

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
	TaskID  string `json:"task_id"`
}

// String returns the location of the task in string representation.
func (l Locator) String() string { return l.Tracker + "/" + l.TaskID }

// Empty returns true if the locator is not specified.
func (l Locator) Empty() bool { return l.Tracker == "" || l.TaskID == "" }

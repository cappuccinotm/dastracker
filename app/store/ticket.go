package store

// Content contains the data in the ticket itself.
type Content struct {
	Body   string            `json:"body"`
	Title  string            `json:"title"`
	Fields map[string]string `json:"fields"` // map[name]value
}

// Ticket describes a basic task/ticket in task tracker.
type Ticket struct {
	ID         string            `json:"id"`
	TrackerIDs map[string]string // map[trackerName]taskID

	Content
}

// Patch updates ticket fields with given update values.
func (t *Ticket) Patch(upd Update) { t.Content = upd.Content }

// Update describes a ticket update.
type Update struct {
	URL     string  `json:"url"`
	Locator Locator `json:"locator"`

	Content
}

// Locator describes the path to the ticket in the specific tracker.
type Locator struct {
	Tracker string `json:"tracker"`
	TaskID  string `json:"task_id"`
}

// String returns the location of the task in string representation.
func (l Locator) String() string { return l.Tracker + "/" + l.TaskID }

// Empty returns true if the locator is not specified.
func (l Locator) Empty() bool { return l.Tracker == "" || l.TaskID == "" }

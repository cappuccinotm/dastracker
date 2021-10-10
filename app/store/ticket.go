package store

// Ticket describes a basic task/ticket in task tracker.
type Ticket struct {
	ID     string
	Body   string
	Title  string
	Fields map[string]string // map[name]value

	TrackerIDs map[string]string // map[trackerName]taskID
}

// Update describes a ticket update.
type Update struct {
	Tracker       string
	TrackerTaskID string
	Body          string
	Title         string
	Fields        map[string]string // map[name]value
}

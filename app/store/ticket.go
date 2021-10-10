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
	URL           string            `json:"url"`
	Tracker       string            `json:"tracker"`
	TrackerTaskID string            `json:"tracker_task_id"`
	Body          string            `json:"body"`
	Title         string            `json:"title"`
	Fields        map[string]string `json:"fields"` // map[name]value
}

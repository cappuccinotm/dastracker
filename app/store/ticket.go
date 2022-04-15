package store

// Content contains the data in the task itself.
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
	Variations Variations `json:"variations"`
}

// Patch updates ticket fields with given update values.
func (t *Ticket) Patch(upd Update) {
	t.Variations.Set(upd.ReceivedFrom.Tracker, Task{
		ID:      upd.ReceivedFrom.ID,
		Content: upd.Content,
	})
}

// Task represents a variation of the Ticket in a particular task tracker.
type Task struct {
	ID string `json:"ID"`
	Content
}

// Variations is a set of Ticket variations in task trackers.
type Variations map[string]Task

// Set sets the task in the ticket
func (v *Variations) Set(tracker string, task Task) {
	if *v == nil {
		*v = make(Variations)
	}
	(*v)[tracker] = task
}

// Get returns the task for the given tracker.
func (v Variations) Get(tracker string) (Task, bool) {
	if v == nil {
		return Task{}, false
	}
	task, ok := v[tracker]
	return task, ok
}

// Locators returns the task tracker and task ID of the ticket in each
// registered task tracker.
func (v Variations) Locators() []Locator {
	locators := make([]Locator, 0, len(v))
	for tracker, task := range v {
		locators = append(locators, Locator{
			Tracker: tracker,
			ID:      task.ID,
		})
	}
	return locators
}

// Update describes a ticket update.
type Update struct {
	ReceivedFrom Locator `json:"received_from"`
	URL          string  `json:"url"`
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

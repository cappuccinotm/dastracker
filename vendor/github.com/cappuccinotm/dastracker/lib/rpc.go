package lib

type Request struct {
	Ticket Ticket `json:"ticket"`
}

// Ticket describes an updated task representation in dastracker.
type Ticket struct {
	ID         string            `json:"id"`
	TrackerIDs map[string]string `json:"trackerIDs"`
	Title      string            `json:"title"`
	Body       string            `json:"body"`
	Fields     map[string]string `json:"fields"`
}

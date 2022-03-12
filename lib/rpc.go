package lib

// Request describes a request to tracker's action.
type Request struct {
	Ticket Ticket `json:"ticket"`
	Vars   Vars   `json:"vars"`
}

// Ticket describes an updated task representation in dastracker.
type Ticket struct {
	ID     string            `json:"id"`
	TaskID string            `json:"task_id"`
	Title  string            `json:"title"`
	Body   string            `json:"body"`
	Fields map[string]string `json:"fields"`
}

// Response describes possible return values of the tracker's action.
type Response struct {
	TaskID string `json:"task_id"` // optional, id of the created task in the tracker
}

// URLKey is a kludge to pass the URL of the webhook to the plugin.
const URLKey = "_url"

// SubscribeReq describes parameters of the subscription for task updates.
type SubscribeReq struct {
	Vars Vars `json:"vars"`
}

// WebhookURL returns the url of the webhook, provided by dastracker.
func (r SubscribeReq) WebhookURL() string { return r.Vars.Get(URLKey) }

// UnsubscribeReq describes parameters of the unsubscription from task updates.
type UnsubscribeReq struct {
	Vars Vars `json:"vars"`
}

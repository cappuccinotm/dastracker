package lib

// Request describes a request to tracker's action.
type Request struct {
	// TaskID in the target plugin, might be empty if the request is for creation
	TaskID string `json:"task_id"`
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

// SubscribeReq describes parameters of the subscription for task updates.
type SubscribeReq struct {
	WebhookURL string `json:"webhook_url"`
	Vars       Vars   `json:"vars"`
}

// SubscribeResp describes the response of the subscription request.
type SubscribeResp struct {
	TrackerRef string `json:"tracker_ref"`
}

// UnsubscribeReq describes parameters of the unsubscription from task updates.
type UnsubscribeReq struct {
	TrackerRef string `json:"tracker_ref"`
}

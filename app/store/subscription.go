package store

// Subscription describes configuration for subscription in trackers.
type Subscription struct {
	ID string `json:"id"`

	TrackerRef string `json:"tracker_ref"`

	TrackerName string `json:"tracker_name"`
	TriggerName string `json:"trigger_name"`

	BaseURL string `json:"base_url"`
}

// URL composes webhook URL from the subscription data.
func (s Subscription) URL() string { return s.BaseURL + "/" + s.ID }

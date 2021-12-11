package store

import "path"

// Webhook describes a webhook configuration for trackers, to not produce
// new webhooks after the app restarts and make sure that we can delete
// existing ones.
type Webhook struct {
	ID string `json:"id"`

	TrackerID string `json:"tracker_id"`

	TrackerName string `json:"tracker_name"`
	TriggerName string `json:"trigger_name"`
	BaseURL     string `json:"base_url"`
}

// URL composes URL from the webhook data.
func (w Webhook) URL() string {
	return path.Join(w.BaseURL, w.TrackerName, w.ID)
}

package store

import (
	"fmt"
	"net/url"
	"path"
)

// Webhook describes a webhook configuration for trackers, to not produce
// new webhooks after the app restarts and make sure that we can delete
// existing ones.
type Webhook struct {
	ID string `json:"id"`

	TrackerRef string `json:"tracker_ref"`

	TrackerName string `json:"tracker_name"`
	TriggerName string `json:"trigger_name"`
	BaseURL     string `json:"base_url"`
}

// URL composes URL from the webhook data.
func (w Webhook) URL() (string, error) {
	u, err := url.Parse(w.BaseURL)
	if err != nil {
		return "", fmt.Errorf("parse base url: %w", err)
	}
	u.Path = path.Join(u.Path, w.TrackerName, w.ID)
	return u.String(), nil
}

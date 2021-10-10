package webhook

import "net/http"

// Interface describes methods available to set up a new webhook.
type Interface interface {
	Set(tracker string, jobHash string, handler http.Handler) error
}

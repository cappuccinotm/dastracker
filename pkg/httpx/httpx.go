package httpx

import "net/http"

// RoundTripperFunc is and adapter to use ordinary functions as http.RoundTripper.
type RoundTripperFunc func(r *http.Request) (*http.Response, error)

// RoundTrip proxies call to the wrapped function.
func (f RoundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

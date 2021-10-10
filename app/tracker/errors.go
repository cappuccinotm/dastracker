package tracker

import "fmt"

// ErrInvalidConf indicates that there appeared an error in the tracker configuration.
type ErrInvalidConf string

// Error returns error message, wrapped by ErrInvalidConf.
func (e ErrInvalidConf) Error() string { return fmt.Sprintf("invalid configuration: %s", string(e)) }

// ErrUnsupportedMethod indicates that the requested method
// is not supported by the driver.
type ErrUnsupportedMethod string

// Error returns the string representation of the error.
func (e ErrUnsupportedMethod) Error() string { return fmt.Sprintf("unsupported method: %s", string(e)) }

// ErrUnexpectedStatus indicates that the remote server returned
// unexpected response status on the request.
type ErrUnexpectedStatus struct {
	RequestBody    []byte
	ResponseBody   []byte
	ResponseStatus int
}

// Error returns the string representation of the error.
func (e ErrUnexpectedStatus) Error() string {
	return fmt.Sprintf("unexpected status: %d", e.ResponseStatus)
}

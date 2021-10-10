package tracker

import (
	"context"
	"fmt"
	"strings"

	"github.com/cappuccinotm/dastracker/app/store"
)

// Interface defines methods for each tracker.
// All computable values from Vars must be already evaluated, thus
// the finite values are provided.
type Interface interface {
	Call(ctx context.Context, call Request) (Response, error)
	SetUpTrigger(ctx context.Context, vars Vars, cb Callback) error
}

// Callback invokes when some action that trigger describes has been appeared.
type Callback interface {
	Do(ctx context.Context, update store.Update) error
}

// CallbackFunc is an adapter to use ordinary functions as Callbacks.
type CallbackFunc func(context.Context, store.Update) error

// Do invokes the wrapped method with provided arguments.
func (f CallbackFunc) Do(ctx context.Context, upd store.Update) error { return f(ctx, upd) }

// Request describes a requests to tracker's action.
type Request struct {
	Method   string
	Vars     Vars
	TicketID string // might be empty, in case if task is not registered yet
}

// Response describes possible return values of the Interface.Call
type Response struct {
	ID string // id of the created task in the tracker.
}

// Vars is an alias for a map with variable values.
type Vars map[string]string

// Has returns true if variable with specified key is present.
func (v Vars) Has(key string) bool { _, ok := v[key]; return ok }

// Get returns the value of the variable.
func (v Vars) Get(name string) string { return v[name] }

// Set sets the value of the variable.
func (v *Vars) Set(name, val string) { (*v)[name] = val }

// List returns a list of strings from var's
// value parsed in form of "string1,string2,string3"
func (v Vars) List(s string) []string { return strings.Split(v.Get(s), ",") }

// ErrInvalidConf indicates that there appeared an error in the tracker configuration.
type ErrInvalidConf string

// Error returns error message, wrapped by ErrInvalidConf.
func (e ErrInvalidConf) Error() string { return fmt.Sprintf("invalid configuration: %s", e) }

// ErrUnsupportedMethod indicates that the requested method
// is not supported by the driver.
type ErrUnsupportedMethod string

// Error returns the string representation of the error.
func (e ErrUnsupportedMethod) Error() string { return fmt.Sprintf("unsupported method: %s", e) }

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

package lib

import "strings"

// SetUpTriggerResp is a response of set_up_trigger call to the remote RPC tracker driver.
type SetUpTriggerResp struct{}

// SetUpTriggerReq is a request to set up a trigger on the desired URL with specified variables.
type SetUpTriggerReq struct {
	URL  string
	Vars Vars
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

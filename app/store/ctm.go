package store

import (
	"strings"

	"github.com/cappuccinotm/dastracker/app/errs"
	"github.com/cappuccinotm/dastracker/lib"
)

// Sequence is a set of actions
type Sequence []Action

// Job describes a control sequence.
type Job struct {
	Name        string   `yaml:"name"`
	TriggerName string   `yaml:"on"`
	Actions     Sequence `yaml:"do"`
}

// Trigger describes an action, which has to appear to trigger
// a control flow.
type Trigger struct {
	Name    string   `yaml:"name"`
	Tracker string   `yaml:"in"`
	With    lib.Vars `yaml:"with"`
}

// Action describes a single call to the tracker's method.
type Action struct {
	Name     string   `yaml:"action"`
	With     lib.Vars `yaml:"with"`
	Detached bool     `yaml:"detached"` // means, that the action could be run asynchronously
}

// Path parses the action name to the tracker name and its method.
func (a Action) Path() (tracker, method string, err error) {
	dividerIdx := strings.IndexRune(a.Name, '/')
	if dividerIdx == -1 || dividerIdx == len(a.Name)-1 || dividerIdx == 0 {
		return "", "", errs.ErrMethodParseFailed(a.Name)
	}

	return a.Name[:dividerIdx], a.Name[dividerIdx+1:], nil
}

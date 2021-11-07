// Package flow provides methods and structures representing each element
// of the control flow tree and some common methods for them.
package flow

import (
	"github.com/cappuccinotm/dastracker/app/store"
	"strings"
)

// Config represents a configuration tree.
type Config struct {
	Trackers []Tracker `yaml:"trackers"`
	Jobs     []Job     `yaml:"jobs"`
}

// Tracker describes a connection parameters to the certain tracker.
type Tracker struct {
	Name   string     `yaml:"name"`
	Driver string     `yaml:"driver"`
	With   store.Vars `yaml:"with"`
}

// Sequence is a set of actions
type Sequence []Action

// Job describes a control sequence.
type Job struct {
	Name    string   `yaml:"name"`
	Trigger Trigger  `yaml:"on"`
	Actions Sequence `yaml:"do"`
}

// Trigger describes an action, which has to appear to trigger
// a control flow.
type Trigger struct {
	Tracker string     `yaml:"tracker"`
	With    store.Vars `yaml:"with"`
}

// Action describes a single call to the tracker's method.
type Action struct {
	Name     string     `yaml:"action"`
	With     store.Vars `yaml:"with"`
	Detached bool       `yaml:"detached"` // means, that the action could be run asynchronously
}

// Path parses the action name to the tracker name and its method.
func (a Action) Path() (tracker, method string) {
	dividerIdx := strings.IndexRune(a.Name, '/')
	return a.Name[:dividerIdx], a.Name[dividerIdx+1:]
}

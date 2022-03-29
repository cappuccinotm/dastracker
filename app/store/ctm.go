package store

import (
	"strings"

	"fmt"
	"github.com/cappuccinotm/dastracker/app/errs"
	"github.com/cappuccinotm/dastracker/lib"
	"gopkg.in/yaml.v3"
)

// Sequence is a set of actions
type Sequence []Step

// UnmarshalYAML implements the yaml.Unmarshaler interface
func (s *Sequence) UnmarshalYAML(node *yaml.Node) error {
	var metadata []struct {
		If     string `yaml:"if"`
		Action string `yaml:"action"`
	}
	var payloads []yaml.Node
	if err := node.Decode(&metadata); err != nil {
		return fmt.Errorf("unmarshal metas: %w", err)
	}
	if err := node.Decode(&payloads); err != nil {
		return fmt.Errorf("unmarshal metas: %w", err)
	}
	for idx, meta := range metadata {
		var step Step
		switch {
		case meta.If != "":
			i := If{}
			if err := payloads[idx].Decode(&i); err != nil {
				return fmt.Errorf("unmarshal 'if' payload: %w", err)
			}
			step = i
		case meta.Action != "":
			a := Action{}
			if err := payloads[idx].Decode(&a); err != nil {
				return fmt.Errorf("unmarshal action payload: %w", err)
			}
			step = a
		default:
			return fmt.Errorf("sequence contains invalid step: %s", node.Value)
		}
		*s = append(*s, step)
	}
	return nil
}

// Step is a single step in a sequence.
type Step interface{}

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

// If describes a conditional flow.
type If struct {
	Condition string   `yaml:"if"`
	Actions   Sequence `yaml:"do"`
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

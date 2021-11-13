package store

import (
	"bytes"
	"fmt"
	"gopkg.in/yaml.v3"
	"os"
	"strings"
	"text/template"
)

// VarsFromMap makes a new instance of Vars, not evaluated yet but filled
// with the provided map.
func VarsFromMap(m map[string]string) Vars { return Vars{vals: m} }

// Vars is an alias for a map with variable values.
type Vars struct {
	vals      map[string]string
	evaluated bool
}

// UnmarshalYAML implements unmarshaler in order to parse the map of values.
func (v *Vars) UnmarshalYAML(value *yaml.Node) error {
	return value.Decode(&v.vals)
}

// Has returns true if variable with specified key is present.
func (v *Vars) Has(key string) bool {
	if v.vals == nil {
		v.vals = map[string]string{}
	}
	_, ok := v.vals[key]
	return ok
}

// Get returns the value of the variable.
func (v Vars) Get(name string) string {
	if v.vals == nil {
		v.vals = map[string]string{}
	}
	return v.vals[name]
}

// Set sets the value of the variable.
func (v *Vars) Set(name, val string) {
	if v.vals == nil {
		v.vals = map[string]string{}
	}
	v.vals[name] = val
}

// List returns a list of strings from var's
// value parsed in form of "string1,string2,string3"
func (v Vars) List(s string) []string { return strings.Split(v.Get(s), ",") }

// Evaluated returns true if vars contains the set of already evaluated values.
func (v Vars) Evaluated() bool { return v.evaluated }

// Evaluate evaluates the final values of each variable.
func (v Vars) Evaluate(upd Update) (Vars, error) {
	if len(v.vals) == 0 {
		return Vars{evaluated: true}, nil
	}

	res := Vars{vals: map[string]string{}, evaluated: true}
	for key, vv := range v.vals {
		tmpl, err := template.New("").Funcs(funcs).Parse(vv)
		if err != nil {
			return Vars{}, fmt.Errorf("parse %q variable: %w", key, err)
		}

		buf := &bytes.Buffer{}
		if err = tmpl.Execute(buf, upd); err != nil {
			return Vars{}, fmt.Errorf("evaluate the value of the %q variable: %w", key, err)
		}

		res.vals[key] = buf.String()
	}

	return res, nil
}

// Equal returns true if two sets of variables represent the same one.
// Note: two sets of variables with different Evaluated state are considered
// to be equal, so the Evaluated state, in case if important, must be checked
// separately.
func (v Vars) Equal(oth Vars) bool {
	if len(v.vals) != len(oth.vals) {
		return false
	}

	for key, val := range v.vals {
		othVal, present := oth.vals[key]
		if !present || val != othVal {
			return false
		}
	}

	return true
}

// map of functions to parse from the config file
var funcs = map[string]interface{}{
	"env": os.Getenv,
	"keys": func(s map[string]string) []string {
		res := make([]string, 0, len(s))
		for k := range s {
			res = append(res, k)
		}
		return res
	},
	"values": func(s map[string]string) []string {
		res := make([]string, 0, len(s))
		for _, v := range s {
			res = append(res, v)
		}
		return res
	},
	"seq": func(s []string) string {
		return strings.Join(s, ",")
	},
}

// EvaluatedVarsFromMap does the same as VarsFromMap,
// but also marks vars as evaluated. Used in tests.
func EvaluatedVarsFromMap(m map[string]string) Vars {
	v := VarsFromMap(m)
	v.evaluated = true
	return v
}

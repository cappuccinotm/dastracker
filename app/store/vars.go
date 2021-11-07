package store

import (
	"bytes"
	"fmt"
	"gopkg.in/yaml.v3"
	"os"
	"strings"
	"text/template"
)

// Vars is an alias for a map with variable values.
type Vars struct {
	vals      map[string]string
	evaluated bool
}

// UnmarshalYAML implements unmarshaler in order to parse the map of
func (v *Vars) UnmarshalYAML(value *yaml.Node) error {
	return value.Decode(&v.vals)
}

// Has returns true if variable with specified key is present.
func (v *Vars) Has(key string) bool { _, ok := v.vals[key]; return ok }

// Get returns the value of the variable.
func (v Vars) Get(name string) string { return v.vals[name] }

// Set sets the value of the variable.
func (v *Vars) Set(name, val string) { v.vals[name] = val }

// List returns a list of strings from var's
// value parsed in form of "string1,string2,string3"
func (v Vars) List(s string) []string { return strings.Split(v.Get(s), ",") }

// Evaluated returns true if vars contains the set of already evaluated values.
func (v Vars) Evaluated() bool { return v.evaluated }

// Evaluate evaluates the final values of each variable.
func (v Vars) Evaluate(upd Update) (Vars, error) {
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

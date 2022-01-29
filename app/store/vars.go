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
type Vars map[string]string

// UnmarshalYAML implements unmarshaler in order to parse the map of values.
func (v *Vars) UnmarshalYAML(value *yaml.Node) error {
	m := map[string]string{}
	if err := value.Decode(&m); err != nil {
		return err
	}
	*v = m
	return nil
}

// Has returns true if variable with specified key is present.
func (v *Vars) Has(key string) bool {
	if *v == nil {
		*v = map[string]string{}
	}
	_, ok := (*v)[key]
	return ok
}

// Get returns the value of the variable.
func (v Vars) Get(name string) string { return v[name] }

// Set sets the value of the variable.
func (v *Vars) Set(name, val string) {
	if *v == nil {
		*v = map[string]string{}
	}
	(*v)[name] = val
}

// List returns a list of strings from var's
// value parsed in form of "string1,string2,string3"
func (v Vars) List(s string) []string { return strings.Split(v.Get(s), ",") }

type evTmpl struct{ Update Update }

// Evaluate evaluates the final values of each variable.
func (v Vars) Evaluate(upd Update) (Vars, error) {
	if len(v) == 0 {
		return nil, nil
	}

	res := Vars(map[string]string{})
	for key, vv := range v {
		tmpl, err := template.New("").Funcs(funcs).Parse(vv)
		if err != nil {
			return Vars{}, fmt.Errorf("parse %q variable: %w", key, err)
		}

		buf := &bytes.Buffer{}
		if err = tmpl.Execute(buf, evTmpl{Update: upd}); err != nil {
			return Vars{}, fmt.Errorf("evaluate the value of the %q variable: %w", key, err)
		}

		res[key] = buf.String()
	}

	return res, nil
}

// Equal returns true if two sets of variables represent the same one.
// Note: two sets of variables with different Evaluated state are considered
// to be equal, so the Evaluated state, in case if important, must be checked
// separately.
func (v Vars) Equal(oth Vars) bool {
	if len(v) != len(oth) {
		return false
	}

	for key, val := range v {
		othVal, present := oth[key]
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

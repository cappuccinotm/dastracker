package tracker

import "strings"

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

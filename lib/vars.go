package lib

import (
	"strings"

	"fmt"
	"gopkg.in/yaml.v3"
	"reflect"
)

// Vars is an alias for a map with variable values.
type Vars map[string]string

func (v *Vars) LoadTo(src interface{}) error {
	if v == nil {
		return nil
	}

	val := reflect.ValueOf(src)
	if val.Kind() != reflect.Pointer {
		return fmt.Errorf("expected pointer, got %s", val.Kind())
	}

	val = val.Elem()

	if val.Kind() != reflect.Struct {
		return fmt.Errorf("pointer must be a reference to struct, got %s", val.Kind())
	}

	for fieldIdx := 0; fieldIdx < val.NumField(); fieldIdx++ {
		field := val.Field(fieldIdx)
		field.Set()
		// todo
	}

}

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

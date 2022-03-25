package store

import "github.com/cappuccinotm/dastracker/lib"

// Tracker describes a connection parameters to the certain tracker.
type Tracker struct {
	Name   string   `yaml:"name"`
	Driver string   `yaml:"driver"`
	With   lib.Vars `yaml:"with"`
}

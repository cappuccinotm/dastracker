package provider

import (
	"context"
	"github.com/cappuccinotm/dastracker/app/store"
)

// Static reads the configuration for continuous task management from
// the yaml file and keeps it in memory.
type Static struct {
	Trackers []Tracker       `yaml:"trackers"`
	Triggers []store.Trigger `yaml:"triggers"`
	Jobs     []store.Job     `yaml:"jobs"`
}

// Tracker describes a connection parameters to the certain tracker.
type Tracker struct {
	Name   string     `yaml:"name"`
	Driver string     `yaml:"driver"`
	With   store.Vars `yaml:"with"`
}

// GetJob returns the job by its name.
func (s *Static) GetJob(_ context.Context, name string) (store.Job, error) {
	panic("implement me")
}

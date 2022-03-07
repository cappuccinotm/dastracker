package flow

import (
	"context"
	"fmt"
	"github.com/cappuccinotm/dastracker/app/errs"
	"github.com/cappuccinotm/dastracker/app/store"
	"github.com/cappuccinotm/dastracker/lib"
	"gopkg.in/yaml.v3"
	"os"
)

// Static reads the configuration for continuous task management from
// the yaml file and keeps it in memory.
type Static struct {
	subscriptions map[string][]store.Job // map[triggerName]jobs
	config
}

type config struct {
	Trackers []Tracker       `yaml:"trackers"`
	Triggers []store.Trigger `yaml:"triggers"`
	Jobs     []store.Job     `yaml:"jobs"`
}

// Tracker describes a connection parameters to the certain tracker.
type Tracker struct {
	Name   string   `yaml:"name"`
	Driver string   `yaml:"driver"`
	With   lib.Vars `yaml:"with"`
}

// NewStatic makes new instance of Static flow provider.
func NewStatic(path string) (*Static, error) {
	bytes, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read configuration file: %w", err)
	}

	var cfg config

	if err = yaml.Unmarshal(bytes, &cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	if err = cfg.validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	svc := &Static{config: cfg, subscriptions: map[string][]store.Job{}}

	for _, job := range cfg.Jobs {
		svc.subscriptions[job.TriggerName] = append(svc.subscriptions[job.TriggerName], job)
	}

	return svc, nil
}

// ListTriggers returns the list of registered triggers with their variables.
func (s *Static) ListTriggers(_ context.Context) ([]store.Trigger, error) {
	return s.config.Triggers, nil
}

// ListTrackers returns the list of registered trackers with their configurations.
func (s *Static) ListTrackers(_ context.Context) ([]Tracker, error) {
	return s.Trackers, nil
}

// ListSubscribedJobs returns the jobs attached to the trigger
// by the name of the trigger.
func (s *Static) ListSubscribedJobs(_ context.Context, triggerName string) ([]store.Job, error) {
	jobs, triggerPresent := s.subscriptions[triggerName]
	if !triggerPresent {
		return nil, fmt.Errorf("trigger was not found: %w", errs.ErrNotFound)
	}
	return jobs, nil
}

func (c config) validate() error {
	trackers := map[string]struct{}{}
	for _, tracker := range c.Trackers {
		trackers[tracker.Name] = struct{}{}
	}

	triggers := map[string]struct{}{}
	for _, trigger := range c.Triggers {
		triggers[trigger.Name] = struct{}{}
		if _, trackerPresent := trackers[trigger.Tracker]; !trackerPresent {
			return fmt.Errorf("tracker %q, referred by trigger %q, is not registered: %w",
				trigger.Tracker, trigger.Name, errs.ErrNotFound)
		}
	}

	for _, job := range c.Jobs {
		if _, triggerPresent := triggers[job.TriggerName]; !triggerPresent {
			return fmt.Errorf("trigger %q, referred by job %q, is not registered: %w",
				job.TriggerName, job.Name, errs.ErrNotFound)
		}

		for _, act := range job.Actions {
			tracker, _, err := act.Path()
			if err != nil {
				return fmt.Errorf("invalid action path: %w", err)
			}
			if _, trackerPresent := trackers[tracker]; !trackerPresent {
				return fmt.Errorf("tracker %q, referred by action %q in job %q, is not registered: %w",
					tracker, act.Name, job.Name, errs.ErrNotFound)
			}
		}
	}

	return nil
}

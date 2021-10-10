package cmd

import (
	"os"
	"strings"
)

// Config describes a single config file.
type Config struct {
	Trackers []Tracker `yaml:"trackers"`
	Jobs     []Job     `yaml:"jobs"`
}

// Tracker describes a single task tracker and its connection.
type Tracker struct {
	Name   string            `yaml:"name"`
	Driver string            `yaml:"driver"`
	Vars   map[string]string `yaml:"with"`
}

// Job is a flow of actions which must happen when the desired
// conditions appeared.
type Job struct {
	Name    string   `yaml:"name"`
	On      Trigger  `yaml:"on"`
	Actions []Action `yaml:"do"`
}

// Trigger describes a change that must appear in order to trigger a job.
type Trigger struct {
	TrackerName string            `yaml:"in"`
	Vars        map[string]string `yaml:"with"`
}

// Action describes a single step in the job.
type Action struct {
	Method string            `yaml:"action"`
	Vars   map[string]string `yaml:"vars"`
}

// map of functions to parse from the config file
var funcs = map[string]interface{}{
	"env": os.Getenv,
	"values": func(s map[string]string) []string {
		var res []string
		for _, v := range s {
			res = append(res, v)
		}
		return res
	},
	"seq": func(s []string) string {
		return strings.Join(s, ",")
	},
}

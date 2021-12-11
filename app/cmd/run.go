package cmd

import (
	"time"
)

// Run starts a tracker listener.
type Run struct {
	ConfLocation string `short:"c" long:"config_location" env:"CONFIG_LOCATION" description:"location of the configuration file"`
	Store        struct {
		Type string `long:"type" env:"TYPE" choice:"bolt" description:"type of storage"`
		Bolt struct {
			Path    string        `long:"path" env:"PATH" default:"./var" description:"parent dir for bolt files"`
			Timeout time.Duration `long:"timeout" env:"TIMEOUT" default:"30s" description:"bolt timeout"`
		} `group:"bolt" namespace:"bolt" env-namespace:"BOLT"`
	} `group:"store" namespace:"store" env-namespace:"STORE"`
	Webhook struct {
		BaseURL string `long:"base_url" env:"BASE_URL" description:"base url for webhooks"`
		Addr    string `long:"addr" env:"ADDR" description:"local address to listen"`
	} `group:"webhook" namespace:"webhook" env-namespace:"WEBHOOK"`
}

// Execute runs the command
func (r Run) Execute(_ []string) error {
	// todo
	panic("implement me")
}

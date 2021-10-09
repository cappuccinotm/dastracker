package cmd

import "github.com/cappuccinotm/dastracker/app/tracker"

type Run struct {
	Github struct {
		Enabled      bool   `long:"enabled" env:"ENABLED" description:"is github access enabled"`
		AppID        string `long:"app_id" env:"APP_ID" description:"github application ID"`
		ClientSecret string
		// some other parameters
	} `group:"github" namespace:"github" env-namespace:"GITHUB"`
	Asana struct {
		Enabled bool `long:"enabled" env:"ENABLED" description:"is asana access enabled"`
		AppID   string
		// blah blah
	}
}

// Execute runs the command
func (r Run) Execute(args []string) error {
	tracker.NewGithub()
}

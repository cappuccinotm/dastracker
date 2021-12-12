package cmd

// RemoveWebhooks removes all webhooks described in the configuration file.
// Might be useful in case of hard shutdown (webhooks didn't turn off when the
// app was shut down).
type RemoveWebhooks struct {
	ConfLocation string `short:"c" long:"config_location" env:"CONFIG_LOCATION" description:"location of the configuration file"`
}

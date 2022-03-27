package main

import (
	"fmt"
	"log"
	"os"

	"github.com/cappuccinotm/dastracker/app/cmd"
	"github.com/cappuccinotm/dastracker/pkg/logx"
	"github.com/hashicorp/logutils"
	"github.com/jessevdk/go-flags"
)

// Opts describes cli commands, arguments and flags of the application.
type Opts struct {
	Run   cmd.Run `command:"run"`
	Debug bool    `long:"dbg" env:"DEBUG" description:"turn on debug mode"`
}

var version = "unknown"

func main() {
	fmt.Printf("dastracker, version: %s\n", version)

	var opts Opts
	p := flags.NewParser(&opts, flags.Default)
	p.CommandHandler = func(command flags.Commander, args []string) error {
		logger := setupLog(opts.Debug)

		// commands implement CommonOptionsCommander to allow passing set of extra options defined for all commands
		c := command.(cmd.CommonOptionsCommander)
		c.SetCommon(cmd.CommonOpts{
			Version: version,
			Logger:  logger,
		})

		if err := command.Execute(args); err != nil {
			logger.Printf("[ERROR] failed to execute command: %+v", err)
		}
		return nil
	}

	// after failure command does not return non-zero code
	if _, err := p.Parse(); err != nil {
		if flagsErr, ok := err.(*flags.Error); ok && flagsErr.Type == flags.ErrHelp {
			os.Exit(0)
		} else {
			os.Exit(1)
		}
	}
}

func setupLog(dbg bool) logx.Logger {
	filter := &logutils.LevelFilter{
		Levels:   []logutils.LogLevel{"DEBUG", "INFO", "WARN", "ERROR"},
		MinLevel: "INFO",
		Writer:   os.Stdout,
	}

	logFlags := log.Ldate | log.Ltime

	if dbg {
		logFlags = log.Ldate | log.Ltime | log.Lmicroseconds | log.Lshortfile
		filter.MinLevel = "DEBUG"
	}

	return logx.Std(log.New(filter, "", logFlags), []string{"DEBUG", "INFO", "WARN", "ERROR"})
}

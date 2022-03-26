package cmd

import (
	"github.com/cappuccinotm/dastracker/pkg/logx"
)

// CommonOptionsCommander extends flags.Commander with SetCommon
// All commands should implement this interfaces
type CommonOptionsCommander interface {
	SetCommon(commonOpts CommonOpts)
	Execute(args []string) error
}

// CommonOpts sets externally from main, shared across all commands
type CommonOpts struct {
	Version string
	Logger  logx.Logger
}

// SetCommon satisfies CommonOptionsCommander interface and sets common option fields
// The method called by main for each command
func (c *CommonOpts) SetCommon(opts CommonOpts) {
	c.Version = opts.Version
	c.Logger = opts.Logger
}

/*
Copyright 2018 The Doctl Authors All rights reserved.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package commands

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"sync/atomic"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/internal/ui"

	"github.com/spf13/cobra"
)

// Command is a wrapper around cobra.Command that adds doctl specific
// functionality.
type Command struct {
	*cobra.Command

	*cobra.Group

	fmtCols []string

	childCommands []*Command

	// overrideNS specifies a namespace to use in config. Set with overrideCmdNS.
	overrideNS string

	// noDefaultSuccess opts out of the default closing line.
	noDefaultSuccess bool
}

// AddCommand adds child commands and adds child commands for cobra as well.
func (c *Command) AddCommand(commands ...*Command) {
	c.childCommands = append(c.childCommands, commands...)
	for _, cmd := range commands {
		c.Command.AddCommand(cmd.Command)
	}
}

// ChildCommands returns the child commands.
func (c *Command) ChildCommands() []*Command {
	return c.childCommands
}

type ValidArgsFunc func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective)

// AddValidArgsFunc sets the function to run for dynamic completions. It errors
// if ValidArgs is already set, since the two are mutually exclusive.
func (c *Command) AddValidArgsFunc(fn ValidArgsFunc) error {
	if len(c.Command.ValidArgs) == 0 {
		c.Command.ValidArgsFunction = fn
		return nil
	}
	return errors.New("unable to add ValidArgsFunction when ValidArgs is already set")
}

// defaultSuccess closes a command that reports nothing of its own.
const defaultSuccess = "Command completed successfully"

// owesClosingLine reports whether doctl still has to say how the command went.
//
// wrote is whether the command reported anything itself. Only a person is
// told, and only when nothing else was: the line is chrome rather than a
// result, so a redirected stderr keeps the silence a script was written
// against.
func (c *Command) owesClosingLine(wrote bool, env ui.Env) bool {
	return !wrote && !reportedOutcome && !c.noDefaultSuccess && env.ErrTTY
}

// reportedOutcome records that doctl has already said how the command went,
// through a notice or the line a wait leaves behind. A warning deliberately
// does not set it: it says something went oddly, not that the command finished,
// so a command that only warns still owes a closing line.
var reportedOutcome bool

// reportingWriter notes whether anything reached stdout, which is what tells a
// command that printed a result from one the user heard nothing from.
//
// A waiter writes from the goroutine it polls on, so the flag is atomic.
type reportingWriter struct {
	out   io.Writer
	wrote atomic.Bool
}

func (w *reportingWriter) Write(p []byte) (int, error) {
	n, err := w.out.Write(p)
	if n > 0 {
		w.wrote.Store(true)
	}

	return n, err
}

// CmdBuilder builds a new command.
func CmdBuilder(parent *Command, cr CmdRunner, cliText, shortdesc string, longdesc string, out io.Writer, options ...cmdOption) *Command {
	return cmdBuilderWithInit(parent, cr, cliText, shortdesc, longdesc, out, true, options...)
}

func cmdBuilderWithInit(parent *Command, cr CmdRunner, cliText, shortdesc string, longdesc string, out io.Writer, initCmd bool, options ...cmdOption) *Command {
	cc := &cobra.Command{
		Use:   cliText,
		Short: shortdesc,
		Long:  longdesc,
	}

	c := &Command{Command: cc}

	if parent != nil {
		parent.AddCommand(c)
	}

	for _, co := range options {
		co(c)
	}

	// Aggregated so every missing or invalid flag is reported together, before
	// Cobra's bare required-flag check and before the handler executes.
	c.Command.PreRunE = func(cmd *cobra.Command, args []string) error {
		return validateCommandFlags(cmd)
	}

	// Defined after the options are applied so their changes are visible here.
	c.Command.Run = func(cmd *cobra.Command, args []string) {
		cfg, err := NewCmdConfig(
			cmdNS(c),
			&doctl.LiveConfig{},
			out,
			args,
			initCmd,
		)
		checkErr(err)

		// A command writes to stdout through its own writer or through the
		// shared charm template output, so both are pointed at one reporter
		// and neither is missed. A command given a writer of its own, as tests
		// do, gets a reporter of its own.
		reported := stdoutSink
		if reported == nil || cfg.Out != io.Writer(Writer) {
			reported = &reportingWriter{out: cfg.Out}
		}
		reported.wrote.Store(false)

		cfg.Out = reported
		cfg.Command = cmd
		reportedOutcome = false

		err = cr(cfg)
		checkErr(err)

		if c.owesClosingLine(reported.wrote.Load(), uiEnv()) {
			reportSuccess(defaultSuccess)
		}
	}

	if cols := c.fmtCols; cols != nil {
		formatHelp := fmt.Sprintf("Columns for output in a comma-separated list. Possible values: `%s`.",
			strings.Join(cols, "`"+", "+"`"))
		AddStringFlag(c, doctl.ArgFormat, "", "", formatHelp)
		AddBoolFlag(c, doctl.ArgNoHeader, "", false, "Return raw data with no headers")
	}

	return c

}

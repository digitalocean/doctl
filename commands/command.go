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

	"github.com/digitalocean/doctl"

	"github.com/spf13/cobra"
)

// Command is a wrapper around cobra.Command that adds doctl specific
// functionality.
type Command struct {
	*cobra.Command

	fmtCols []string

	childCommands []*Command
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

// AddValidArgsFunc sets the function to run for dynamic completions
// ValidArgsFunc and ValidArgs are mutually exclusive. This function will
// return an error if ValidArgs is already set.
func (c *Command) AddValidArgsFunc(fn ValidArgsFunc) error {
	if len(c.Command.ValidArgs) == 0 {
		c.Command.ValidArgsFunction = fn
		return nil
	}
	return errors.New("unable to add ValidArgsFunction when ValidArgs is already set")
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
		Run: func(cmd *cobra.Command, args []string) {
			c, err := NewCmdConfig(
				cmdNS(cmd),
				&doctl.LiveConfig{},
				out,
				args,
				initCmd,
			)
			checkErr(err)

			err = cr(c)
			checkErr(err)
		},
	}

	c := &Command{Command: cc}

	if parent != nil {
		parent.AddCommand(c)
	}

	for _, co := range options {
		co(c)
	}

	if cols := c.fmtCols; cols != nil {
		formatHelp := fmt.Sprintf("Columns for output in a comma-separated list. Possible values: `%s`",
			strings.Join(cols, "`"+", "+"`"))
		AddStringFlag(c, doctl.ArgFormat, "", "", formatHelp)
		AddBoolFlag(c, doctl.ArgNoHeader, "", false, "Return raw data with no headers")
	}

	return c

}

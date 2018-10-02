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

import "github.com/spf13/cobra"

// Command is a wrapper around cobra.Command that adds doctl specific
// functionality.
type Command struct {
	*cobra.Command

	// DocCategories are the documentation categories this command belongs to.
	DocCategories []string

	fmtCols []string

	childCommands []*Command
	IsIndex       bool
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

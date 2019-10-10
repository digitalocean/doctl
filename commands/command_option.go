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

import "github.com/digitalocean/doctl/commands/displayers"

// cmdOption allow configuration of a command.
type cmdOption func(*Command)

// aliasOpt adds aliases for a command.
func aliasOpt(aliases ...string) cmdOption {
	return func(c *Command) {
		if c.Aliases == nil {
			c.Aliases = []string{}
		}

		c.Aliases = append(c.Aliases, aliases...)
	}
}

// displayerType sets the columns for display for a command.
func displayerType(d displayers.Displayable) cmdOption {
	return func(c *Command) {
		c.fmtCols = d.Cols()
	}
}

// hiddenCmd make a command hidden.
func hiddenCmd() cmdOption {
	return func(c *Command) {
		c.Hidden = true
	}
}

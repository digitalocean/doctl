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

// overrideCmdNS specifies a namespace to use in config overriding the
// normal usage of the parent command's name. This is useful in cases
// where deeply nested subcommands have conflicting names. See uptime_alerts.go
// for example usage.
func overrideCmdNS(ns string) cmdOption {
	return func(c *Command) {
		c.overrideNS = ns
	}
}

// agentPrettyErrors enables styled error cards for agent subcommands.
func agentPrettyErrors() cmdOption {
	return func(c *Command) {
		c.prettyAgentErrors = true
	}
}

// agentsNS keeps viper keys under agents.* after the primary command became
// `doctl open-harness-runtime` (with agent/agents aliases). Spread into CmdBuilder options.
func agentsNS(opts ...cmdOption) []cmdOption {
	return append([]cmdOption{agentPrettyErrors(), overrideCmdNS("agents")}, opts...)
}

// agentSubNS is for nested agent groups (config / triggers / checkpoint) that
// need both pretty errors and a custom viper namespace.
func agentSubNS(ns string, opts ...cmdOption) []cmdOption {
	return append([]cmdOption{agentPrettyErrors(), overrideCmdNS(ns)}, opts...)
}

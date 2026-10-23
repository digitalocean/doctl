/*
Copyright 2026 The Doctl Authors All rights reserved.
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

// Command gen-command-tree prints the doctl command tree as JSON: every
// command, including hidden and deprecated ones, with its help text, aliases
// and flags. Tools that document doctl can read it instead of importing the
// commands package.
//
//	go run ./scripts/gen-command-tree > commands.json
package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/commands"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type tree struct {
	Version string  `json:"version"`
	Root    command `json:"root"`
}

type command struct {
	Name       string   `json:"name"`
	Path       string   `json:"path"`
	Use        string   `json:"use"`
	UseLine    string   `json:"use_line"`
	Short      string   `json:"short,omitempty"`
	Long       string   `json:"long,omitempty"`
	Example    string   `json:"example,omitempty"`
	Aliases    []string `json:"aliases,omitempty"`
	Deprecated string   `json:"deprecated,omitempty"`
	// Available is false for hidden and deprecated commands, which are left
	// out of help.
	Available bool `json:"available"`
	// HelpTopic is true for commands that only carry help text.
	HelpTopic       bool      `json:"help_topic,omitempty"`
	LocalFlags      []flag    `json:"local_flags,omitempty"`
	InheritedFlags  []flag    `json:"inherited_flags,omitempty"`
	PersistentFlags []flag    `json:"persistent_flags,omitempty"`
	Commands        []command `json:"commands,omitempty"`
}

type flag struct {
	Name        string              `json:"name"`
	Shorthand   string              `json:"shorthand,omitempty"`
	Usage       string              `json:"usage"`
	Type        string              `json:"type"`
	Default     string              `json:"default"`
	Hidden      bool                `json:"hidden,omitempty"`
	Annotations map[string][]string `json:"annotations,omitempty"`
}

func main() {
	// Cobra adds `doctl completion`, `doctl help` and the --help flags only
	// when the command runs.
	commands.DoitCmd.InitDefaultCompletionCmd()
	commands.DoitCmd.InitDefaultHelpCmd()

	t := tree{
		Version: doctl.DoitVersion.String(),
		Root:    export(commands.DoitCmd.Command),
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(t); err != nil {
		fmt.Fprintf(os.Stderr, "gen-command-tree: %v\n", err)
		os.Exit(1)
	}
}

func export(c *cobra.Command) command {
	c.InitDefaultHelpFlag()

	// Cobra merges inherited flags into a command lazily, and the usage line
	// depends on them, so they are merged before anything else is read.
	inherited := flags(c.InheritedFlags())

	out := command{
		Name:            c.Name(),
		Path:            c.CommandPath(),
		Use:             c.Use,
		UseLine:         c.UseLine(),
		Short:           c.Short,
		Long:            c.Long,
		Example:         c.Example,
		Aliases:         c.Aliases,
		Deprecated:      c.Deprecated,
		Available:       c.IsAvailableCommand(),
		HelpTopic:       c.IsAdditionalHelpTopicCommand(),
		LocalFlags:      flags(c.LocalFlags()),
		InheritedFlags:  inherited,
		PersistentFlags: flags(c.PersistentFlags()),
	}
	for _, sub := range c.Commands() {
		out.Commands = append(out.Commands, export(sub))
	}
	return out
}

func flags(fs *pflag.FlagSet) []flag {
	var out []flag
	fs.VisitAll(func(f *pflag.Flag) {
		out = append(out, flag{
			Name:        f.Name,
			Shorthand:   f.Shorthand,
			Usage:       f.Usage,
			Type:        f.Value.Type(),
			Default:     f.DefValue,
			Hidden:      f.Hidden,
			Annotations: f.Annotations,
		})
	})
	return out
}

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
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/digitalocean/doctl/commands/displayers"
	"github.com/digitalocean/doctl/pluginhost"
	"github.com/spf13/cobra"
)

// Plugin creates the plugin commands hierarchy.
func Plugin() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:        "plugin",
			Short:      "Display commands to manage plugins",
			Long:       "plugin is used to access plugin commands",
			Aliases:    []string{"p"},
			Hidden:     true,
			Deprecated: "This command will be removed in a future version of doctl",
		},
	}

	CmdBuilder(cmd, RunPluginList, "list", "List plugins", "List plugins", Writer,
		aliasOpt("ls"))

	CmdBuilder(cmd, RunPluginRun, "run", "Run a plugin", "Run a plugin", Writer)

	return cmd
}

// RunPluginRun is a command for running a plugin.
func RunPluginRun(c *CmdConfig) error {
	if len(c.Args) == 0 {
		return fmt.Errorf("missing plugin name")
	}

	plugs, err := searchPlugins()
	if err != nil {
		return err
	}

	var selectedPlugin *displayers.PlugDesc
	for i, p := range plugs {
		if p.Name == c.Args[0] {
			selectedPlugin = &plugs[i]
		}
	}

	if selectedPlugin == nil {
		return fmt.Errorf("unknown plugin %q", c.Args[0])
	}

	var pluginArgs []string
	if len(c.Args) > 1 {
		pluginArgs = c.Args[1:]
	}

	host, err := pluginhost.NewHost(selectedPlugin.Path)
	if err != nil {
		return err
	}

	var method string
	var methodArgs []string

	switch l := len(pluginArgs); {
	case l == 0:
		method = "Default"
	case l == 1:
		method = pluginArgs[0]
	default:
		method = pluginArgs[0]
		methodArgs = pluginArgs[1:]
	}

	if len(pluginArgs) > 1 {
		methodArgs = pluginArgs[1:]
	}

	results, err := host.Call(selectedPlugin.Name+"."+strings.Title(method), methodArgs...)
	if err != nil {
		return err
	}

	fmt.Fprintln(c.Out, results)
	return nil
}

// RunPluginList is a command for listing available plugins.
func RunPluginList(c *CmdConfig) error {
	plugs, err := searchPlugins()
	if err != nil {
		return err
	}

	item := &displayers.Plugin{Plugins: plugs}
	return c.Display(item)
}

func searchPlugins() ([]displayers.PlugDesc, error) {
	envPath := os.Getenv("PATH")
	paths := strings.Split(envPath, string(os.PathListSeparator))

	var plugs []displayers.PlugDesc

	for _, p := range paths {
		matches, err := filepath.Glob(filepath.Join(p, "doit-provider-*"))
		if err != nil {
			return nil, err
		}

		for _, pluginPath := range matches {
			name := pluginName(pluginPath)
			plugs = append(plugs, displayers.PlugDesc{Path: pluginPath, Name: name})
		}
	}

	return plugs, nil
}

func pluginName(p string) string {
	base := filepath.Base(p)
	return strings.TrimPrefix(base, "doit-provider-")
}

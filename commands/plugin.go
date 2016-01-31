package commands

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/bryanl/doit"
	"github.com/bryanl/doit/pluginhost"
	"github.com/spf13/cobra"
)

// Plugin creates the plugin commands heirarchy.
func Plugin() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "plugin",
		Short:   "plugin commands",
		Long:    "plugin is used to access plugin commands",
		Aliases: []string{"p"},
	}

	cmdBuilder(cmd, RunPluginList, "list", "list plugins", writer,
		aliasOpt("ls"))

	cmdBuilder(cmd, RunPluginRun, "run", "run plugin", writer)

	return cmd
}

// RunPluginRun is a command for running a plugin.
func RunPluginRun(ns string, config doit.Config, out io.Writer, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("missing plugin name")
	}

	plugs, err := searchPlugins()
	if err != nil {
		return err
	}

	var selectedPlugin *plugDesc
	for i, p := range plugs {
		if p.Name == args[0] {
			selectedPlugin = &plugs[i]
		}
	}

	if selectedPlugin == nil {
		return fmt.Errorf("unknown plugin %q", args[0])
	}

	var pluginArgs []string
	if len(args) > 1 {
		pluginArgs = args[1:]
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

	fmt.Fprintln(out, results)
	return nil
}

// RunPluginList is a command for listing available plugins.
func RunPluginList(ns string, config doit.Config, out io.Writer, args []string) error {
	plugs, err := searchPlugins()
	if err != nil {
		return err
	}

	dc := &displayer{
		ns:     ns,
		config: config,
		item:   &plugin{plugins: plugs},
		out:    out,
	}

	return dc.Display()
}

type plugDesc struct {
	Path string `json:"path"`
	Name string `json:"name"`
}

func searchPlugins() ([]plugDesc, error) {
	envPath := os.Getenv("PATH")
	paths := strings.Split(envPath, string(os.PathListSeparator))

	var plugs []plugDesc

	for _, p := range paths {
		matches, err := filepath.Glob(filepath.Join(p, "doit-provider-*"))
		if err != nil {
			return nil, err
		}

		for _, pluginPath := range matches {
			name := pluginName(pluginPath)
			plugs = append(plugs, plugDesc{Path: pluginPath, Name: name})
		}
	}

	return plugs, nil
}

func pluginName(p string) string {
	base := filepath.Base(p)
	return strings.TrimPrefix(base, "doit-provider-")
}

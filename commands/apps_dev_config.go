package commands

import (
	"errors"
	"fmt"
	"strings"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/commands/charm/template"
	"github.com/digitalocean/doctl/internal/apps/workspace"
	"github.com/spf13/cobra"
)

const (
	// DefaultDevConfigFile is the name of the default dev configuration file.
	DefaultDevConfigFile = "dev-config.yaml"
)

// AppsDevConfig creates the apps dev config command subtree.
func AppsDevConfig() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:     "config",
			Aliases: []string{"c"},
			Short:   "Display commands for working with app platform local development configuration settings.",
			Long: fmt.Sprintf(`[BETA] Display commands for working with app platform local development configuration settings.

Configuration Format:
%s

`, workspace.SampleDevConfigFile),
		},
	}

	set := CmdBuilder(
		cmd,
		RunAppsDevConfigSet,
		"set KEY=VALUE...",
		"Set a value in the local development configuration settings.",
		fmt.Sprintf(`Set a value in the local development configuration settings.

KEY is the name of a configuration option, for example: spec=/path/to/app.yaml
Nested component KEYs can also be set, for example: components.my-component.build_command="go build ."

Multiple KEY=VALUE pairs may be specified separated by a space.

Configuration Format:
%s

`, workspace.SampleDevConfigFile),
		Writer,
	)

	AddStringFlag(
		set, doctl.ArgAppDevConfig,
		"", "",
		`Path to the app dev config.`,
	)

	unset := CmdBuilder(
		cmd,
		RunAppsDevConfigUnset,
		"unset KEY...",
		"Unset a value in the local development configuration settings.",
		fmt.Sprintf(`Unset a value in the local development configuration settings.

KEY is the name of a configuration option to unset, for example: spec
Nested component KEYs can also be unset, for example: components.my-component.build_command

Multiple KEYs may be specified separated by a space.

Configuration Format:
%s

`, workspace.SampleDevConfigFile),
		Writer,
	)

	AddStringFlag(
		unset, doctl.ArgAppDevConfig,
		"", "",
		`Path to the app dev config.`,
	)

	return cmd
}

// RunAppsDevConfigSet runs the set configuration command.
func RunAppsDevConfigSet(c *CmdConfig) error {
	if len(c.Args) == 0 {
		return errors.New("you must provide at least one argument")
	}

	ws, err := appDevWorkspace(c)
	if err != nil {
		return fmt.Errorf("preparing workspace: %w", err)
	}

	for _, arg := range c.Args {
		split := strings.SplitN(arg, "=", 2)
		if len(split) != 2 {
			return errors.New("unexpected arg: " + arg)
		}
		err := ws.Config.Set(split[0], split[1])
		if err != nil {
			return err
		}
		template.Print(`{{success checkmark}} set new value for {{highlight .}}{{nl}}`, split[0])
	}

	err = ws.Config.Write()
	if err != nil {
		return err
	}

	return nil
}

// RunAppsDevConfigUnset runs the unset configuration command.
func RunAppsDevConfigUnset(c *CmdConfig) error {
	if len(c.Args) == 0 {
		return errors.New("you must provide at least one argument")
	}

	ws, err := appDevWorkspace(c)
	if err != nil {
		return fmt.Errorf("preparing workspace: %w", err)
	}

	for _, arg := range c.Args {
		err = ws.Config.Set(arg, "")
		if err != nil {
			return err
		}

		template.Print(`{{success checkmark}} unset {{highlight .}}{{nl}}`, arg)
	}

	err = ws.Config.Write()
	if err != nil {
		return err
	}

	return nil
}

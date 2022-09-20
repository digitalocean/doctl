package commands

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/commands/charm/template"
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
			Short:   "Display commands for working with app platform local development configuration files.",
			Long:    `[BETA] Display commands for working with app platform local development configuration files.`,
		},
	}

	set := CmdBuilder(
		cmd,
		RunAppsDevConfigSet,
		"set KEY=VALUE...",
		"Set dev configuration settings.",
		"Set dev configuration settings for a build.",
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
		"Unset dev configuration settings.",
		"Unset dev configuration settings for a build.",
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

var errNoGitRepo = errors.New("no git repository found")

// findTopLevelGitDir ...
func findTopLevelGitDir(workingDir string) (string, error) {
	dir, err := filepath.Abs(workingDir)
	if err != nil {
		return "", err
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", errors.New("no git repository found")
		}
		dir = parent
	}
}

package commands

import (
	"bytes"
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/commands/charm/template"
	"github.com/digitalocean/doctl/internal/apps/config"
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
			Long:    `Display commands for working with app platform local development configuration files.`,
		},
	}

	set := CmdBuilder(
		cmd,
		RunAppsDevConfigSet,
		"set KEY=VALUE...",
		"Set dev configuration settings.",
		"Set dev configuration settings for a build.",
		// 		fmt.Sprintf(`Set dev configuration settings for a build.

		// Valid Keys: %s
		// `, config.ValidAppDevKeys()),
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
		// 		fmt.Sprintf(`Unset dev configuration settings for a build.

		// Valid Keys: %s
		// `, config.ValidAppDevKeys()),
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

	dev, err := newAppDevConfig(c)
	if err != nil {
		return err
	}

	for _, arg := range c.Args {
		split := strings.SplitN(arg, "=", 2)
		if len(split) != 2 {
			return errors.New("unexpected arg: " + arg)
		}
		err := dev.Set(split[0], split[1])
		if err != nil {
			return err
		}
		template.Print(`{{success checkmark}} set new value for {{highlight .}}{{nl}}`, split[0])
	}

	err = dev.WriteConfig()
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

	dev, err := newAppDevConfig(c)
	if err != nil {
		return err
	}

	for _, arg := range c.Args {
		err = dev.Set(arg, "")
		if err != nil {
			return err
		}

		template.Print(`{{success checkmark}} unset {{highlight .}}{{nl}}`, arg)
	}

	err = dev.WriteConfig()
	if err != nil {
		return err
	}

	return nil
}

func newAppDevConfig(cmdConfig *CmdConfig) (*config.AppDev, error) {
	devConfigFilePath, err := cmdConfig.Doit.GetString(cmdConfig.NS, doctl.ArgAppDevConfig)
	if err != nil {
		return nil, err
	}
	return config.New(devConfigFilePath)
}

func ensureStringInFile(file string, val string) error {
	if _, err := os.Stat(file); errors.Is(err, os.ErrNotExist) {
		f, err := os.OpenFile(
			file,
			os.O_APPEND|os.O_CREATE|os.O_WRONLY,
			0644,
		)
		if err != nil {
			return err
		}
		defer f.Close()
		_, err = f.WriteString(val)
		return err
	} else if err != nil {
		return err
	}

	b, err := ioutil.ReadFile(file)
	if err != nil {
		return err
	}

	if exists, err := regexp.Match(regexp.QuoteMeta(val), b); err != nil {
		return err
	} else if !exists {
		f, err := os.OpenFile(
			file,
			os.O_APPEND|os.O_WRONLY,
			0644,
		)
		if err != nil {
			return err
		}
		defer f.Close()

		if !bytes.HasSuffix(b, []byte("\n")) {
			val = "\n" + val
		}

		_, err = f.WriteString(val)
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

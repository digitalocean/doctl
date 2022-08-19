package commands

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/commands/displayers"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	// DefaultDevConfigFile is the name of the default dev configuration file.
	DefaultDevConfigFile = "dev-config.yaml"
)

type appDevUnknownKeyErr struct {
	key string
}

func (e *appDevUnknownKeyErr) Error() string {
	return fmt.Sprintf("unknown key: %s\nvalid keys: %s", e.key, outputValidAppDevKeys())
}

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
		fmt.Sprintf(`Set dev configuration settings for a build.

Valid Keys: %s
`, outputValidAppDevKeys()),
		Writer,
		displayerType(&displayers.Apps{}),
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
		fmt.Sprintf(`Unset dev configuration settings for a build.
		
Valid Keys: %s
`, outputValidAppDevKeys()),
		Writer,
		displayerType(&displayers.Apps{}),
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
		split := strings.Split(arg, "=")
		if len(split) != 2 {
			return errors.New("unexpected arg: " + arg)
		}
		err := dev.Set(split[0], split[1])
		if err != nil {
			return err
		}
	}

	err = dev.WriteConfig()
	if err != nil {
		return err
	}

	return nil
}

// RunAppsDevConfigUnset runs the set configuration command.
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
	}

	err = dev.WriteConfig()
	if err != nil {
		return err
	}

	return nil
}

type appDevConfig struct {
	cmdConfig *CmdConfig
	viper     *viper.Viper
}

var validAppDevKeys = map[string]bool{
	doctl.ArgApp:                true,
	doctl.ArgAppSpec:            true,
	doctl.ArgEnvFile:            true,
	doctl.ArgRegistryName:       true,
	doctl.ArgAppDevBuildCommand: true,
}

func outputValidAppDevKeys() string {
	keys := []string{}
	for k := range validAppDevKeys {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return strings.Join(keys, ", ")
}

func (c *appDevConfig) WriteConfig() error {
	return c.viper.WriteConfig()
}

func (c *appDevConfig) Set(key string, value string) error {
	if !validAppDevKeys[key] {
		return &appDevUnknownKeyErr{key}
	}
	c.viper.Set(key, value)
	return nil
}

func (c *appDevConfig) GetString(key string) (string, error) {
	if !validAppDevKeys[key] {
		return "", &appDevUnknownKeyErr{key}
	}
	if c.cmdConfig != nil {
		if v, err := c.cmdConfig.Doit.GetString(c.cmdConfig.NS, key); v != "" {
			return v, nil
		} else if err != nil {
			return "", err
		}
	}
	return c.viper.GetString(key), nil
}

func newAppDevConfig(cmdConfig *CmdConfig) (*appDevConfig, error) {
	config := &appDevConfig{
		cmdConfig: cmdConfig,
		viper:     viper.New(),
	}

	devConfigFilePath, err := cmdConfig.Doit.GetString(cmdConfig.NS, doctl.ArgAppDevConfig)
	if err != nil {
		return nil, err
	}

	if devConfigFilePath == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, err
		}
		configDir, err := findTopLevelGitDir(cwd)
		if err != nil {
			return nil, err
		}
		configDir = filepath.Join(configDir, ".do")
		err = os.MkdirAll(configDir, os.ModePerm)
		if err != nil {
			return nil, err
		}
		devConfigFilePath = filepath.Join(configDir, DefaultDevConfigFile)
		if err := ensureStringInFile(devConfigFilePath, ""); err != nil {
			return nil, err
		}
		if err := ensureStringInFile(filepath.Join(configDir, ".gitignore"), "dev-config.yaml"); err != nil {
			return nil, err
		}
	} else if _, err := os.Stat(devConfigFilePath); err != nil {
		return nil, err
	}

	config.viper.SetConfigType("yaml")
	config.viper.SetConfigFile(devConfigFilePath)

	if err := config.viper.ReadInConfig(); err != nil {
		return nil, err
	}

	return config, nil
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

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
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"syscall"

	"github.com/digitalocean/doctl"

	"golang.org/x/crypto/ssh/terminal"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	yaml "gopkg.in/yaml.v2"
)

// ErrUnknownTerminal signifies an unknown terminal. It is returned when doit
// can't ascertain the current terminal type with requesting an auth token.
var (
	ErrUnknownTerminal = errors.New("unknown terminal")
	cfgFileWriter      = defaultConfigFileWriter // create default cfgFileWriter
)

// retrieveUserTokenFromCommandLine is a function that can retrieve a token. By default,
// it will prompt the user. In test, you can replace this with code that returns the appropriate response.
func retrieveUserTokenFromCommandLine() (string, error) {
	if !terminal.IsTerminal(int(os.Stdout.Fd())) {
		return "", ErrUnknownTerminal
	}

	fmt.Print("DigitalOcean access token: ")
	passwdBytes, err := terminal.ReadPassword(int(syscall.Stdin))
	if err != nil {
		return "", err
	}

	return string(passwdBytes), nil
}

// UnknownSchemeError signifies an unknown HTTP scheme.
type UnknownSchemeError struct {
	Scheme string
}

var _ error = &UnknownSchemeError{}

func (use *UnknownSchemeError) Error() string {
	return "unknown scheme: " + use.Scheme
}

// Auth creates auth commands for doctl.
func Auth() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:   "auth",
			Short: "auth commands",
			Long:  "auth is used to access auth commands",
		},
	}

	cmdBuilderWithInit(cmd, RunAuthInit(retrieveUserTokenFromCommandLine), "init", "initialize configuration", "initialize configuration", Writer, false)
	cmdBuilderWithInit(cmd, RunAuthSwitch, "switch", "writes the auth context permanently to config", "writes the auth context permanently to config", Writer, false)
	cmdAuthList := cmdBuilderWithInit(cmd, RunAuthList, "list", "lists available auth contexts", "lists available auth contexts", Writer, false, aliasOpt("ls"))
	// The command runner expects that any command named "list" accepts a
	// format flag, so we include here despite only supporting text output for
	// this command.
	AddStringFlag(cmdAuthList, doctl.ArgFormat, "", "", "Columns for output in a comma separated list. Possible values: text")

	return cmd
}

// RunAuthInit initializes the doctl config. Configuration is stored in $XDG_CONFIG_HOME/doctl. On Unix, if
// XDG_CONFIG_HOME is not set, use $HOME/.config. On Windows use %APPDATA%/doctl/config.
func RunAuthInit(retrieveUserTokenFunc func() (string, error)) func(c *CmdConfig) error {
	return func(c *CmdConfig) error {
		token := c.getContextAccessToken()

		if token == "" {
			in, err := retrieveUserTokenFunc()
			if err != nil {
				return fmt.Errorf("unable to read DigitalOcean access token: %s", err)
			}
			token = strings.TrimSpace(in)
		} else {
			fmt.Fprintf(c.Out, "Using token [%v]", token)
			fmt.Fprintln(c.Out)
		}

		c.setContextAccessToken(string(token))

		fmt.Fprintln(c.Out)
		fmt.Fprint(c.Out, "Validating token... ")

		// need to initial the godo client since we've changed the configuration.
		if err := c.initServices(c); err != nil {
			return fmt.Errorf("unable to initialize DigitalOcean API client with new token: %s", err)
		}

		if _, err := c.Account().Get(); err != nil {
			fmt.Fprintln(c.Out, "invalid token")
			fmt.Fprintln(c.Out)
			return fmt.Errorf("unable to use supplied token to access API: %s", err)
		}

		fmt.Fprintln(c.Out, "OK")
		fmt.Fprintln(c.Out)

		return writeConfig()
	}
}

// RunAuthList lists all available auth contexts from the user's doctl config.
func RunAuthList(c *CmdConfig) error {
	context := Context
	if context == "" {
		context = viper.GetString("context")
	}
	contexts := viper.GetStringMap("auth-contexts")

	displayAuthContexts(c.Out, context, contexts)
	return nil
}

func displayAuthContexts(out io.Writer, currentContext string, contexts map[string]interface{}) {
	// Because the default context isn't present on the auth-contexts field,
	// we add it manually so that it's always included in the output, and so
	// we can check if it's the current context.
	contexts[doctl.ArgDefaultContext] = true

	// Extract and sort the map keys so that the order that we display the
	// auth contexts is consistent.
	keys := make([]string, 0)
	for ctx := range contexts {
		keys = append(keys, ctx)
	}
	sort.Strings(keys)

	for _, ctx := range keys {
		if ctx == currentContext {
			fmt.Fprintln(out, ctx, "(current)")
			continue
		}
		fmt.Fprintln(out, ctx)
	}
}

// RunAuthSwitch changes the default context and writes it to the
// configuration.
func RunAuthSwitch(c *CmdConfig) error {
	context := Context
	if context == "" {
		context = viper.GetString("context")
	}

	viper.Set("context", context)

	fmt.Printf("Now using context [%s] by default\n", context)
	return writeConfig()
}

func writeConfig() error {
	f, err := cfgFileWriter()
	if err != nil {
		return err
	}

	defer f.Close()

	b, err := yaml.Marshal(viper.AllSettings())
	if err != nil {
		return errors.New("unable to encode configuration to YAML format")
	}

	_, err = f.Write(b)
	if err != nil {
		return errors.New("unable to write configuration")
	}

	return nil
}

func defaultConfigFileWriter() (io.WriteCloser, error) {
	cfgFile := viper.GetString("config")
	f, err := os.Create(cfgFile)
	if err != nil {
		return nil, err
	}
	if err := os.Chmod(cfgFile, 0600); err != nil {
		return nil, err
	}

	return f, nil
}

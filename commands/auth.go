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
	"os"
	"strings"

	"golang.org/x/crypto/ssh/terminal"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// ErrUnknownTerminal signifies an unknown terminal. It is returned when doit
// can't ascertain the current terminal type with requesting an auth token.
var ErrUnknownTerminal = errors.New("unknown terminal")

// retrieveUserTokenFromCommandLine is a function that can retrieve a token. By default,
// it will prompt the user. In test, you can replace this with code that returns the appropriate response.
func retrieveUserTokenFromCommandLine() (string, error) {
	if !terminal.IsTerminal(int(os.Stdout.Fd())) {
		return "", ErrUnknownTerminal
	}

	fmt.Print("DigitalOcean access token: ")
	passwdBytes, err := terminal.ReadPassword(0)
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

	cmdBuilderWithInit(cmd, RunAuthInit(retrieveUserTokenFromCommandLine), "init", "initialize configuration", Writer, false, docCategories("auth"))
	cmdBuilderWithInit(cmd, RunAuthSwitch, "switch", "writes the auth context permanently to config", Writer, false, docCategories("auth"))

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

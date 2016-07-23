/*
Copyright 2016 The Doctl Authors All rights reserved.
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
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"

	"golang.org/x/crypto/ssh/terminal"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// ErrUnknownTerminal signies an unknown terminal. It is returned when doit
// can't ascertain the current terminal type with requesting an auth token.
var ErrUnknownTerminal = errors.New("unknown terminal")

// retrieveUserTokenFunc is a function that can retrieve a token. By default,
// it will prompt the user. In test, you can replace this with code that returns the appropriate response.
var retrieveUserTokenFunc = func() (string, error) {
	if !terminal.IsTerminal(int(os.Stdout.Fd())) {
		return "", ErrUnknownTerminal
	}

	reader := bufio.NewReader(os.Stdin)
	fmt.Print("DigitalOcean access token: ")
	return reader.ReadString('\n')
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

	cmdBuilderWithInit(cmd, RunAuthInit, "init", "initialize configuration", Writer, false, docCategories("auth"))

	return cmd
}

// RunAuthInit initializes the doctl config. Configuration is stored in $XDG_CONFIG_HOME/doctl. On Unix, if
// XDG_CONFIG_HOME is not set, use $HOME/.config. On Windows use %APPDATA%/doctl/config.
func RunAuthInit(c *CmdConfig) error {
	in, err := retrieveUserTokenFunc()
	if err != nil {
		return fmt.Errorf("unable to read DigitalOcean access token: %s", err)
	}

	token := strings.TrimSpace(in)

	viper.Set("access-token", string(token))

	fmt.Fprintln(c.Out)
	fmt.Fprint(c.Out, "Validating token: ")

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

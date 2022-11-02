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
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/digitalocean/doctl/do"
)

const (
	// noCapture is the string constant recognized by the plugin.  It suppresses output
	// capture when in the initial (command) position.
	noCapture = "nocapture"
)

// ServerlessExec executes a serverless command
func ServerlessExec(c *CmdConfig, command string, args ...string) (do.ServerlessOutput, error) {
	serverless := c.Serverless()
	err := serverless.CheckServerlessStatus()
	if err != nil {
		return do.ServerlessOutput{}, err
	}
	return serverlessExecNoCheck(serverless, command, args)
}

func serverlessExecNoCheck(serverless do.ServerlessService, command string, args []string) (do.ServerlessOutput, error) {
	cmd, err := serverless.Cmd(command, args)
	if err != nil {
		return do.ServerlessOutput{}, err
	}
	return serverless.Exec(cmd)
}

// RunServerlessExec is a variant of ServerlessExec convenient for calling from stylized command runners
// Sets up the arguments and (especially) the flags for the actual call
func RunServerlessExec(command string, c *CmdConfig, booleanFlags []string, stringFlags []string) (do.ServerlessOutput, error) {
	serverless := c.Serverless()
	err := serverless.CheckServerlessStatus()
	if err != nil {
		return do.ServerlessOutput{}, err
	}

	args := getFlatArgsArray(c, booleanFlags, stringFlags)
	cmd, err := serverless.Cmd(command, args)
	if err != nil {
		return do.ServerlessOutput{}, err
	}

	return serverless.Exec(cmd)
}

// RunServerlessExecStreaming is like RunServerlessExec but assumes that output will not be captured and can be streamed.
func RunServerlessExecStreaming(command string, c *CmdConfig, booleanFlags []string, stringFlags []string) error {
	serverless := c.Serverless()
	err := serverless.CheckServerlessStatus()
	if err != nil {
		return err
	}

	args := getFlatArgsArray(c, booleanFlags, stringFlags)
	args = append([]string{command}, args...)

	cmd, err := serverless.Cmd(noCapture, args)
	if err != nil {
		return err
	}
	// TODO the following does not filter output.  We might want output filtering as part of
	// this function.
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return serverless.Stream(cmd)
}

// PrintServerlessTextOutput prints the output of a serverless command execution in a
// textual form (often, this can be improved upon).
// Prints Formatted if present.
// Else, prints Captured if present.
// Else, prints Table or Entity using generic JSON formatting.
// We don't expect both Table and Entity to be present and have no
// special handling for that.
func (c *CmdConfig) PrintServerlessTextOutput(output do.ServerlessOutput) error {
	var err error
	if len(output.Formatted) > 0 {
		_, err = fmt.Fprintln(c.Out, strings.Join(output.Formatted, "\n"))
	} else if len(output.Captured) > 0 {
		_, err = fmt.Fprintln(c.Out, strings.Join(output.Captured, "\n"))
	} else if len(output.Table) > 0 {
		_, err = fmt.Fprintln(c.Out, genericJSON(output.Table))
	} else if output.Entity != nil {
		_, err = fmt.Fprintln(c.Out, genericJSON(output.Entity))
	} // else no output (unusual but not impossible)

	return err
}

func hashAccessToken(c *CmdConfig) string {
	return do.HashAccessToken(c.getContextAccessToken())
}

// Determines whether the serverless appears to be connected.  The purpose is
// to fail fast (when feasible) on serverless that are clearly not connected.
// However, it is important not to add excessive overhead on each call (e.g.
// asking the plugin to validate credentials), so the test is not foolproof.
// It merely tests whether a credentials directory has been created for the
// current doctl access token and appears to have a credentials.json in it.
func isServerlessConnected(leafCredsDir string, serverlessDir string) bool {
	creds := do.GetCredentialDirectory(leafCredsDir, serverlessDir)
	credsFile := filepath.Join(creds, do.CredentialsFile)
	_, err := os.Stat(credsFile)
	return !os.IsNotExist(err)
}

// Converts something "object-like" but untyped to generic JSON
// Designed for human eyes; does not provide an explicit error
// result
func genericJSON(toFormat interface{}) string {
	bytes, err := json.MarshalIndent(&toFormat, "", "  ")
	if err != nil {
		return "<not representable as JSON>"
	}
	return string(bytes)
}

// Convert the actual args, the boolean flags, and the string flags for a command
// into a flat array which are passed to the plugin as 'args'.
func getFlatArgsArray(c *CmdConfig, booleanFlags []string, stringFlags []string) []string {
	args := append([]string{}, c.Args...)
	for _, flag := range booleanFlags {
		truth, err := c.Doit.GetBool(c.NS, flag)
		if truth && err == nil {
			args = append(args, "--"+flag)
		}
	}
	for _, flag := range stringFlags {
		value, err := c.Doit.GetString(c.NS, flag)
		if err == nil && len(value) > 0 {
			args = append(args, "--"+flag, value)
		}
	}

	return args
}

// getServerlessDirectory returns the "serverless" directory in which the artifacts for serverless support
// are stored.  Returns the name of the directory whether or not it exists.  The standard location
// (and the only one that customers are expected to use) is relative to the defaultConfigHome.
func getServerlessDirectory() string {
	return filepath.Join(defaultConfigHome(), "sandbox")
}

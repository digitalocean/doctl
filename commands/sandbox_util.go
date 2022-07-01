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
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/digitalocean/doctl/do"
)

const (
	// Minimum required version of the sandbox plugin code.  The first part is
	// the version of the incorporated Nimbella CLI and the second part is the
	// version of the bridge code in the sandbox plugin repository.
	minSandboxVersion = "4.1.0-1.3.0"

	// The version of nodejs to download alongsize the plugin download.
	nodeVersion = "v16.13.0"

	// noCapture is the string constant recognized by the plugin.  It suppresses output
	// capture when in the initial (command) position.
	noCapture = "nocapture"

	// credsDir is the directory under the sandbox where all credentials are stored.
	// It in turn has a subdirectory for each access token employed (formed as a prefix of the token).
	credsDir = "creds"
)

// SandboxExec executes a sandbox command
func SandboxExec(c *CmdConfig, command string, args ...string) (do.SandboxOutput, error) {
	sandbox := c.Sandbox()
	err := sandbox.CheckSandboxStatus(hashAccessToken(c))
	if err != nil {
		return do.SandboxOutput{}, err
	}
	return sandboxExecNoCheck(sandbox, command, args)
}

func sandboxExecNoCheck(sandbox do.SandboxService, command string, args []string) (do.SandboxOutput, error) {
	cmd, err := sandbox.Cmd(command, args)
	if err != nil {
		return do.SandboxOutput{}, err
	}
	return sandbox.Exec(cmd)
}

// RunSandboxExec is a variant of SandboxExec convenient for calling from stylized command runners
// Sets up the arguments and (especially) the flags for the actual call
func RunSandboxExec(command string, c *CmdConfig, booleanFlags []string, stringFlags []string) (do.SandboxOutput, error) {
	sandbox := c.Sandbox()
	err := sandbox.CheckSandboxStatus(hashAccessToken(c))
	if err != nil {
		return do.SandboxOutput{}, err
	}

	args := getFlatArgsArray(c, booleanFlags, stringFlags)
	cmd, err := sandbox.Cmd(command, args)
	if err != nil {
		return do.SandboxOutput{}, err
	}

	return sandbox.Exec(cmd)
}

// RunSandboxExecStreaming is like RunSandboxExec but assumes that output will not be captured and can be streamed.
func RunSandboxExecStreaming(command string, c *CmdConfig, booleanFlags []string, stringFlags []string) error {
	sandbox := c.Sandbox()
	err := sandbox.CheckSandboxStatus(hashAccessToken(c))
	if err != nil {
		return err
	}

	args := getFlatArgsArray(c, booleanFlags, stringFlags)
	args = append([]string{command}, args...)

	cmd, err := sandbox.Cmd(noCapture, args)
	if err != nil {
		return err
	}
	// TODO the following does not filter output.  We might want output filtering as part of
	// this function.
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return sandbox.Stream(cmd)
}

// PrintSandboxTextOutput prints the output of a sandbox command execution in a
// textual form (often, this can be improved upon).
// Prints Formatted if present.
// Else, prints Captured if present.
// Else, prints Table or Entity using generic JSON formatting.
// We don't expect both Table and Entity to be present and have no
// special handling for that.
func (c *CmdConfig) PrintSandboxTextOutput(output do.SandboxOutput) error {
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
	token := c.getContextAccessToken()
	hasher := sha1.New()
	hasher.Write([]byte(token))
	sha := hasher.Sum(nil)
	return hex.EncodeToString(sha[:4])
}

// Determines whether the sandbox appears to be connected.  The purpose is
// to fail fast (when feasible) on sandboxes that are clearly not connected.
// However, it is important not to add excessive overhead on each call (e.g.
// asking the plugin to validate credentials), so the test is not foolproof.
// It merely tests whether a credentials directory has been created for the
// current doctl access token and appears to have a credentials.json in it.
func isSandboxConnected(leafCredsDir string, sandboxDir string) bool {
	creds := do.GetCredentialDirectory(leafCredsDir, sandboxDir)
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

// getSandboxDirectory returns the "sandbox" directory in which the artifacts for sandbox support
// are stored.  Returns the name of the directory whether or not it exists.  The standard location
// (and the only one that customers are expected to use) is relative to the defaultConfigHome.
func getSandboxDirectory() string {
	return filepath.Join(defaultConfigHome(), "sandbox")
}

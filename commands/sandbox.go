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
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/do"
	"github.com/spf13/cobra"
)

const NODE_VERSION = "v14.16.0"
const MIN_SANDBOX_VERSION = "2.3.1-1.0.0"

// SandboxNotInstalledErr is the error returned to users when the sandbox is not installed.
var SandboxNotInstalledErr = errors.New("The sandbox is not installed (use `doctl sandbox install`)")

// SandboxNeedsUpgradeErr is the error returned to users when the sandbox is at too low a version
var SandboxNeedsUpgradeErr = errors.New("The sandbox support needs to be upgraded (use `doctl sandbox upgrade`)")

// SandboxNotConnectedErr is the error returned to users when the sandbox is not connected to a namespace
var SandboxNotConnectedErr = errors.New("A sandbox is installed but not connected to a function namespace (use `doctl sandbox connect`)")

// Contains support for 'sandbox' commands provided by a hidden install of the Nimbella CLI
// The literal command 'doctl sandbox' is used only to install the sandbox and drive the
// 'nim auth' subtree as needed for the integration.  All other 'nim' subtrees are shimmed
// with independent 'doctl' commands as needed.
func Sandbox() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:   "sandbox",
			Short: "[Beta] Develop functions in a sandbox prior to deploying them in an app",
			Long: `The ` + "`" + `doctl sandbox` + "`" + ` commands provide a development sandbox for functions.  A sandbox has a local file system component and a cloud component.
A one-time install of the sandbox software is needed (use ` + "`" + `doctl sandbox install` + "`" + ` to install the software, then ` + "`" + `doctl sandbox connect` + "`" + ` to
connect to the cloud component of the sandbox provided with your account).  Other ` + "`" + `doctl sandbox` + "`" + ` commands are used to develop and test.`,
			Aliases: []string{"sbx"},
			Hidden:  !isSandboxInstalled(),
		},
	}

	// TODO: combine "install" and "connect into a single "enable" command, then "uninstall" should become "disable".
	CmdBuilder(cmd, RunSandboxInstall, "install", "Installs the sandbox support",
		`This command installs additional software under `+"`"+`doctl`+"`"+` needed to make the other sandbox commands work.
The install operation is long-running, and a network connection is required.`,
		Writer)

	CmdBuilder(cmd, RunSandboxUpgrade, "upgrade", "Upgrades sandbox support to match this version of doctl",
		`This command upgrades the sandbox support software under `+"`"+`doctl`+"`"+` by installing over the existing version.
The install operation is long-running, and a network connection is required.`,
		Writer)

	CmdBuilder(cmd, RunSandboxUninstall, "uninstall", "Removes the sandbox support", `Removes sandbox support from `+"`"+`doctl`+"`",
		Writer)

	CmdBuilder(cmd, RunSandboxConnect, "connect [<token>|<namespace>]", "Connect the cloud portion of your sandbox",
		`This command connects the cloud portion of your sandbox (needed for testing).
There are several ways to use it.
1. If your account has been explicitly provisioned with sandbox support, invoke with no arguments.
2. (DO internal) If you obtained a namespace for your account using the alpha console, you may
use either the namespace name or the generated token as the argument.  The ability to generate
and use a token is for compatibility with the previous behavior of the command and is likely
to be deprecated and removed`,
		Writer)

	CmdBuilder(cmd, RunSandboxStatus, "status", "Provide information about your sandbox",
		`This command reports the status of your sandbox and some details
concerning its connected cloud portion`, Writer)

	cmd.AddCommand(Activations())
	cmd.AddCommand(Functions())
	SandboxExtras(cmd)
	return cmd
}

// RunSandboxInstall performs the network installation of the 'nim' adjunct to support sandbox development
func RunSandboxInstall(c *CmdConfig) error {
	status := c.checkSandboxStatus()
	if status == SandboxNeedsUpgradeErr {
		fmt.Fprintln(c.Out, "Sandbox support is already installed, but needs an upgrade for this version of `doctl`.")
		fmt.Fprintln(c.Out, "Use `doctl sandbox upgrade` to upgrade the support.")
		return nil
	}
	if status == nil {
		fmt.Fprintln(c.Out, "Sandbox support is already installed at an appropriate version.  No action needed.")
		return nil
	}
	sandboxDir, _ := getSandboxDirectory()
	return c.installSandbox(sandboxDir, false)
}

// 'doctl sandbox upgrade' is a variant on 'doctl sandbox install' for installing over an existing version when
// the existing version is inadequate as detected by isSandboxUpToDate()
func RunSandboxUpgrade(c *CmdConfig) error {
	status := c.checkSandboxStatus()
	if status == nil {
		fmt.Fprintln(c.Out, "Sandbox support is already installed at an appropriate version.  No action needed.")
		// TODO should there be an option to upgrade beyond the minimum needed?
		return nil
	}
	if status == SandboxNotInstalledErr {
		fmt.Fprintln(c.Out, "Sandbox support was never installed.  Use `doctl sandbox install`.")
		return nil
	}
	sandboxDir, _ := getSandboxDirectory()
	return c.installSandbox(sandboxDir, true)
}

// The uninstall command
func RunSandboxUninstall(c *CmdConfig) error {
	sandboxDir, exists := getSandboxDirectory()
	if !exists {
		return errors.New("Nothing to uninstall: no sandbox was found")
	}
	return os.RemoveAll(sandboxDir)
}

// The connect command
func RunSandboxConnect(c *CmdConfig) error {
	var arg string
	var token bool
	var creds do.SandboxCredentials
	var err error
	if len(c.Args) > 1 {
		return doctl.NewTooManyArgsErr(c.NS)
	}
	if len(c.Args) == 1 {
		arg = c.Args[0]
		token = isJWT(arg)
	}
	if token {
		creds, err = c.Sandbox().ResolveToken(context.TODO(), arg)
	} else {
		creds, err = c.Sandbox().ResolveNamespace(context.TODO(), arg)
	}
	if err != nil {
		return err
	}
	result, err := SandboxExec(c, "auth/login", "--auth", creds.Auth, "--apihost", creds.ApiHost)
	if err != nil {
		return err
	}
	mapResult := result.Entity.(map[string]interface{})
	fmt.Fprintf(c.Out, "Connected to function namespace '%s' on API host '%s'\n", mapResult["namespace"], mapResult["apihost"])
	fmt.Fprintln(c.Out)
	return nil
}

// Determine whether an argument is a JWT or a namespace.  The heuristic is based on the length of the
// string and whether it contains two dots (then JWT, otherwise namespace).
func isJWT(candidate string) bool {
	if len(candidate) < 30 {
		return false
	}
	parts := strings.Split(candidate, ".")
	return len(parts) == 3
}

// The status command
func RunSandboxStatus(c *CmdConfig) error {
	status := c.checkSandboxStatus()
	if status == SandboxNeedsUpgradeErr || status == SandboxNotInstalledErr {
		return status
	}
	if status != nil {
		return fmt.Errorf("Unexpected error: %w", status)
	}
	result, err := SandboxExec(c, "auth/current", "--apihost", "--name")
	if err != nil || len(result.Error) > 0 {
		return SandboxNotConnectedErr
	}
	if result.Entity == nil {
		return errors.New("Could not retrieve information about the connected namespace")
	}
	mapResult := result.Entity.(map[string]interface{})
	fmt.Fprintf(c.Out, "Connected to function namespace '%s' on API host '%s'\n\n", mapResult["name"], mapResult["apihost"])
	return nil
}

// "Public" functions

// Executes a sandbox command
func SandboxExec(c *CmdConfig, command string, args ...string) (do.SandboxOutput, error) {
	err := c.checkSandboxStatus()
	if err != nil {
		return do.SandboxOutput{}, err
	}
	sandbox := c.Sandbox()
	cmd, err := sandbox.Cmd(command, args)
	if err != nil {
		return do.SandboxOutput{}, err
	}
	// If DEBUG is specified, we need to open up stderr for that stream.  The stdout stream
	// will continue to work for returning structured results.
	if os.Getenv("DEBUG") != "" {
		cmd.Stderr = os.Stderr
	}

	return sandbox.Exec(cmd)
}

// A variant of SandboxExec convenient for calling from stylized command runners
// Sets up the arguments and (especially) the flags for the actual call
func RunSandboxExec(command string, c *CmdConfig, booleanFlags []string, stringFlags []string) (do.SandboxOutput, error) {
	err := c.checkSandboxStatus()
	if err != nil {
		return do.SandboxOutput{}, err
	}

	sandbox := c.Sandbox()
	args := getFlatArgsArray(c, booleanFlags, stringFlags)
	cmd, err := sandbox.Cmd(command, args)
	if err != nil {
		return do.SandboxOutput{}, err
	}

	return sandbox.Exec(cmd)
}

// Like RunSandboxExec but assumes that output will not be captured and can be streamed.
func RunSandboxExecStreaming(command string, c *CmdConfig, booleanFlags []string, stringFlags []string) error {
	err := c.checkSandboxStatus()
	if err != nil {
		return err
	}
	sandbox := c.Sandbox()

	args := getFlatArgsArray(c, booleanFlags, stringFlags)
	cmd, err := sandbox.Cmd(command, args)
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
	if len(output.Formatted) > 0 {
		fmt.Fprintf(c.Out, strings.Join(output.Formatted, "\n"))
	} else if len(output.Captured) > 0 {
		fmt.Fprintf(c.Out, strings.Join(output.Captured, "\n"))
	} else if len(output.Table) > 0 {
		fmt.Fprintf(c.Out, genericJSON(output.Table))
	} else if output.Entity != nil {
		fmt.Fprintf(c.Out, genericJSON(output.Entity))
	} // else no output (unusual but not impossible)

	_, err := fmt.Fprintln(c.Out)

	return err
}

// Check install status and return an appropriate error for common issues
// such as not installed or needs upgrade.  Returns nil when no error.
func CheckSandboxStatus() error {
	sandboxDir, exists := getSandboxDirectory()
	if !exists {
		return SandboxNotInstalledErr
	}
	if !sandboxUptodate(sandboxDir) {
		return SandboxNeedsUpgradeErr
	}
	return nil
}

// Working subroutine for 'install' and 'upgrade'
func InstallSandbox(sandboxDir string, upgrading bool) error {
	// Make a temporary directory for use during the install
	tmp, err := ioutil.TempDir("", "doctl-sandbox")
	if err != nil {
		return err
	}
	// Download the nodejs tarball for this os and architecture
	goos := runtime.GOOS
	arch := runtime.GOARCH
	if arch == "amd64" {
		arch = "x64"
	} else if arch == "windows" {
		arch = "win"
	}
	fmt.Print("Downloading...")
	var nodeFileName string
	var nodeDir string
	// Download nodejs only if necessary
	if !upgrading || !canReuseNode(sandboxDir) {
		nodeDir = fmt.Sprintf("node-%s-%s-%s", NODE_VERSION, goos, arch)
		URL := fmt.Sprintf("https://nodejs.org/dist/%s/%s.tar.gz", NODE_VERSION, nodeDir)
		nodeFileName = filepath.Join(tmp, "node-install.tar.gz")
		err = download(URL, nodeFileName)
		if err != nil {
			return err
		}
	}
	// Download the fat tarball with the nim CLI, deployer, and sandbox bridge
	// TODO do these need to be arch-specific?  Currently assuming not.
	URL := fmt.Sprintf("https://do-serverless-tools.nyc3.digitaloceanspaces.com/doctl-sandbox-%s.tar.gz", MIN_SANDBOX_VERSION)
	sandboxFileName := filepath.Join(tmp, "doctl-sandbox.tar.gz")
	err = download(URL, sandboxFileName)
	if err != nil {
		return err
	}
	// Exec the tar binary at least once to unpack the fat tarball and possibly a second time if
	// node was downloaded.  If node was not download, just move the existing binary into place.
	// TODO eliminate use of separate tar binary so we can support windows install with pure go code
	fmt.Print("Unpacking...")
	cmd := exec.Command("tar", "-C", tmp, "-xzf", sandboxFileName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("%s", output)
		return err
	}
	if nodeFileName != "" {
		cmd := exec.Command("tar", "-C", tmp, "-xzf", nodeFileName)
		output, err := cmd.CombinedOutput()
		if err != nil {
			fmt.Printf("%s", output)
			return err
		}
	}
	// Move artifacts to final location
	fmt.Print("Installing...")
	srcPath := filepath.Join(tmp, "sandbox")
	if upgrading {
		// Preserve credentials by moving them from target (which will be replaced) to source.
		credPath := filepath.Join(sandboxDir, ".nimbella")
		relocPath := filepath.Join(srcPath, ".nimbella")
		err = os.Rename(credPath, relocPath)
		if err != nil {
			return err
		}
		// Preserve existing node if necessary
		if nodeFileName == "" {
			// Node was not downloaded
			err = moveExistingNode(sandboxDir, srcPath)
			if err != nil {
				return err
			}
		}
	}
	// Remove former sandboxDir before moving in the new one
	err = os.RemoveAll(sandboxDir)
	if err != nil {
		return err
	}
	err = os.Rename(srcPath, sandboxDir)
	if err != nil {
		return err
	}
	if nodeFileName != "" {
		srcPath = filepath.Join(tmp, nodeDir, "bin", "node")
		destPath := filepath.Join(sandboxDir, "node")
		err = os.Rename(srcPath, destPath)
		if err != nil {
			return err
		}
	}
	fmt.Println("\nDone")
	return nil
}

// "Private" utility functions

// getCurrentSandboxVersion gets the version of the current sandbox.
// To be called only when sandbox is known to exist.
// Returns "0" if the installed sandbox pre-dates the versioning system
// Otherwise, returns the version string stored in the sandbox directory.
func getCurrentSandboxVersion(sandboxDir string) string {
	versionFile := filepath.Join(sandboxDir, "version")
	contents, err := ioutil.ReadFile(versionFile)
	if err != nil {
		return "0"
	}
	return string(contents)
}

// Answers whether sandbox is installed
func isSandboxInstalled() bool {
	_, yes := getSandboxDirectory()
	return yes
}

// Gets the version of the node binary in the sandbox.  Determine if it is
// usable or whether it has to be upgraded.
func canReuseNode(sandboxDir string) bool {
	nodeBin := filepath.Join(sandboxDir, "node")
	cmd := exec.Command(nodeBin, "--version")
	result, err := cmd.Output()
	if err == nil {
		installed := strings.TrimSpace(string(result))
		return installed == NODE_VERSION
	}
	return false
}

// Moves the existing node binary from the sandbox that contains it to the new sandbox being
// staged during an upgrade.  This preserves it for reuse and avoids the need to download.
func moveExistingNode(existing string, staging string) error {
	srcPath := filepath.Join(existing, "node")
	destPath := filepath.Join(staging, "node")
	return os.Rename(srcPath, destPath)
}

// Download a network file to a local file
func download(URL, targetFile string) error {
	response, err := http.Get(URL)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != 200 {
		return errors.New(fmt.Sprintf("Received status code %d attempting to download from %s",
			response.StatusCode, URL))
	}
	file, err := os.Create(targetFile)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = io.Copy(file, response.Body)
	if err != nil {
		return err
	}
	return nil
}

// getSandboxDirectory returns the "sandbox" directory in which the artifacts for sandbox support
// are stored.  Returns the name of the directory and whether it exists.
func getSandboxDirectory() (string, bool) {
	sandboxDir := filepath.Join(defaultConfigHome(), "sandbox")
	_, err := os.Stat(sandboxDir)
	exists := !os.IsNotExist(err)
	return sandboxDir, exists
}

// sandboxUpToDate answers whether the installed version of the sandbox is at least
// what is required by doctl
func sandboxUptodate(sandboxDir string) bool {
	return getCurrentSandboxVersion(sandboxDir) >= MIN_SANDBOX_VERSION
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
// This also adjusts certain flag names and values between doctl usage and nim usage.
// 1.  The flag 'function' is renamed to 'action' if specified.
// 2.  The flag 'exclude' is checked to ensure that, if empty, it is set to "web" and
//     if non-empty, the "web" value as added to it.
// 3.  If the flag 'package' appears, the flag 'deployed' is added.
// TODO these adjustments belong further up the call stack to avoid unintended collisions.
// Some modest refactoring should enable that.
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
			if flag == "function" {
				flag = "action"
				// TODO: Should this be added to the args?
			} else if flag == "exclude" {
				// --exclude non-empty, add web
				args = append(args, "--exclude", value+",web")
			} else if flag == "package" {
				args = append(args, "--deployed", "--package", value)
			} else if flag == "param" {
				values := strings.Split(value, " ")
				args = append(args, "--param")
				args = append(args, values...)
			} else {
				args = append(args, "--"+flag, value)
			}
		} else if err == nil && flag == "exclude" {
			// --exclude not specified, set it to "web"
			args = append(args, "--exclude", "web")
		}
	}

	return args
}

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

	"github.com/spf13/cobra"
)

const NODE_VERSION = "14.16.0"

// This is what is returned from calls to the sandbox
type SandboxOutput = struct {
	Table     []map[string]interface{} `json:"table,omitempty"`
	Captured  []string                 `json:"captured,omitempty"`
	Formatted []string                 `json:"formatted,omitempty"`
	Entity    interface{}              `json:"entity,omitempty"`
	Error     string                   `json:"error,omitempty"`
}

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
		},
	}

	// TODO: combine "install" and "connect into a single "enable" command, then "uninstall" should become "disable".
	// We also need an update strategy.
	cmdBuilderWithInit(cmd, RunSandboxInstall, "install", "Installs the sandbox support",
		`This command installs additional software under `+"`"+`doctl`+"`"+` needed to make the other sandbox commands work.
The install operation is long-running, and a network connection is required.`,
		Writer, false)

	cmdBuilderWithInit(cmd, RunSandboxUninstall, "uninstall", "Removes the sandbox support", `Removes sandbox support from `+"`"+`doctl`+"`",
		Writer, false)

	cmdBuilderWithInit(cmd, RunSandboxConnect, "connect <token>", "Connect the cloud portion of your sandbox",
		`This command connects the cloud portion of your sandbox (needed for testing) by using a provided token.
You obtain the token from the cloud console (details TBD)`,
		Writer, false)

	cmdBuilderWithInit(cmd, RunSandboxStatus, "status", "Provide information about your sandbox",
		`This command reports the status of your sandbox and some details
concerning its connected cloud portion`, Writer, false)

	cmd.AddCommand(Activations())
	cmd.AddCommand(Functions())
	SandboxExtras(cmd)
	return cmd
}

// RunSandboxInstall performs the network installation of the 'nim' adjunct to support sandbox development
func RunSandboxInstall(c *CmdConfig) error {
	// Check that the sandbox isn't already installed
	sandboxDir, sandboxExists := getSandboxDirectory()
	if sandboxExists {
		return errors.New("An existing sandbox install was detected.  Uninstall before installing again.")
	}
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
	}
	nodeDir := fmt.Sprintf("node-v%s-%s-%s", NODE_VERSION, goos, arch)
	URL := fmt.Sprintf("https://nodejs.org/dist/v%s/%s.tar.xz", NODE_VERSION, nodeDir)
	nodeFileName := filepath.Join(tmp, "node-install.tar.xz")
	fmt.Print("Downloading...")
	err = download(URL, nodeFileName)
	if err != nil {
		return err
	}
	// Download the fat tarball with the nim CLI, deployer, and sandbox bridge
	// TODO do these need to be arch-specific?  Currently assuming not.
	URL = "https://do-serverless-tools.nyc3.digitaloceanspaces.com/doctl-sandbox.tar.xz"
	sandboxFileName := filepath.Join(tmp, "doctl-sandbox.tar.xz")
	err = download(URL, sandboxFileName)
	if err != nil {
		return err
	}
	// Exec tar binary twice to unpack the two tarballs into the tmp directory
	fmt.Print("Unpacking...")
	cmd := exec.Command("tar", "-C", tmp, "-xJf", nodeFileName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("%s", output)
		return err
	}
	cmd = exec.Command("tar", "-C", tmp, "-xJf", sandboxFileName)
	output, err = cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("%s", output)
		return err
	}
	// Move artifacts to final location
	fmt.Print("Installing...")
	srcPath := filepath.Join(tmp, "sandbox")
	err = os.Rename(srcPath, sandboxDir)
	if err != nil {
		return err
	}
	srcPath = filepath.Join(tmp, nodeDir, "bin", "node")
	destPath := filepath.Join(sandboxDir, "node")
	err = os.Rename(srcPath, destPath)
	if err != nil {
		return err
	}
	fmt.Println("\nDone")
	return nil
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
	err := ensureOneArg(c)
	if err != nil {
		return err
	}
	token := c.Args[0]
	result, err := SandboxExec("auth/login", token)
	if err != nil {
		return err
	}
	mapResult := result.Entity.(map[string]interface{})
	fmt.Printf("Connected to function namespace '%s' on API host '%s'\n", mapResult["namespace"], mapResult["apihost"])
	return nil
}

// The status command
func RunSandboxStatus(c *CmdConfig) error {
	result, err := SandboxExec("auth/current", "--apihost", "--name")
	if err != nil || len(result.Error) > 0 {
		if IsSandboxInstalled() {
			return errors.New("A sandbox is installed but not connected to a function namespace (see 'doctl sandbox connect')")
			return nil
		}
		return errors.New("sandbox is not installed (use 'doctl sandbox install')")
	}
	if result.Entity == nil {
		return errors.New("Could not retrieve information about the connected namespace")
	}
	mapResult := result.Entity.(map[string]interface{})
	fmt.Printf("Connected to function namespace '%s' on API host '%s'\n", mapResult["name"], mapResult["apihost"])
	return nil
}

// "Public" functions

// Executes a sandbox command
func SandboxExec(command string, args ...string) (SandboxOutput, error) {
	cmd, err := setupSandboxSubprocess(command, args)
	if err != nil {
		return SandboxOutput{}, err
	}
	// If DEBUG is specified, we need to open up stderr for that stream.  The stdout stream
	// will continue to work for returning structured results.
	if os.Getenv("DEBUG") != "" {
		cmd.Stderr = os.Stderr
	}
	output, err := cmd.Output()
	if err != nil {
		// Ignore "errors" that are just non-zero exit.  The
		// sandbox uses this as a secondary indicator but the output
		// is still trustworthy (and includes error information inline)
		if _, ok := err.(*exec.ExitError); !ok {
			// Real error of some sort
			return SandboxOutput{}, err
		}
	}
	var result SandboxOutput
	err = json.Unmarshal(output, &result)
	if err != nil {
		return SandboxOutput{}, err
	}
	// Result is sound JSON but if it has an Error field the rest is not trustworthy
	if len(result.Error) > 0 {
		return SandboxOutput{}, errors.New(result.Error)
	}
	// Result is both sound and error free
	return result, nil
}

// A variant of SandboxExec convenient for calling from stylized command runners
// Sets up the arguments and (especially) the flags for the actual call
func RunSandboxExec(command string, c *CmdConfig, booleanFlags []string, stringFlags []string) (SandboxOutput, error) {
	args := getFlatArgsArray(c, booleanFlags, stringFlags)
	return SandboxExec(command, args...)
}

// Like RunSandboxExec but assumes that output will not be captured and can be streamed.
func RunSandboxExecStreaming(command string, c *CmdConfig, booleanFlags []string, stringFlags []string) error {
	args := getFlatArgsArray(c, booleanFlags, stringFlags)
	cmd, err := setupSandboxSubprocess(command, args)
	if err != nil {
		return err
	}
	// TODO the following does not filter output.  We might want output filtering as part of
	// this function.
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// Prints the output of a sandbox command execution in a
// textual form (often, this can be improved upon).
// Prints Formatted if present.
// Else, prints Captured if present.
// Else, prints Table or Entity using generic JSON formatting.
// We don't expect both Table and Entity to be present and have no
// special handling for that.
func PrintSandboxTextOutput(output SandboxOutput) {
	if len(output.Formatted) > 0 {
		fmt.Println(strings.Join(output.Formatted, "\n"))
	} else if len(output.Captured) > 0 {
		fmt.Println(strings.Join(output.Captured, "\n"))
	} else if len(output.Table) > 0 {
		fmt.Println(genericJSON(output.Table))
	} else if output.Entity != nil {
		fmt.Println(genericJSON(output.Entity))
	} // else no output (unusual but not impossible)
}

// Answers whether sandbox is installed
func IsSandboxInstalled() bool {
	_, yes := getSandboxDirectory()
	return yes
}

// "Private" utility functions

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

// Returns the "sandbox" directory in which the artifacts for sandbox support are stored.
// Returns the name of the directory and whether or not it exists.
func getSandboxDirectory() (string, bool) {
	sandboxDir := filepath.Join(defaultConfigHome(), "sandbox")
	_, err := os.Stat(sandboxDir)
	return sandboxDir, !os.IsNotExist(err)
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
			} else if flag == "exclude" {
				// --exclude non-empty, add web
				value = value + ",web"
			} else if flag == "package" {
				args = append(args, "--deployed")
			}
			args = append(args, "--"+flag, value)
		} else if err == nil && flag == "exclude" {
			// --exclude not specified, set it to "web"
			args = append(args, "--exclude", "web")
		}
	}
	return args
}

// Check for sandbox install, then setup a subprocess to run it for a given command and set of arguments
func setupSandboxSubprocess(command string, args []string) (*exec.Cmd, error) {
	sandboxDir, exists := getSandboxDirectory()
	if !exists {
		return nil, errors.New("The sandbox is not installed.  Use `doctl sandbox install` to install it")
	}
	node := filepath.Join(sandboxDir, "node")
	sandboxJs := filepath.Join(sandboxDir, "sandbox.js")
	nimbellaDir := filepath.Join(sandboxDir, ".nimbella")
	args = append([]string{sandboxJs, command}, args...)
	cmd := exec.Command(node, args...)
	cmd.Env = append(os.Environ(), "NIMBELLA_DIR="+nimbellaDir)
	return cmd, nil
}

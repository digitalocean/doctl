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

	"github.com/spf13/cobra"
)

const NODE_VERSION = "14.16.0"

// This is what is returned from calls to the sandbox
type SandboxOutput = struct {
	Table    []interface{} `json:"table"`
	Captured []string      `json:"captured"`
	Entity   interface{}   `json:"entity"`
}

// Contains support for 'sandbox' commands provided by a hidden install of the Nimbella CLI
// The literal command 'doctl sandbox' is used only to install the sandbox and drive the
// 'nim auth' subtree as needed for the integration.  All other 'nim' subtrees are shimmed
// with independent 'doctl' commands as needed.
func Sandbox() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:   "sandbox",
			Short: "Display commands for managing a serverless development sandbox",
			Long: `The ` + "`" + `doctl sandbox` + "`" + ` commands allow you to manage a serverless development sandbox as an optional add-on to ` + "`" + `doctl.
You can install and de-install the sandbox support, or use a token to connect to your sandbox namespace.`,
		},
	}

	cmdBuilderWithInit(cmd, RunSandboxInstall, "install", "Installs the sandbox support", `This command installs an add-on to `+"`"+`doctl that supports
sandbox development of serverless apps.  The command is long-running, and a network connection is required.`, Writer, false)

	cmdBuilderWithInit(cmd, RunSandboxUninstall, "uninstall", "Removes the sandbox support", ``, Writer, false)

	cmdBuilderWithInit(cmd, RunSandboxConnect, "connect <token>", "Connect your sandbox", `This command connects to your sandbox namespace using a token.
You obtain the token from the cloud console (details TBD)`, Writer, false)

	return cmd
}

// Executes a sandbox command
func SandboxExec(command ...string) (SandboxOutput, error) {
	sandboxDir, exists := getSandboxDirectory()
	if !exists {
		return SandboxOutput{}, errors.New("The sandbox is not installed.  Use `doctl sandbox install` to install it")
	}
	node := filepath.Join(sandboxDir, "node")
	sandboxJs := filepath.Join(sandboxDir, "sandbox.js")
	nimbellaDir := filepath.Join(sandboxDir, ".nimbella")
	args := append([]string{sandboxJs}, command...)
	cmd := exec.Command(node, args...)
	cmd.Env = append(os.Environ(), "NIMBELLA_DIR="+nimbellaDir)
	output, err := cmd.Output()
	if err != nil {
		return SandboxOutput{}, err
	}
	var result SandboxOutput
	err = json.Unmarshal(output, &result)
	if err != nil {
		return SandboxOutput{}, err
	}
	return result, nil
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

// Returns the "sandbox" directory in which the artifacts for sandbox support are stored.
// Returns the name of the directory and whether or not it exists.
func getSandboxDirectory() (string, bool) {
	sandboxDir := filepath.Join(defaultConfigHome(), "sandbox")
	_, err := os.Stat(sandboxDir)
	return sandboxDir, !os.IsNotExist(err)
}

// Invoke the sandbox bridge

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
	fmt.Printf("Output was: %v\n", result)
	return nil
}

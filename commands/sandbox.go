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
	"crypto/sha1"
	"encoding/hex"
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
	"github.com/digitalocean/doctl/pkg/extract"
	"github.com/spf13/cobra"
)

const (
	// Minimum required version of the sandbox plugin code.  The first part is
	// the version of the incorporated Nimbella CLI and the second part is the
	// version of the bridge code in the sandbox plugin repository.
	minSandboxVersion = "3.0.2-1.2.0"

	// The version of nodejs to download alongsize the plugin downl
	nodeVersion = "v16.13.0"

	// noCapture is the string constant recognized by the plugin.  It suppresses output
	// capture when in the initial (command) position.
	noCapture = "nocapture"

	// credsDir is the directory under the sandbox where all credentials are stored.
	// It in turn has a subdirectory for each access token employed (formed as a prefix of the token).
	credsDir = "creds"
)

var (
	// ErrSandboxNotInstalled is the error returned to users when the sandbox is not installed.
	ErrSandboxNotInstalled = errors.New("The sandbox is not installed (use `doctl sandbox install`)")
	// ErrSandboxNeedsUpgrade is the error returned to users when the sandbox is at too low a version
	ErrSandboxNeedsUpgrade = errors.New("The sandbox support needs to be upgraded (use `doctl sandbox upgrade`)")
	// ErrSandboxNotConnected is the error returned to users when the sandbox is not connected to a namespace
	ErrSandboxNotConnected = errors.New("A sandbox is installed but not connected to a function namespace (use `doctl sandbox connect`)")
	// ErrUndeployAllAndArgs is the error returned when the --all flag is used along with args on undeploy
	errUndeployAllAndArgs = errors.New("command line arguments and the `--all` flag are mutually exclusive")
	// ErrUndeployTooFewArgs is the error returned when neither --all nor args are specified on undeploy
	errUndeployTooFewArgs = errors.New("either command line arguments or `--all` must be specified")
)

// Sandbox contains support for 'sandbox' commands provided by a hidden install of the Nimbella CLI
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

	CmdBuilder(cmd, RunSandboxConnect, "connect", "Connect the cloud portion of your sandbox",
		`This command connects the cloud portion of your sandbox (needed for testing).`,
		Writer)

	status := CmdBuilder(cmd, RunSandboxStatus, "status", "Provide information about your sandbox",
		`This command reports the status of your sandbox and some details concerning its connected cloud portion.
With the `+"`"+`--languages flag, it will report the supported languages.`, Writer)
	AddBoolFlag(status, "languages", "l", false, "show available languages (if connected to the cloud)")

	undeploy := CmdBuilder(cmd, RunSandboxUndeploy, "undeploy [<package|function>...]",
		"Removes resources from the cloud portion of your sandbox",
		`This command removes functions, entire packages, or all functions and packages, from the cloud portion
of your sandbox.  In general, deploying new content does not remove old content although it may overwrite it.
Use `+"`"+`doctl sandbox undeploy`+"`"+` to effect removal.  The command accepts a list of functions or packages.
Functions should be listed in `+"`"+`pkgName/fnName`+"`"+` form, or `+"`"+`fnName`+"`"+` for a function not in any package.
The `+"`"+`--packages`+"`"+` flag causes arguments without slash separators to be intepreted as packages, in which case
the entire packages are removed.`, Writer)
	AddBoolFlag(undeploy, "packages", "p", false, "interpret simple name arguments as packages")
	AddBoolFlag(undeploy, "all", "", false, "remove all packages and functions")

	cmd.AddCommand(Activations())
	cmd.AddCommand(Functions())
	SandboxExtras(cmd)
	return cmd
}

// RunSandboxInstall performs the network installation of the 'nim' adjunct to support sandbox development
func RunSandboxInstall(c *CmdConfig) error {
	status := c.checkSandboxStatus()
	if status == ErrSandboxNeedsUpgrade {
		fmt.Fprintln(c.Out, "Sandbox support is already installed, but needs an upgrade for this version of `doctl`.")
		fmt.Fprintln(c.Out, "Use `doctl sandbox upgrade` to upgrade the support.")
		return nil
	}
	if status == nil {
		fmt.Fprintln(c.Out, "Sandbox support is already installed at an appropriate version.  No action needed.")
		return nil
	}
	sandboxDir, _ := getSandboxDirectory()
	return c.installSandbox(c, sandboxDir, false)
}

// RunSandboxUpgrade is a variant on RunSandboxInstall for installing over an existing version when
// the existing version is inadequate as detected by checkSandboxStatus()
func RunSandboxUpgrade(c *CmdConfig) error {
	status := c.checkSandboxStatus()
	if status == nil {
		fmt.Fprintln(c.Out, "Sandbox support is already installed at an appropriate version.  No action needed.")
		// TODO should there be an option to upgrade beyond the minimum needed?
		return nil
	}
	if status == ErrSandboxNotInstalled {
		fmt.Fprintln(c.Out, "Sandbox support was never installed.  Use `doctl sandbox install`.")
		return nil
	}
	sandboxDir, _ := getSandboxDirectory()
	return c.installSandbox(c, sandboxDir, true)
}

// RunSandboxUninstall removes the sandbox support and any stored credentials
func RunSandboxUninstall(c *CmdConfig) error {
	sandboxDir, exists := getSandboxDirectory()
	if !exists {
		return errors.New("Nothing to uninstall: no sandbox was found")
	}
	return os.RemoveAll(sandboxDir)
}

// RunSandboxConnect implements the sandbox connect command
func RunSandboxConnect(c *CmdConfig) error {
	var (
		creds do.SandboxCredentials
		err   error
	)
	if len(c.Args) > 0 {
		return doctl.NewTooManyArgsErr(c.NS)
	}
	creds, err = c.Sandbox().GetSandboxNamespace(context.TODO())
	if err != nil {
		return err
	}
	result, err := SandboxExec(c, "auth/login", "--auth", creds.Auth, "--apihost", creds.APIHost)
	if err != nil {
		return err
	}
	mapResult := result.Entity.(map[string]interface{})
	fmt.Fprintf(c.Out, "Connected to function namespace '%s' on API host '%s'\n", mapResult["namespace"], mapResult["apihost"])
	fmt.Fprintln(c.Out)
	return nil
}

// RunSandboxStatus gives a report on the status of the sandbox (installed, up to date, connected)
func RunSandboxStatus(c *CmdConfig) error {
	status := c.checkSandboxStatus()
	if status == ErrSandboxNeedsUpgrade || status == ErrSandboxNotInstalled {
		return status
	}
	if status != nil {
		return fmt.Errorf("Unexpected error: %w", status)
	}
	result, err := SandboxExec(c, "auth/current", "--apihost", "--name")
	if err != nil || len(result.Error) > 0 {
		return ErrSandboxNotConnected
	}
	if result.Entity == nil {
		return errors.New("Could not retrieve information about the connected namespace")
	}
	mapResult := result.Entity.(map[string]interface{})
	fmt.Fprintf(c.Out, "Connected to function namespace '%s' on API host '%s'\n\n", mapResult["name"], mapResult["apihost"])
	displayRuntimes, _ := c.Doit.GetBool(c.NS, "languages")
	if displayRuntimes {
		result, err = SandboxExec(c, "info", "--runtimes")
		if result.Error == "" && err == nil {
			fmt.Fprintf(c.Out, "Available runtimes:\n")
			c.PrintSandboxTextOutput(result)
		}
	}
	return nil
}

// RunSandboxUndeploy implements the 'doctl sandbox undeploy' command
func RunSandboxUndeploy(c *CmdConfig) error {
	haveArgs := len(c.Args) > 0
	pkgFlag, _ := c.Doit.GetBool(c.NS, "packages")
	all, _ := c.Doit.GetBool(c.NS, "all")
	if haveArgs && all {
		return errUndeployAllAndArgs
	}
	if !haveArgs && !all {
		return errUndeployTooFewArgs
	}
	if all {
		return cleanNamespace(c)
	}
	var lastError error
	errorCount := 0
	for _, arg := range c.Args {
		var err error
		if strings.Contains(arg, "/") || !pkgFlag {
			err = deleteFunction(c, arg)
		} else {
			err = deletePackage(c, arg)
		}
		if err != nil {
			lastError = err
			errorCount++
		}
	}
	if errorCount > 0 {
		return fmt.Errorf("there were %d errors detected, e.g.: %w", errorCount, lastError)
	}
	if all {
		fmt.Fprintln(c.Out, "All sandbox content has been undeployed")
	} else {
		fmt.Fprintln(c.Out, "The requested resources have been undeployed")
	}
	return nil
}

// cleanNamespace is a subroutine of RunSandboxDeploy for clearing the entire namespace
func cleanNamespace(c *CmdConfig) error {
	result, err := SandboxExec(c, "namespace/clean", "--force")
	if err != nil {
		return err
	}
	if result.Error != "" {
		return fmt.Errorf(result.Error)
	}
	return nil
}

// deleteFunction is a subroutine of RunSandboxDeploy for deleting one function
func deleteFunction(c *CmdConfig, fn string) error {
	result, err := SandboxExec(c, "action/delete", fn)
	if err != nil {
		return err
	}
	if result.Error != "" {
		return fmt.Errorf(result.Error)
	}
	return nil
}

// deletePackage is a subroutine of RunSandboxDeploy for deleting a package
func deletePackage(c *CmdConfig, pkg string) error {
	result, err := SandboxExec(c, "package/delete", pkg, "--recursive")
	if err != nil {
		return err
	}
	if result.Error != "" {
		return fmt.Errorf(result.Error)
	}
	return nil
}

// "Public" functions

// SandboxExec executes a sandbox command
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

// RunSandboxExec is a variant of SandboxExec convenient for calling from stylized command runners
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

// RunSandboxExecStreaming is like RunSandboxExec but assumes that output will not be captured and can be streamed.
func RunSandboxExecStreaming(command string, c *CmdConfig, booleanFlags []string, stringFlags []string) error {
	err := c.checkSandboxStatus()
	if err != nil {
		return err
	}
	sandbox := c.Sandbox()

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

// CheckSandboxStatus checks install status and returns an appropriate error for common issues
// such as not installed or needs upgrade.  Returns nil when no error.
func CheckSandboxStatus() error {
	sandboxDir, exists := getSandboxDirectory()
	if !exists {
		return ErrSandboxNotInstalled
	}
	if !sandboxUptodate(sandboxDir) {
		return ErrSandboxNeedsUpgrade
	}
	return nil
}

// InstallSandbox is the working subroutine for 'install' and 'upgrade'
func InstallSandbox(c *CmdConfig, sandboxDir string, upgrading bool) error {
	// Make a temporary directory for use during the install
	tmp, err := ioutil.TempDir("", "doctl-sandbox")
	if err != nil {
		return err
	}

	// Download the nodejs tarball for this os and architecture
	fmt.Print("Downloading...")

	goos := runtime.GOOS
	arch := runtime.GOARCH
	if arch == "amd64" {
		arch = "x64"
	}
	if goos == "windows" {
		goos = "win"
	}

	var (
		nodeURL      string
		nodeFileName string
		nodeDir      string
	)

	// Download nodejs only if necessary
	if !upgrading || !canReuseNode(sandboxDir) {
		nodeDir = fmt.Sprintf("node-%s-%s-%s", nodeVersion, goos, arch)
		nodeURL = fmt.Sprintf("https://nodejs.org/dist/%s/%s.tar.gz", nodeVersion, nodeDir)
		nodeFileName = filepath.Join(tmp, "node-install.tar.gz")

		if goos == "win" {
			nodeURL = fmt.Sprintf("https://nodejs.org/dist/%s/%s.zip", nodeVersion, nodeDir)
			nodeFileName = filepath.Join(tmp, "node-install.zip")
		}

		err = download(nodeURL, nodeFileName)
		if err != nil {
			return err
		}
	}

	// Download the fat tarball with the nim CLI, deployer, and sandbox bridge
	// TODO do these need to be arch-specific?  Currently assuming not.
	URL := fmt.Sprintf("https://do-serverless-tools.nyc3.digitaloceanspaces.com/doctl-sandbox-%s.tar.gz", minSandboxVersion)
	sandboxFileName := filepath.Join(tmp, "doctl-sandbox.tar.gz")
	err = download(URL, sandboxFileName)
	if err != nil {
		return err
	}

	// Exec the Extract utility at least once to unpack the fat tarball and possibly a second time if
	// node was downloaded.  If node was not downloaded, just move the existing binary into place.
	fmt.Print("Unpacking...")
	err = extract.Extract(sandboxFileName, tmp)
	if err != nil {
		return err
	}

	if nodeFileName != "" {
		err = extract.Extract(nodeFileName, tmp)
		if err != nil {
			return err
		}
	}

	// Move artifacts to final location
	fmt.Print("Installing...")
	srcPath := filepath.Join(tmp, "sandbox")
	if upgrading {
		// Preserve credentials by moving them from target (which will be replaced) to source.
		err = preserveCreds(c, srcPath, sandboxDir)
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
	} else {
		// Make new empty credentials directory
		emptyCreds := filepath.Join(srcPath, credsDir)
		err = os.Mkdir(emptyCreds, 0700)
		if err != nil {
			return nil
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
		if goos == "win" {
			srcPath = filepath.Join(tmp, nodeDir, "node.exe")
			destPath = filepath.Join(sandboxDir, "node.exe")
		}
		err = os.Rename(srcPath, destPath)
		if err != nil {
			return err
		}
	}
	fmt.Println("\nDone")
	return nil
}

// preserveCreds preserves existing credentials in a sandbox directory
// that is about to be replaced by moving them to the staging directory
// containing the replacement.
func preserveCreds(c *CmdConfig, stagingDir string, sandboxDir string) error {
	credPath := filepath.Join(sandboxDir, credsDir)
	relocPath := filepath.Join(stagingDir, credsDir)
	err := os.Rename(credPath, relocPath)
	if err == nil {
		return nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	// There was no creds directory.  Check for legacy form and convert as part
	// of preserving.
	legacyCredPath := filepath.Join(sandboxDir, ".nimbella")
	err = os.Mkdir(relocPath, 0700)
	if err != nil {
		return err
	}
	moveLegacyTo := getCredentialDirectory(c, stagingDir)
	return os.Rename(legacyCredPath, moveLegacyTo)
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
		return installed == nodeVersion
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
		return fmt.Errorf("Received status code %d attempting to download from %s",
			response.StatusCode, URL)
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

// getCredentialDirectory returns the directory in which credentials should be stored for a given
// CmdConfig.  The actual leaf directory is a function of the access token being used.  This ties
// sandbox credentials to DO credentials
func getCredentialDirectory(c *CmdConfig, sandboxDir string) string {
	token := c.getContextAccessToken()
	hasher := sha1.New()
	hasher.Write([]byte(token))
	sha := hasher.Sum(nil)
	leafDir := hex.EncodeToString(sha[:4])
	return filepath.Join(sandboxDir, credsDir, leafDir)
}

// sandboxUpToDate answers whether the installed version of the sandbox is at least
// what is required by doctl
func sandboxUptodate(sandboxDir string) bool {
	return getCurrentSandboxVersion(sandboxDir) >= minSandboxVersion
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

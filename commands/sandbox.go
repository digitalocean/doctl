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

var (
	// ErrSandboxNotInstalled is the error returned to users when the sandbox is not installed.
	ErrSandboxNotInstalled = errors.New("Serverless support is not installed (use `doctl serverless install`)")
	// ErrSandboxNeedsUpgrade is the error returned to users when the sandbox is at too low a version
	ErrSandboxNeedsUpgrade = errors.New("Serverless support needs to be upgraded (use `doctl serverless upgrade`)")
	// ErrSandboxNotConnected is the error returned to users when the sandbox is not connected to a namespace
	ErrSandboxNotConnected = errors.New("Serverless support is installed but not connected to a functions namespace (use `doctl serverless connect`)")
	// errUndeployAllAndArgs is the error returned when the --all flag is used along with args on undeploy
	errUndeployAllAndArgs = errors.New("command line arguments and the `--all` flag are mutually exclusive")
	// errUndeployTooFewArgs is the error returned when neither --all nor args are specified on undeploy
	errUndeployTooFewArgs = errors.New("either command line arguments or `--all` must be specified")

	// languageKeywords maps the backend's runtime category names to keywords accepted as languages
	// Note: this table has all languages for which we possess samples.  Only those with currently
	// active runtimes will display.
	languageKeywords map[string][]string = map[string][]string{
		"nodejs":     {"javascript", "js"},
		"deno":       {"deno"},
		"go":         {"go", "golang"},
		"java":       {"java"},
		"php":        {"php"},
		"python":     {"python", "py"},
		"ruby":       {"ruby"},
		"rust":       {"rust"},
		"swift":      {"swift"},
		"dotnet":     {"csharp", "cs"},
		"typescript": {"typescript", "ts"},
	}
)

// Sandbox contains support for 'serverless' commands provided by a hidden install of the Nimbella CLI
func Sandbox() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:   "serverless",
			Short: "Develop and test serverless functions",
			Long: `The ` + "`" + `doctl serverless` + "`" + ` commands provide an environment for developing and testing serverless functions.
One or more local file system areas are employed, along with a 'functions namespace' in the cloud.
A one-time install of the serverless software is needed (use ` + "`" + `doctl serverless install` + "`" + ` to install the software,
then ` + "`" + `doctl serverless connect` + "`" + ` to connect to a functions namespace provided with your account).
Other ` + "`" + `doctl serverless` + "`" + ` commands are used to develop and test.`,
			Aliases: []string{"sandbox", "sbx", "sls"},
		},
	}

	cmdBuilderWithInit(cmd, RunSandboxInstall, "install", "Installs the serverless support",
		`This command installs additional software under `+"`"+`doctl`+"`"+` needed to make the other serverless commands work.
The install operation is long-running, and a network connection is required.`,
		Writer, false)

	CmdBuilder(cmd, RunSandboxUpgrade, "upgrade", "Upgrades serverless support to match this version of doctl",
		`This command upgrades the serverless support software under `+"`"+`doctl`+"`"+` by installing over the existing version.
The install operation is long-running, and a network connection is required.`,
		Writer)

	CmdBuilder(cmd, RunSandboxUninstall, "uninstall", "Removes the serverless support", `Removes serverless support from `+"`"+`doctl`+"`",
		Writer)

	CmdBuilder(cmd, RunSandboxConnect, "connect", "Connects local serverless support to your functions namespace",
		`This command connects `+"`"+`doctl serverless`+"`"+` to your functions namespace (needed for testing).`,
		Writer)

	status := CmdBuilder(cmd, RunSandboxStatus, "status", "Provide information about serverless support",
		`This command reports the status of serverless support and some details concerning its connected functions namespace.
With the `+"`"+`--languages flag, it will report the supported languages.
With the `+"`"+`--version flag, it will show just version information about the serverless component`, Writer)
	AddBoolFlag(status, "languages", "l", false, "show available languages (if connected to the cloud)")
	AddBoolFlag(status, "version", "", false, "just show the version, don't check status")

	undeploy := CmdBuilder(cmd, RunSandboxUndeploy, "undeploy [<package|function>...]",
		"Removes resources from your functions namespace",
		`This command removes functions, entire packages, or all functions and packages, from your function
namespace.  In general, deploying new content does not remove old content although it may overwrite it.
Use `+"`"+`doctl serverless undeploy`+"`"+` to effect removal.  The command accepts a list of functions or packages.
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
	status := c.checkSandboxStatus(c)
	switch status {
	case nil:
		fmt.Fprintln(c.Out, "Serverless support is already installed at an appropriate version.  No action needed.")
		return nil
	case ErrSandboxNeedsUpgrade:
		fmt.Fprintln(c.Out, "Serverless support is already installed, but needs an upgrade for this version of `doctl`.")
		fmt.Fprintln(c.Out, "Use `doctl serverless upgrade` to upgrade the support.")
		return nil
	case ErrSandboxNotConnected:
		fmt.Fprintln(c.Out, "Serverless support is already installed at an appropriate version, but not connected to a functions namespace.  Use `doctl serverless connect`.")
		return nil
	}

	sandboxDir, _ := getSandboxDirectory()

	return c.installSandbox(c, sandboxDir, false)
}

// RunSandboxUpgrade is a variant on RunSandboxInstall for installing over an existing version when
// the existing version is inadequate as detected by checkSandboxStatus()
func RunSandboxUpgrade(c *CmdConfig) error {
	status := c.checkSandboxStatus(c)
	switch status {
	case nil:
		fmt.Fprintln(c.Out, "Serverless support is already installed at an appropriate version.  No action needed.")
		// TODO should there be an option to upgrade beyond the minimum needed?
		return nil
	case ErrSandboxNotInstalled:
		fmt.Fprintln(c.Out, "Serverless support was never installed.  Use `doctl serverless install`.")
		return nil
	case ErrSandboxNotConnected:
		fmt.Fprintln(c.Out, "Serverless support is already installed at an appropriate version, but not connected to a functions namespace.  Use `doctl serverless connect`.")
		return nil
	}

	sandboxDir, _ := getSandboxDirectory()

	return c.installSandbox(c, sandboxDir, true)
}

// RunSandboxUninstall removes the sandbox support and any stored credentials
func RunSandboxUninstall(c *CmdConfig) error {
	sandboxDir, exists := getSandboxDirectory()
	if !exists {
		return errors.New("Nothing to uninstall: no serverless support was found")
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

	// Non-standard check for the connect command (only): it's ok to not be connected.
	err = c.checkSandboxStatus(c)
	if err != nil && err != ErrSandboxNotConnected {
		return err
	}

	// Get the credentials for the sandbox namespace
	sandbox := c.Sandbox()
	creds, err = sandbox.GetSandboxNamespace(context.TODO())
	if err != nil {
		return err
	}

	// Store the credentials
	err = sandbox.WriteCredentials(creds)
	if err != nil {
		return err
	}

	fmt.Fprintf(c.Out, "Connected to functions namespace '%s' on API host '%s'\n", creds.Namespace, creds.APIHost)
	fmt.Fprintln(c.Out)
	return nil
}

// RunSandboxStatus gives a report on the status of the sandbox (installed, up to date, connected)
func RunSandboxStatus(c *CmdConfig) error {
	status := c.checkSandboxStatus(c)
	if status == ErrSandboxNotInstalled {
		return status
	}
	version, _ := c.Doit.GetBool(c.NS, "version")
	if version {
		if status == ErrSandboxNeedsUpgrade {
			sandboxDir, _ := getSandboxDirectory() // we know it exists
			currentVersion := getCurrentSandboxVersion(sandboxDir)
			fmt.Fprintf(c.Out, "Current: %s, required: %s\n", currentVersion, getMinSandboxVersion())
			return nil
		}
		fmt.Fprintln(c.Out, getMinSandboxVersion())
		return nil
	}
	if status == ErrSandboxNeedsUpgrade || status == ErrSandboxNotConnected {
		return status
	}
	if status != nil {
		return fmt.Errorf("Unexpected error: %w", status)
	}
	// Check the connected state more deeply (since this is a status command we want to
	// be more accurate; the connected check in checkSandboxStatus is lightweight and heuristic).
	result, err := SandboxExec(c, "auth/current", "--apihost", "--name")
	if err != nil || len(result.Error) > 0 {
		return ErrSandboxNotConnected
	}
	if result.Entity == nil {
		return errors.New("Could not retrieve information about the connected namespace")
	}
	mapResult := result.Entity.(map[string]interface{})
	apiHost := mapResult["apihost"].(string)
	fmt.Fprintf(c.Out, "Connected to functions namespace '%s' on API host '%s'\n", mapResult["name"], apiHost)
	fmt.Fprintf(c.Out, "Serverless software version is %s\n\n", minSandboxVersion)
	languages, _ := c.Doit.GetBool(c.NS, "languages")
	if languages {
		return showLanguageInfo(c, apiHost)
	}
	return nil
}

// showLanguageInfo is called by RunSandboxStatus when --languages is specified
func showLanguageInfo(c *CmdConfig, APIHost string) error {
	info, err := c.Sandbox().GetHostInfo(APIHost)
	if err != nil {
		return err
	}
	fmt.Fprintf(c.Out, "Supported Languages:\n")
	for language := range info.Runtimes {
		fmt.Fprintf(c.Out, "%s:\n", language)
		keywords := strings.Join(languageKeywords[language], ", ")
		fmt.Fprintf(c.Out, "  Keywords: %s\n", keywords)
		fmt.Fprintf(c.Out, "  Runtime versions:\n")
		runtimes := info.Runtimes[language]
		for _, runtime := range runtimes {
			tag := ""
			if runtime.Default {
				tag = fmt.Sprintf(" (%s:default)", language)
			}
			if runtime.Deprecated {
				tag = " (deprecated)"
			}
			fmt.Fprintf(c.Out, "    %s%s\n", runtime.Kind, tag)
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
		fmt.Fprintln(c.Out, "All resources in the functions namespace have been undeployed")
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
	err := c.checkSandboxStatus(c)
	if err != nil {
		return do.SandboxOutput{}, err
	}
	return sandboxExecNoCheck(c, command, args)
}

func sandboxExecNoCheck(c *CmdConfig, command string, args []string) (do.SandboxOutput, error) {
	sandbox := c.Sandbox()
	cmd, err := sandbox.Cmd(command, args)
	if err != nil {
		return do.SandboxOutput{}, err
	}
	return sandbox.Exec(cmd)
}

// RunSandboxExec is a variant of SandboxExec convenient for calling from stylized command runners
// Sets up the arguments and (especially) the flags for the actual call
func RunSandboxExec(command string, c *CmdConfig, booleanFlags []string, stringFlags []string) (do.SandboxOutput, error) {
	err := c.checkSandboxStatus(c)
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
	err := c.checkSandboxStatus(c)
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
func CheckSandboxStatus(c *CmdConfig) error {
	sandboxDir, exists := getSandboxDirectory()
	if !exists {
		return ErrSandboxNotInstalled
	}
	if !sandboxUptodate(sandboxDir) {
		return ErrSandboxNeedsUpgrade
	}
	if !isSandboxConnected(c, sandboxDir) {
		return ErrSandboxNotConnected
	}
	return nil
}

// InstallSandbox is the working subroutine for 'install' and 'upgrade'
func InstallSandbox(c *CmdConfig, sandboxDir string, upgrading bool) error {
	// Make a temporary directory for use during the install.
	// Note: we don't let this be allocated in the system temporaries area because
	// that might be on a separate file system, meaning that the final install step
	// will require an additional copy rather than a simple rename.
	tmp, err := ioutil.TempDir(configHome(), "sbx-install")
	if err != nil {
		return err
	}

	// Download the nodejs tarball for this os and architecture
	fmt.Print("Downloading...")

	goos := runtime.GOOS
	arch := runtime.GOARCH
	nodeBin := "node"
	if arch == "amd64" {
		arch = "x64"
	}
	if arch == "386" {
		if goos == "linux" {
			return errors.New("serverless support is not available for 32-bit linux")
		}
		arch = "x86"
	}
	if goos == "windows" {
		goos = "win"
		nodeBin = "node.exe"
	}

	var (
		nodeURL      string
		nodeFileName string
		nodeDir      string
	)

	// Download nodejs only if necessary
	if !upgrading || !canReuseNode(sandboxDir, nodeBin) {
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
	URL := fmt.Sprintf("https://do-serverless-tools.nyc3.digitaloceanspaces.com/doctl-sandbox-%s.tar.gz",
		getMinSandboxVersion())
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
			err = moveExistingNode(sandboxDir, srcPath, nodeBin)
			if err != nil {
				return err
			}
		}
	} else {
		// Make new empty credentials directory
		emptyCreds := filepath.Join(srcPath, credsDir)
		err = os.MkdirAll(emptyCreds, 0700)
		if err != nil {
			return nil
		}

		// Create the sandbox directory if necessary.
		err := os.MkdirAll(sandboxDir, 0755)
		if err != nil {
			return err
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
		if goos == "win" {
			srcPath = filepath.Join(tmp, nodeDir, nodeBin)
		} else {
			// Additional nesting in non-windows case
			srcPath = filepath.Join(tmp, nodeDir, "bin", nodeBin)
		}
		destPath := filepath.Join(sandboxDir, nodeBin)
		err = os.Rename(srcPath, destPath)
		if err != nil {
			return err
		}
	}
	// Clean up temp directory
	fmt.Print("Cleaning up...")
	os.RemoveAll(tmp) // Best effort, ignore error
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
	err = os.MkdirAll(relocPath, 0700)
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

// Gets the version of the node binary in the sandbox.  Determine if it is
// usable or whether it has to be upgraded.
func canReuseNode(sandboxDir string, nodeBin string) bool {
	fullNodeBin := filepath.Join(sandboxDir, nodeBin)
	cmd := exec.Command(fullNodeBin, "--version")
	result, err := cmd.Output()
	if err == nil {
		installed := strings.TrimSpace(string(result))
		return installed == nodeVersion
	}
	return false
}

// Moves the existing node binary from the sandbox that contains it to the new sandbox being
// staged during an upgrade.  This preserves it for reuse and avoids the need to download.
func moveExistingNode(existing string, staging string, nodeBin string) error {
	srcPath := filepath.Join(existing, nodeBin)
	destPath := filepath.Join(staging, nodeBin)
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
// are stored.  Returns the name of the directory and whether it exists.  The standard location
// (and the only one that customers are expected to use) is relative to the defaultConfigHome.
// For testing purposes, an override can be provided via an environment variable.
func getSandboxDirectory() (string, bool) {
	sandboxDir, shouldOverride := os.LookupEnv("OVERRIDE_SANDBOX_DIR")
	if !shouldOverride {
		sandboxDir = filepath.Join(defaultConfigHome(), "sandbox")
	}
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

	// When running as a snap, the credential are stored separately from the
	// actual sandbox install. So we ignore any override of the sandboxDir here.
	_, isSnap := os.LookupEnv("SNAP")
	if isSnap {
		sandboxDir = filepath.Join(configHome(), "sandbox")
	}

	return filepath.Join(sandboxDir, credsDir, leafDir)
}

// sandboxUpToDate answers whether the installed version of the sandbox is at least
// what is required by doctl
func sandboxUptodate(sandboxDir string) bool {
	return getCurrentSandboxVersion(sandboxDir) >= getMinSandboxVersion()
}

// Determines whether the sandbox appears to be connected.  The purpose is
// to fail fast (when feasible) on sandboxes that are clearly not connected.
// However, it is important not to add excessive overhead on each call (e.g.
// asking the plugin to validate credentials), so the test is not foolproof.
// It merely tests whether a credentials directory has been created for the
// current doctl access token and appears to have a credentials.json in it.
func isSandboxConnected(c *CmdConfig, sandboxDir string) bool {
	creds := getCredentialDirectory(c, sandboxDir)
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

// Return the minSandboxVersion (allows the constant to be overridden via an environment variable)
func getMinSandboxVersion() string {
	fromEnv := os.Getenv("minSandboxVersion")
	if fromEnv != "" {
		return fromEnv
	}
	return minSandboxVersion
}

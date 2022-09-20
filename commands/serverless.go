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
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/do"
	"github.com/spf13/cobra"
)

var (
	// errUndeployAllAndArgs is the error returned when the --all flag is used along with args on undeploy
	errUndeployAllAndArgs = errors.New("command line arguments and the `--all` flag are mutually exclusive")

	// errUndeployTooFewArgs is the error returned when neither --all nor args are specified on undeploy
	errUndeployTooFewArgs = errors.New("either command line arguments or `--all` must be specified")

	// errUndeployTrigPkg is the error returned when both --packages and --triggers are specified on undeploy
	errUndeployTrigPkg = errors.New("the `--packages` and `--triggers` flags are mutually exclusive")

	// languageKeywords maps the backend's runtime category names to keywords accepted as languages
	// Note: this table has all languages for which we possess samples.  Only those with currently
	// active runtimes will display.
	languageKeywords map[string][]string = map[string][]string{
		"nodejs": {"javascript", "js", "typescript", "ts"},
		"deno":   {"deno"},
		"go":     {"go", "golang"},
		"java":   {"java"},
		"php":    {"php"},
		"python": {"python", "py"},
		"ruby":   {"ruby"},
		"rust":   {"rust"},
		"swift":  {"swift"},
		"dotnet": {"csharp", "cs"},
	}
)

// Serverless contains support for 'serverless' commands provided by a hidden install of the Nimbella CLI
func Serverless() *Command {
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

	cmdBuilderWithInit(cmd, RunServerlessInstall, "install", "Installs the serverless support",
		`This command installs additional software under `+"`"+`doctl`+"`"+` needed to make the other serverless commands work.
The install operation is long-running, and a network connection is required.`,
		Writer, false)

	CmdBuilder(cmd, RunServerlessUpgrade, "upgrade", "Upgrades serverless support to match this version of doctl",
		`This command upgrades the serverless support software under `+"`"+`doctl`+"`"+` by installing over the existing version.
The install operation is long-running, and a network connection is required.`,
		Writer)

	CmdBuilder(cmd, RunServerlessUninstall, "uninstall", "Removes the serverless support", `Removes serverless support from `+"`"+`doctl`+"`",
		Writer)

	CmdBuilder(cmd, RunServerlessConnect, "connect", "Connects local serverless support to your functions namespace",
		`This command connects `+"`"+`doctl serverless`+"`"+` to your functions namespace (needed for testing).`,
		Writer)

	status := CmdBuilder(cmd, RunServerlessStatus, "status", "Provide information about serverless support",
		`This command reports the status of serverless support and some details concerning its connected functions namespace.
With the `+"`"+`--languages flag, it will report the supported languages.
With the `+"`"+`--version flag, it will show just version information about the serverless component`, Writer)
	AddBoolFlag(status, "languages", "l", false, "show available languages (if connected to the cloud)")
	AddBoolFlag(status, "version", "", false, "just show the version, don't check status")

	undeploy := CmdBuilder(cmd, RunServerlessUndeploy, "undeploy [<package|function>...]",
		"Removes resources from your functions namespace",
		`This command removes functions, entire packages, or all functions and packages, from your function
namespace.  In general, deploying new content does not remove old content although it may overwrite it.
Use `+"`"+`doctl serverless undeploy`+"`"+` to effect removal.  The command accepts a list of functions or packages.
Functions should be listed in `+"`"+`pkgName/fnName`+"`"+` form, or `+"`"+`fnName`+"`"+` for a function not in any package.
The `+"`"+`--packages`+"`"+` flag causes arguments without slash separators to be intepreted as packages, in which case
the entire packages are removed.`, Writer)
	AddBoolFlag(undeploy, "packages", "p", false, "interpret simple name arguments as packages")
	AddBoolFlag(undeploy, "triggers", "", false, "interpret all arguments as triggers")
	AddBoolFlag(undeploy, "all", "", false, "remove all packages and functions")
	undeploy.Flags().MarkHidden("triggers") // support is experimental at this point

	cmd.AddCommand(Activations())
	cmd.AddCommand(Functions())
	cmd.AddCommand(Namespaces())
	cmd.AddCommand(Triggers())
	ServerlessExtras(cmd)
	return cmd
}

// RunServerlessInstall performs the network installation of the 'nim' adjunct to support serverless development
func RunServerlessInstall(c *CmdConfig) error {
	var (
		serverless   do.ServerlessService
		credsLeafDir string
		status       error
	)

	// When building the snap package, we need to install the serverless plugin
	// without a fully configured and authenticated doctl. So we only fully init
	// the service if SNAP_SANDBOX_INSTALL is not set.
	_, isSnapInstall := os.LookupEnv("SNAP_SANDBOX_INSTALL")
	if isSnapInstall {
		serverlessDir := os.Getenv("OVERRIDE_SANDBOX_DIR")
		serverless = do.NewServerlessService(nil, serverlessDir, "")
		status = do.ErrServerlessNotInstalled
	} else {
		if err := c.initServices(c); err != nil {
			return err
		}
		credsLeafDir = hashAccessToken(c)
		serverless = c.Serverless()
		status = serverless.CheckServerlessStatus(credsLeafDir)
	}
	switch status {
	case nil:
		fmt.Fprintln(c.Out, "Serverless support is already installed at an appropriate version.  No action needed.")
		return nil
	case do.ErrServerlessNeedsUpgrade:
		fmt.Fprintln(c.Out, "Serverless support is already installed, but needs an upgrade for this version of `doctl`.")
		fmt.Fprintln(c.Out, "Use `doctl serverless upgrade` to upgrade the support.")
		return nil
	case do.ErrServerlessNotConnected:
		fmt.Fprintln(c.Out, "Serverless support is already installed at an appropriate version, but not connected to a functions namespace.  Use `doctl serverless connect`.")
		return nil
	}
	return serverless.InstallServerless(credsLeafDir, false)
}

// RunServerlessUpgrade is a variant on RunServerlessInstall for installing over an existing version when
// the existing version is inadequate as detected by checkServerlessStatus()
func RunServerlessUpgrade(c *CmdConfig) error {
	credsLeafDir := hashAccessToken(c)
	serverless := c.Serverless()
	status := serverless.CheckServerlessStatus(credsLeafDir)
	switch status {
	case nil:
		fmt.Fprintln(c.Out, "Serverless support is already installed at an appropriate version.  No action needed.")
		// TODO should there be an option to upgrade beyond the minimum needed?
		return nil
	case do.ErrServerlessNotInstalled:
		fmt.Fprintln(c.Out, "Serverless support was never installed.  Use `doctl serverless install`.")
		return nil
	case do.ErrServerlessNotConnected:
		fmt.Fprintln(c.Out, "Serverless support is already installed at an appropriate version, but not connected to a functions namespace.  Use `doctl serverless connect`.")
		return nil
	}
	return serverless.InstallServerless(credsLeafDir, true)
}

// RunServerlessUninstall removes the serverless support and any stored credentials
func RunServerlessUninstall(c *CmdConfig) error {
	err := c.Serverless().CheckServerlessStatus(hashAccessToken(c))
	if err == do.ErrServerlessNotInstalled {
		return errors.New("Nothing to uninstall: no serverless support was found")
	}
	return os.RemoveAll(getServerlessDirectory())
}

// RunServerlessConnect implements the serverless connect command
func RunServerlessConnect(c *CmdConfig) error {
	var (
		err error
	)

	if len(c.Args) > 1 {
		return doctl.NewTooManyArgsErr(c.NS)
	}

	sls := c.Serverless()

	// Non-standard check for the connect command (only): it's ok to not be connected.
	err = sls.CheckServerlessStatus(hashAccessToken(c))
	if err != nil && err != do.ErrServerlessNotConnected {
		return err
	}

	ctx := context.TODO()

	// If an arg is specified, retrieve the namespaces that match and proceed according to whether there
	// are 0, 1, or >1 matches.
	if len(c.Args) > 0 {
		list, err := getMatchingNamespaces(ctx, sls, c.Args[0])
		if err != nil {
			return err
		}
		if len(list) == 0 {
			return fmt.Errorf("you have no namespaces matching '%s'", c.Args[0])
		}
		return connectFromList(ctx, sls, list, c.Out)
	}
	list, err := getMatchingNamespaces(ctx, sls, "")
	if err != nil {
		return err
	}
	if len(list) == 0 {
		return errors.New("you must create a namespace with `doctl namespace create`, specifying a region and label")
	}
	return connectFromList(ctx, sls, list, c.Out)
}

// connectFromList connects a namespace based on a non-empty list of namespaces.  If the list is
// singular that determines the namespace that will be connected.  Otherwise, this is determined
// via a prompt.
func connectFromList(ctx context.Context, sls do.ServerlessService, list []do.OutputNamespace, out io.Writer) error {
	var ns do.OutputNamespace
	if len(list) == 1 {
		ns = list[0]
	} else {
		ns = chooseFromList(list, out)
		if ns.Namespace == "" {
			return nil
		}
	}
	creds, err := sls.GetNamespace(ctx, ns.Namespace)
	if err != nil {
		return err
	}
	return finishConnecting(sls, creds, ns.Label, out)
}

// connectChoiceReader is the bufio.Reader for reading the user's response to the prompt to choose
// a namespace.  It can be replaced for testing.
var connectChoiceReader *bufio.Reader = bufio.NewReader(os.Stdin)

// chooseFromList displays a list of namespaces (label, region, id) assigning each one a number.
// The user can than respond to a prompt that chooses from the list by number.  The response 'x' is
// also accepted and exits the command.
func chooseFromList(list []do.OutputNamespace, out io.Writer) do.OutputNamespace {
	for i, ns := range list {
		fmt.Fprintf(out, "%d: %s in %s, label=%s\n", i, ns.Namespace, ns.Region, ns.Label)
	}
	for {
		fmt.Fprintln(out, "Choose a namespace by number or 'x' to exit")
		choice, err := connectChoiceReader.ReadString('\n')
		if err != nil {
			continue
		}
		choice = strings.TrimSpace(choice)
		if choice == "x" {
			return do.OutputNamespace{}
		}
		i, err := strconv.Atoi(choice)
		if err == nil && i >= 0 && i < len(list) {
			return list[i]
		}
	}
}

// finishConnecting performs the final steps of 'doctl serverless connect'.
func finishConnecting(sls do.ServerlessService, creds do.ServerlessCredentials, label string, out io.Writer) error {
	// Store the credentials
	err := sls.WriteCredentials(creds)
	if err != nil {
		return err
	}

	labelTag := ""
	if label != "" {
		labelTag = " (label=" + label + ")"
	}
	fmt.Fprintf(out, "Connected to functions namespace '%s' on API host '%s'%s\n", creds.Namespace, creds.APIHost, labelTag)
	fmt.Fprintln(out)
	return nil
}

// RunServerlessStatus gives a report on the status of the serverless (installed, up to date, connected)
func RunServerlessStatus(c *CmdConfig) error {
	status := c.Serverless().CheckServerlessStatus(hashAccessToken(c))
	if status == do.ErrServerlessNotInstalled {
		return status
	}
	version, _ := c.Doit.GetBool(c.NS, "version")
	if version {
		if status == do.ErrServerlessNeedsUpgrade {
			serverlessDir := getServerlessDirectory() // we know it exists
			currentVersion := do.GetCurrentServerlessVersion(serverlessDir)
			fmt.Fprintf(c.Out, "Current: %s, required: %s\n", currentVersion, do.GetMinServerlessVersion())
			return nil
		}
		fmt.Fprintln(c.Out, do.GetMinServerlessVersion())
		return nil
	}
	if status == do.ErrServerlessNeedsUpgrade || status == do.ErrServerlessNotConnected {
		return status
	}
	if status != nil {
		return fmt.Errorf("Unexpected error: %w", status)
	}
	// Check the connected state more deeply (since this is a status command we want to
	// be more accurate; the connected check in checkServerlessStatus is lightweight and heuristic).
	result, err := ServerlessExec(c, "auth/current", "--apihost", "--name")
	if err != nil || len(result.Error) > 0 {
		return do.ErrServerlessNotConnected
	}
	if result.Entity == nil {
		return errors.New("Could not retrieve information about the connected namespace")
	}
	mapResult := result.Entity.(map[string]interface{})
	apiHost := mapResult["apihost"].(string)
	fmt.Fprintf(c.Out, "Connected to functions namespace '%s' on API host '%s'\n", mapResult["name"], apiHost)
	fmt.Fprintf(c.Out, "Serverless software version is %s\n\n", do.GetMinServerlessVersion())
	languages, _ := c.Doit.GetBool(c.NS, "languages")
	if languages {
		return showLanguageInfo(c, apiHost)
	}
	return nil
}

// showLanguageInfo is called by RunServerlessStatus when --languages is specified
func showLanguageInfo(c *CmdConfig, APIHost string) error {
	info, err := c.Serverless().GetHostInfo(APIHost)
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

// RunServerlessUndeploy implements the 'doctl serverless undeploy' command
func RunServerlessUndeploy(c *CmdConfig) error {
	haveArgs := len(c.Args) > 0
	pkgFlag, _ := c.Doit.GetBool(c.NS, "packages")
	trigFlag, _ := c.Doit.GetBool(c.NS, "triggers")
	all, _ := c.Doit.GetBool(c.NS, "all")
	if haveArgs && all {
		return errUndeployAllAndArgs
	}
	if !haveArgs && !all {
		return errUndeployTooFewArgs
	}
	if pkgFlag && trigFlag {
		return errUndeployTrigPkg
	}
	if all && trigFlag {
		return cleanTriggers(c)
	}
	if all {
		return cleanNamespace(c)
	}
	var lastError error
	errorCount := 0
	var ctx context.Context
	var sls do.ServerlessService
	if trigFlag {
		ctx = context.TODO()
		sls = c.Serverless()
	}
	for _, arg := range c.Args {
		var err error
		if trigFlag {
			err = sls.DeleteTrigger(ctx, arg)
		} else if strings.Contains(arg, "/") || !pkgFlag {
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

// cleanNamespace is a subroutine of RunServerlessDeploy for clearing the entire namespace
func cleanNamespace(c *CmdConfig) error {
	result, err := ServerlessExec(c, "namespace/clean", "--force")
	if err != nil {
		return err
	}
	if result.Error != "" {
		return fmt.Errorf(result.Error)
	}
	return nil
}

// deleteFunction is a subroutine of RunServerlessDeploy for deleting one function
func deleteFunction(c *CmdConfig, fn string) error {
	result, err := ServerlessExec(c, "action/delete", fn)
	if err != nil {
		return err
	}
	if result.Error != "" {
		return fmt.Errorf(result.Error)
	}
	return nil
}

// deletePackage is a subroutine of RunServerlessDeploy for deleting a package
func deletePackage(c *CmdConfig, pkg string) error {
	result, err := ServerlessExec(c, "package/delete", pkg, "--recursive")
	if err != nil {
		return err
	}
	if result.Error != "" {
		return fmt.Errorf(result.Error)
	}
	return nil
}

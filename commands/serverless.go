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
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/commands/charm/template"
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
			Short: "Develop, test, and deploy serverless functions",
			Long: `The ` + "`" + `doctl serverless` + "`" + ` commands provide an environment for developing, testing, and deploying serverless functions.
One or more local file system areas are employed, along with one or more 'functions namespaces' in the cloud.
A one-time install of the serverless software is needed (use ` + "`" + `doctl serverless install` + "`" + ` to install the software,
then ` + "`" + `doctl serverless connect` + "`" + ` to connect to a functions namespace associated with your account).
Other ` + "`" + `doctl serverless` + "`" + ` commands are used to develop, test, and deploy.`,
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

	connect := CmdBuilder(cmd, RunServerlessConnect, "connect [<hint>]", "Connects local serverless support to a functions namespace",
		`This command connects `+"`"+`doctl serverless`+"`"+` support to a functions namespace of your choice.
The optional argument should be a (complete or partial) match to a namespace label or id.
If there is no argument, all namespaces are matched.  If the result is exactly one namespace,
you are connected to it.  If there are multiple namespaces, you have an opportunity to choose
the one you want from a dialog.  Use `+"`"+`doctl serverless namespaces`+"`"+` to create, delete, and
list your namespaces.`,
		Writer)
	// The apihost and auth flags will always be hidden.  They support testing using doctl on clusters that are not in production
	// and hence are unknown to the portal.
	AddStringFlag(connect, "apihost", "", "", "")
	AddStringFlag(connect, "auth", "", "", "")
	connect.Flags().MarkHidden("apihost")
	connect.Flags().MarkHidden("auth")

	status := CmdBuilder(cmd, RunServerlessStatus, "status", "Provide information about serverless support",
		`This command reports the status of serverless support and some details concerning its connected functions namespace.
With the `+"`"+`--languages flag, it will report the supported languages.
With the `+"`"+`--version flag, it will show just version information about the serverless component`, Writer)
	AddBoolFlag(status, "languages", "l", false, "show available languages (if connected to the cloud)")
	AddBoolFlag(status, "version", "", false, "just show the version, don't check status")
	AddBoolFlag(status, "credentials", "", false, "")
	// The --credentials flag is needed when working on clusters that are not in production.
	// It captures the current credentials to be restored later using 'connect --apihost --auth'.
	// It is hidden since this kind of low level manipulation should not normally be necessary.
	status.Flags().MarkHidden("credentials")

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
	AddBoolFlag(undeploy, doctl.ArgForce, doctl.ArgShortForce, false, "Delete namespace resources without confirmation prompt")
	undeploy.Flags().MarkHidden(doctl.ArgForce)
	undeploy.Flags().MarkHidden("triggers") // support is experimental at this point
	// The apihost and auth flags will always be hidden.  They support testing using doctl on clusters that are not in production
	// and hence are unknown to the portal.
	AddStringFlag(undeploy, "apihost", "", "", "")
	AddStringFlag(undeploy, "auth", "", "", "")
	undeploy.Flags().MarkHidden("apihost")
	undeploy.Flags().MarkHidden("auth")

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
	// the service if SNAP_SANDBOX_INSTALL is not set.  We also do the same thing
	// when installing doctl in a docker image build, where we know there is no
	// previous install and we don't want to expose a DO API token.
	_, isSnapInstall := os.LookupEnv("SNAP_SANDBOX_INSTALL")
	_, isDockerInstall := os.LookupEnv("DOCKER_SANDBOX_INSTALL")
	if isSnapInstall || isDockerInstall {
		var serverlessDir string
		if isSnapInstall {
			serverlessDir = os.Getenv("OVERRIDE_SANDBOX_DIR")
		} else {
			serverlessDir = getServerlessDirectory()
		}
		serverless = do.NewServerlessService(nil, serverlessDir, "")
		status = do.ErrServerlessNotInstalled
	} else {
		if err := c.initServices(c); err != nil {
			return err
		}
		credsLeafDir = hashAccessToken(c)
		serverless = c.Serverless()
		status = serverless.CheckServerlessStatus()
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
	status := serverless.CheckServerlessStatus()
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
	err := c.Serverless().CheckServerlessStatus()
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
	sls := c.Serverless()

	// Support the hidden capability to connect to non-production clusters to support various kinds of testing.
	// The presence of 'auth' and 'apihost' flags trumps other parts of the syntax, but both must be present.
	apihost, _ := c.Doit.GetString(c.NS, "apihost")
	auth, _ := c.Doit.GetString(c.NS, "auth")
	if len(apihost) > 0 && len(auth) > 0 {
		namespace, err := sls.GetNamespaceFromCluster(apihost, auth)
		if err != nil {
			return err
		}
		credential := do.ServerlessCredential{Auth: auth}
		creds := do.ServerlessCredentials{
			APIHost:     apihost,
			Namespace:   namespace,
			Credentials: map[string]map[string]do.ServerlessCredential{apihost: {namespace: credential}},
		}
		return finishConnecting(sls, creds, c.Out)
	}
	if len(apihost) > 0 || len(auth) > 0 {
		return fmt.Errorf("If either of 'apihost' or 'auth' is specified then both must be specified")
	}
	// Neither 'auth' nor 'apihost' was specified, so continue with other options.

	if len(c.Args) > 1 {
		return doctl.NewTooManyArgsErr(c.NS)
	}

	// Non-standard check for the connect command (only): it's ok to not be connected.
	err = sls.CheckServerlessStatus()
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
	return finishConnecting(sls, creds, out)
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
func finishConnecting(sls do.ServerlessService, creds do.ServerlessCredentials, out io.Writer) error {
	// Store the credentials
	err := sls.WriteCredentials(creds)
	if err != nil {
		return err
	}

	fmt.Fprintln(out, getConnectedMessage(creds))
	fmt.Fprintln(out)
	return nil
}

// getConnectedMessage formats the output message for being connected to a specific namespace (with optional label)
func getConnectedMessage(creds do.ServerlessCredentials) string {
	labelTag := ""
	if creds.Label != "" {
		labelTag = " (label=" + creds.Label + ")"
	}
	return fmt.Sprintf("Connected to functions namespace '%s' on API host '%s'%s", creds.Namespace, creds.APIHost, labelTag)
}

// RunServerlessStatus gives a report on the status of the serverless (installed, up to date, connected)
func RunServerlessStatus(c *CmdConfig) error {
	sls := c.Serverless()
	status := sls.CheckServerlessStatus()
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
	creds, err := sls.ReadCredentials()
	if err != nil {
		return nil
	}
	auth := creds.Credentials[creds.APIHost][creds.Namespace].Auth
	checkNS, err := sls.GetNamespaceFromCluster(creds.APIHost, auth)
	if err != nil || checkNS != creds.Namespace {
		return do.ErrServerlessNotConnected
	}
	// Before doing the normal display, check for the --credentials flag, which requests return of the
	// 'creds' value as JSON.
	credentials, _ := c.Doit.GetBool(c.NS, "credentials")
	if credentials {
		type showCreds struct {
			Auth      string `json:auth`
			APIHost   string `json:apihost`
			Namespace string `json:namespace`
			Path      string `json:path`
		}
		toShow := showCreds{Auth: auth, APIHost: creds.APIHost, Namespace: creds.Namespace, Path: sls.CredentialsPath()}
		credsOutput, err := json.MarshalIndent(toShow, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(credsOutput))
		return nil
	}

	fmt.Fprintln(c.Out, getConnectedMessage(creds))
	fmt.Fprintf(c.Out, "Serverless software version is %s\n\n", do.GetMinServerlessVersion())
	languages, _ := c.Doit.GetBool(c.NS, "languages")
	if languages {
		return showLanguageInfo(c, creds.APIHost)
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

	sls := c.Serverless()

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

	// Permit cleanup of arbitrary namespaces (needed in some administrative uses)
	auth, _ := c.Doit.GetString(c.NS, "auth")
	apihost, _ := c.Doit.GetString(c.NS, "apihost")
	if auth != "" && apihost != "" {
		sls.SetEffectiveCredentials(auth, apihost)
	} else if auth != "" || apihost != "" {
		return errors.New("the --auth and --apihost flags must be specified together")
	}

	if all {
		err := sls.CleanNamespace()
		if err != nil {
			return err
		}
		template.Print(`{{success checkmark}} All resources in the namespace have been undeployed.{{nl}}`, nil)
		return nil
	}

	var lastError error
	errorCount := 0
	var ctx context.Context

	if trigFlag {
		ctx = context.TODO()
	}

	for _, arg := range c.Args {
		var err error
		if trigFlag {
			err = sls.DeleteTrigger(ctx, arg)
		} else if strings.Contains(arg, "/") || !pkgFlag {
			err = sls.DeleteFunction(arg, true)
		} else {
			err = sls.DeletePackage(arg, true)
		}

		if err != nil {
			lastError = err
			errorCount++
		}
	}

	if errorCount > 0 {
		return fmt.Errorf("there were %d errors detected, e.g.: %w", errorCount, lastError)
	}
	template.Print(`{{success checkmark}} The requested resources have been undeployed.{{nl}}`, nil)
	return nil
}

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
	"github.com/digitalocean/doctl"
	"github.com/spf13/cobra"
)

// Activations generates the sandbox 'activations' subtree for addition to the doctl command
func Activations() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:   "activations",
			Short: "Work with activation records",
			Long: `The subcommands of ` + "`" + `doctl sandbox activations` + "`" + ` will list or retrieve results, logs, or complete
"activation records" which result from invoking functions deployed to your sandbox.`,
			Aliases: []string{"actv"},
		},
	}

	get := CmdBuilder(cmd, RunActivationsGet, "get [<activationId>]", "Retrieves an Activation",
		`Use `+"`"+`doctl sandbox activations get`+"`"+` to retrieve the activation record for a previously invoked function.
There are several options for specifying the activation you want.  You can limit output to the result
or the logs.  The `+"`"+`doctl sandbox activation logs`+"`"+` command has additional advanced capabilities for retrieving
logs.`,
		Writer)
	AddBoolFlag(get, "last", "l", false, "Fetch the most recent activation (default)")
	AddIntFlag(get, "skip", "s", 0, "SKIP number of activations")
	AddBoolFlag(get, "logs", "g", false, "Emit only the logs, stripped of time stamps and stream identifier")
	AddBoolFlag(get, "result", "r", false, "Emit only the result")
	AddStringFlag(get, "function", "f", "", "Fetch activations for a specific function")
	AddBoolFlag(get, "quiet", "q", false, "Suppress last activation information header")

	list := CmdBuilder(cmd, RunActivationsList, "list [<activation_name>]", "Lists Activations for which records exist",
		`Use `+"`"+`doctl sandbox activations list`+"`"+` to list the activation records that are present in the cloud for previously
invoked functions.`,
		Writer)
	AddStringFlag(list, "limit", "l", "", "only return LIMIT number of activations (default 30, max 200)")
	AddStringFlag(list, "skip", "s", "", "exclude the first SKIP number of activations from the result")
	AddStringFlag(list, "since", "", "", "return activations with timestamps later than SINCE; measured in milliseconds since Th, 01, Jan 1970")
	AddStringFlag(list, "upto", "", "", "return activations with timestamps earlier than UPTO; measured in milliseconds since Th, 01, Jan 1970")
	AddBoolFlag(list, "count", "", false, "show only the total number of activations")
	AddBoolFlag(list, "full", "f", false, "include full activation description")

	logs := CmdBuilder(cmd, RunActivationsLogs, "logs [<activationId>]", "Retrieves the Logs for an Activation",
		`Use `+"`"+`doctl sandbox activations logs`+"`"+` to retrieve the logs portion of one or more activation records
with various options, such as selecting by package or function, and optionally watching continuously
for new arrivals.`,
		Writer)
	AddStringFlag(logs, "function", "f", "", "Fetch logs for a specific function")
	AddStringFlag(logs, "package", "p", "", "Fetch logs for a specific package")
	AddBoolFlag(logs, "last", "l", false, "Fetch the most recent activation logs (default)")
	AddIntFlag(logs, "limit", "n", 1, "Fetch the last LIMIT activation logs (up to 200)")
	AddBoolFlag(logs, "strip", "r", false, "strip timestamp information and output first line only")
	AddBoolFlag(logs, "follow", "", false, "Fetch logs continuously")

	result := CmdBuilder(cmd, RunActivationsResult, "result [<activationId>]", "Retrieves the Results for an Activation",
		`Use `+"`"+`doctl sandbox activations result`+"`"+` to retrieve just the results portion
of one or more activation records.`,
		Writer)
	AddBoolFlag(result, "last", "l", false, "Fetch the most recent activation result (default)")
	AddIntFlag(result, "limit", "n", 1, "Fetch the last LIMIT activation results (default 30, max 200)")
	AddIntFlag(result, "skip", "s", 0, "SKIP number of activations")
	AddStringFlag(result, "function", "f", "", "Fetch results for a specific function")
	AddBoolFlag(result, "quiet", "q", false, "Suppress last activation information header")

	return cmd
}

// RunActivationsGet supports the 'activations get' command
func RunActivationsGet(c *CmdConfig) error {
	argCount := len(c.Args)
	if argCount > 1 {
		return doctl.NewTooManyArgsErr(c.NS)
	}
	replaceFunctionWithAction(c)
	output, err := RunSandboxExec(activationGet, c, []string{flagLast, flagLogs, flagResult, flagQuiet}, []string{flagSkip, flagAction})
	if err != nil {
		return err
	}
	return c.PrintSandboxTextOutput(output)
}

// RunActivationsList supports the 'activations list' command
func RunActivationsList(c *CmdConfig) error {
	argCount := len(c.Args)
	if argCount > 1 {
		return doctl.NewTooManyArgsErr(c.NS)
	}
	output, err := RunSandboxExec(activationList, c, []string{flagCount, flagFull}, []string{flagLimit, flagSkip, flagSince, flagUpto})
	if err != nil {
		return err
	}
	return c.PrintSandboxTextOutput(output)
}

// RunActivationsLogs supports the 'activations logs' command
func RunActivationsLogs(c *CmdConfig) error {
	argCount := len(c.Args)
	if argCount > 1 {
		return doctl.NewTooManyArgsErr(c.NS)
	}
	replaceFunctionWithAction(c)
	augmentPackageWithDeployed(c)
	if isWatching(c) {
		return RunSandboxExecStreaming(activationLogs, c, []string{flagLast, flagStrip, flagWatch, flagDeployed}, []string{flagAction, flagPackage, flagLimit})
	}
	output, err := RunSandboxExec(activationLogs, c, []string{flagLast, flagStrip, flagWatch, flagDeployed}, []string{flagAction, flagPackage, flagLimit})
	if err != nil {
		return err
	}
	return c.PrintSandboxTextOutput(output)
}

// isWatching (1) modifies the config replacing the "follow" flag (significant to doctl) with the
// "watch" flag (significant to nim)  (2) Returns whether the command should be run in streaming mode
// (will be true iff follow/watch is true).
func isWatching(c *CmdConfig) bool {
	yes, err := c.Doit.GetBool(c.NS, flagFollow)
	if yes && err == nil {
		c.Doit.Set(c.NS, flagWatch, true)
		return true
	}
	return false
}

// RunActivationsResult supports the 'activations result' command
func RunActivationsResult(c *CmdConfig) error {
	argCount := len(c.Args)
	if argCount > 1 {
		return doctl.NewTooManyArgsErr(c.NS)
	}
	replaceFunctionWithAction(c)
	output, err := RunSandboxExec(activationResult, c, []string{flagLast, flagQuiet}, []string{flagLimit, flagSkip, flagAction})
	if err != nil {
		return err
	}
	return c.PrintSandboxTextOutput(output)
}

// replaceFunctionWithAction detects that --function was specified and renames it to --action (which is what nim
// will expect to see).
func replaceFunctionWithAction(c *CmdConfig) {
	value, err := c.Doit.GetString(c.NS, flagFunction)
	if err == nil && value != "" {
		c.Doit.Set(c.NS, flagFunction, "")
		c.Doit.Set(c.NS, flagAction, value)
	}
}

// augmentPackageWithDeployed detects that --package was specified and adds the --deployed flag if so.
// The code in 'nim' (inherited from Adobe I/O) will otherwise look for a deployment manifest which we
// don't want to support here.
func augmentPackageWithDeployed(c *CmdConfig) {
	value, err := c.Doit.GetString(c.NS, flagPackage)
	if err == nil && value != "" {
		c.Doit.Set(c.NS, flagDeployed, true)
	}
}

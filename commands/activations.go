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
	"io"
	"os"
	"os/signal"
	"regexp"
	"syscall"
	"time"

	"github.com/apache/openwhisk-client-go/whisk"
	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/commands/charm/text"
	"github.com/digitalocean/doctl/commands/displayers"
	"github.com/digitalocean/doctl/do"
	"github.com/spf13/cobra"
)

// ShownActivation is what is actually shown as an activation ... it adds a date field which is a human-readable
// version of the start field.
type ShownActivation struct {
	whisk.Activation
	Date string `json:"date,omitempty"`
}

// Activations generates the serverless 'activations' subtree for addition to the doctl command
func Activations() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:   "activations",
			Short: "Work with activation records",
			Long: `The subcommands of ` + "`" + `doctl serverless activations` + "`" + ` will list or retrieve results, logs, or complete
"activation records" which result from invoking functions deployed to your functions namespace.`,
			Aliases: []string{"activation", "actv"},
		},
	}

	get := CmdBuilder(cmd, RunActivationsGet, "get [<activationId>]", "Retrieves an Activation",
		`Use `+"`"+`doctl serverless activations get`+"`"+` to retrieve the activation record for a previously invoked function.
There are several options for specifying the activation you want.  You can limit output to the result
or the logs.  The `+"`"+`doctl serverless activation logs`+"`"+` command has additional advanced capabilities for retrieving
logs.`,
		Writer)
	AddBoolFlag(get, "last", "l", false, "Fetch the most recent activation (default)")
	AddIntFlag(get, "skip", "s", 0, "SKIP number of activations")
	AddBoolFlag(get, "logs", "g", false, "Emit only the logs, stripped of time stamps and stream identifier")
	AddBoolFlag(get, "result", "r", false, "Emit only the result")
	AddStringFlag(get, "function", "f", "", "Fetch activations for a specific function")
	AddBoolFlag(get, "quiet", "q", false, "Suppress last activation information header")

	list := CmdBuilder(cmd, RunActivationsList, "list [<function_name>]", "Lists Activations for which records exist",
		`Use `+"`"+`doctl serverless activations list`+"`"+` to list the activation records that are present in the cloud for previously
invoked functions.`,
		Writer,
		aliasOpt("ls"),
		displayerType(&displayers.Activation{}),
	)
	AddIntFlag(list, "limit", "l", 30, "only return LIMIT number of activations (default 30, max 200)")
	AddIntFlag(list, "skip", "s", 0, "exclude the first SKIP number of activations from the result")
	AddIntFlag(list, "since", "", 0, "return activations with timestamps later than SINCE; measured in milliseconds since Th, 01, Jan 1970")
	AddIntFlag(list, "upto", "", 0, "return activations with timestamps earlier than UPTO; measured in milliseconds since Th, 01, Jan 1970")
	AddBoolFlag(list, "count", "", false, "show only the total number of activations")
	AddBoolFlag(list, "full", "f", false, "include full activation description")

	logs := CmdBuilder(cmd, RunActivationsLogs, "logs [<activationId>]", "Retrieves the Logs for an Activation",
		`Use `+"`"+`doctl serverless activations logs`+"`"+` to retrieve the logs portion of one or more activation records
with various options, such as selecting by package or function, and optionally watching continuously
for new arrivals.`,
		Writer)
	AddStringFlag(logs, "function", "f", "", "Fetch logs for a specific function")
	AddStringFlag(logs, "package", "p", "", "Fetch logs for a specific package")
	AddBoolFlag(logs, "last", "l", false, "Fetch the most recent activation logs (default)")
	AddIntFlag(logs, "limit", "n", 1, "Fetch the last LIMIT activation logs (up to 200)")
	AddBoolFlag(logs, "strip", "r", false, "strip timestamp information and output first line only")
	AddBoolFlag(logs, "follow", "", false, "Fetch logs continuously")

	// This is the default behavior, so we want to prevent users from explicitly using this flag. We don't want to remove it
	// to maintain backwards compatibility
	logs.Flags().MarkHidden("last")

	result := CmdBuilder(cmd, RunActivationsResult, "result [<activationId>]", "Retrieves the Results for an Activation",
		`Use `+"`"+`doctl serverless activations result`+"`"+` to retrieve just the results portion
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
	var id string
	if argCount > 0 {
		id = c.Args[0]
	}
	logsFlag, _ := c.Doit.GetBool(c.NS, flagLogs)
	resultFlag, _ := c.Doit.GetBool(c.NS, flagResult)
	quietFlag, _ := c.Doit.GetBool(c.NS, flagQuiet)
	// There is also a 'last' flag, which is historical.  Since it's behavior is the
	// default, and the past convention was to ignore it if a single id was specified,
	// (rather than indicating an error), it is completely ignored here but accepted for
	// backward compatibility.  In the aio implementation (incorporated in nim, previously
	// incorporated here), the flag had to be set explicitly (rather than just implied) in
	// order to get a "banner" (additional informational line)  when requesting logs or
	// result only.  This seems pointless and we will always display the banner for a
	// single logs or result output unless --quiet is specified.
	skipFlag, _ := c.Doit.GetInt(c.NS, flagSkip) // 0 if not there
	functionFlag, _ := c.Doit.GetString(c.NS, flagFunction)
	sls := c.Serverless()
	if id == "" {
		// If there is no id, the convention is to retrieve the last activation, subject to possible
		// filtering or skipping
		options := whisk.ActivationListOptions{Limit: 1, Skip: skipFlag}
		if functionFlag != "" {
			options.Name = functionFlag
		}
		list, err := sls.ListActivations(options)
		if err != nil {
			return err
		}
		if len(list) == 0 {
			return fmt.Errorf("no activations were returned")
		}
		activation := list[0]
		id = activation.ActivationID
		if !quietFlag && (logsFlag || resultFlag) {
			makeBanner(c.Out, activation)
		}
	}
	if logsFlag {
		activation, err := sls.GetActivationLogs(id)
		if err != nil {
			return err
		}
		if len(activation.Logs) == 0 {
			return fmt.Errorf("no logs available")
		}
		printLogs(c.Out, true, activation)
	} else if resultFlag {
		response, err := sls.GetActivationResult(id)
		if err != nil {
			return err
		}
		if response.Result == nil {
			return fmt.Errorf("no result available")
		}
		printResult(c.Out, response.Result)
	} else {
		activation, err := sls.GetActivation(id)
		if err != nil {
			return err
		}
		printActivationRecord(c.Out, activation)
	}
	return nil
}

// makeBanner is a subroutine that prints a single "banner" line summarizing information about an
// activation.  This is done in conjunction with a request to print only logs or only the result, since,
// otherwise, it is difficult to know what activation is being talked about.
func makeBanner(writer io.Writer, activation whisk.Activation) {
	end := time.UnixMilli(activation.End).Format("01/02 03:04:05")
	init := text.NewStyled("=== ").Muted()
	body := fmt.Sprintf("%s %s %s %s:%s", activation.ActivationID, displayers.GetActivationStatus(activation.StatusCode),
		end, displayers.GetActivationFunctionName(activation), activation.Version)
	msg := text.NewStyled(body).Highlight()
	fmt.Fprintln(writer, init.String()+msg.String())
}

// printLog is a subroutine for printing just the logs of an activation
func printLogs(writer io.Writer, strip bool, activation whisk.Activation) {
	for _, log := range activation.Logs {
		if strip {
			log = stripLog(log)
		}
		fmt.Fprintln(writer, log)
	}
}

// dtsRegex is a regular expression that matches the prefix of some activation log entries.
// It is used by stripLog to remove that prefix
var dtsRegex = regexp.MustCompile(`(?U)\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:.*: `)

// stripLog strips the prefix from log entries
func stripLog(entry string) string {
	// `2019-10-11T19:08:57.298Z       stdout: login-success ::  { code: ...`
	// should become: `login-success ::  { code: ...`
	found := dtsRegex.FindString(entry)
	return entry[len(found):]
}

// printResult is a subroutine for printing just the result of an activation
func printResult(writer io.Writer, result *whisk.Result) {
	var msg string
	bytes, err := json.MarshalIndent(result, "", "  ")
	if err == nil {
		msg = string(bytes)
	} else {
		msg = "<unable to represent the result as JSON>"
	}
	fmt.Fprintln(writer, msg)
}

// printActivationRecord is a subroutine for printing the entire activation record
func printActivationRecord(writer io.Writer, activation whisk.Activation) {
	var msg string
	date := time.UnixMilli(activation.Start).Format("2006-01-02 03:04:05")
	toShow := ShownActivation{Activation: activation, Date: date}
	bytes, err := json.MarshalIndent(toShow, "", "  ")
	if err == nil {
		msg = string(bytes)
	} else {
		msg = "<unable to represent the activation as JSON>"
	}
	fmt.Fprintln(writer, msg)
}

// RunActivationsList supports the 'activations list' command
func RunActivationsList(c *CmdConfig) error {
	argCount := len(c.Args)

	if argCount > 1 {
		return doctl.NewTooManyArgsErr(c.NS)
	}
	sls := c.Serverless()

	var name string
	if argCount > 0 {
		name = c.Args[0]
	}

	countFlags, _ := c.Doit.GetBool(c.NS, flagCount)
	fullFlag, _ := c.Doit.GetBool(c.NS, flagFull)
	skipFlag, _ := c.Doit.GetInt(c.NS, flagSkip)
	sinceFlag, _ := c.Doit.GetInt(c.NS, flagSince)
	upToFlag, _ := c.Doit.GetInt(c.NS, flagUpto)
	limitFlag, _ := c.Doit.GetInt(c.NS, flagLimit)

	limit := limitFlag
	if limitFlag > 200 {
		limit = 200
	}

	if countFlags {
		options := whisk.ActivationCountOptions{Since: int64(sinceFlag), Upto: int64(upToFlag), Name: name}
		count, err := sls.GetActivationCount(options)
		if err != nil {
			return err
		}

		if name != "" {
			fmt.Fprintf(c.Out, "You have %d activations in this namespace for function %s \n", count.Activations, name)
		} else {
			fmt.Fprintf(c.Out, "You have %d activations in this namespace \n", count.Activations)
		}
		return nil
	}

	options := whisk.ActivationListOptions{Limit: limit, Skip: skipFlag, Since: int64(sinceFlag), Upto: int64(upToFlag), Docs: fullFlag, Name: name}

	actv, err := sls.ListActivations(options)
	if err != nil {
		return err
	}

	items := &displayers.Activation{Activations: actv}
	if fullFlag {
		return items.JSON(c.Out)
	}

	return c.Display(items)
}

// RunActivationsLogs supports the 'activations logs' command
func RunActivationsLogs(c *CmdConfig) error {
	argCount := len(c.Args)

	if argCount > 1 {
		return doctl.NewTooManyArgsErr(c.NS)
	}

	var activationId string
	if argCount == 1 {
		activationId = c.Args[0]
	}

	sls := c.Serverless()

	limitFlag, _ := c.Doit.GetInt(c.NS, flagLimit)
	stripFlag, _ := c.Doit.GetBool(c.NS, flagStrip)
	followFlag, _ := c.Doit.GetBool(c.NS, flagFollow)
	functionFlag, _ := c.Doit.GetString(c.NS, flagFunction)
	packageFlag, _ := c.Doit.GetString(c.NS, flagPackage)

	limit := limitFlag
	if limitFlag > 200 {
		limit = 200
	}

	if activationId != "" {
		actv, err := sls.GetActivationLogs(activationId)
		if err != nil {
			return err
		}
		printLogs(c.Out, stripFlag, actv)
		return nil

	} else if followFlag {
		sigChannel := make(chan os.Signal, 1)
		signal.Notify(sigChannel, os.Interrupt, syscall.SIGTERM)
		errChannel := make(chan error, 1)

		go pollActivations(errChannel, sls, c.Out, functionFlag, packageFlag)

		select {
		case <-sigChannel:
			fmt.Fprintf(c.Out, "\r")
			return nil
		case e := <-errChannel:
			return e
		}
	}

	listOptions := whisk.ActivationListOptions{Limit: limit, Name: functionFlag, Docs: true}
	actvs, err := sls.ListActivations(listOptions)

	if err != nil {
		return err
	}

	if packageFlag != "" {
		actvs = filterPackages(actvs, packageFlag)
	}

	for _, a := range reverseActivations(actvs) {
		makeBanner(c.Out, a)
		printLogs(c.Out, stripFlag, a)
		fmt.Fprintln(c.Out)
	}
	return nil
}

// Polls the ActivationList API at an interval and prints the results.
func pollActivations(ec chan error, sls do.ServerlessService, writer io.Writer, functionFlag string, packageFlag string) {
	ticker := time.NewTicker(time.Second * 5)
	tc := ticker.C
	var lastActivationTimestamp int64 = 0
	requestLimit := 1

	// There seems to be a race condition where functions invocation that start before lastActivationTimestamp
	// but is not completed by the time we make the list activation request will display twice. So prevent this issue
	// we keep track of the activation ids displayed, so we don't display the logs twice.
	printedActivations := map[string]int64{}

	for {
		select {
		case <-tc:
			options := whisk.ActivationListOptions{Limit: requestLimit, Since: lastActivationTimestamp, Docs: true, Name: functionFlag}
			actv, err := sls.ListActivations(options)

			if err != nil {
				ec <- err
				ticker.Stop()
				break
			}

			if packageFlag != "" {
				actv = filterPackages(actv, packageFlag)
			}

			if len(actv) > 0 {
				for _, activation := range reverseActivations(actv) {
					_, knownActivation := printedActivations[activation.ActivationID]

					if knownActivation {
						continue
					}

					printedActivations[activation.ActivationID] = activation.Start

					makeBanner(writer, activation)
					printLogs(writer, false, activation)
					fmt.Fprintln(writer)
				}

				lastItem := actv[len(actv)-1]
				lastActivationTimestamp = lastItem.Start + 100
				requestLimit = 0
			}
		}
	}
}

// Filters the activations to only return activations belonging to the package.
func filterPackages(activations []whisk.Activation, packageName string) []whisk.Activation {
	filteredActv := []whisk.Activation{}

	for _, activation := range activations {
		inPackage := displayers.GetActivationPackageName(activation) == packageName
		if inPackage {
			filteredActv = append(filteredActv, activation)
		}
	}
	return filteredActv
}

func reverseActivations(actv []whisk.Activation) []whisk.Activation {
	a := make([]whisk.Activation, len(actv))
	copy(a, actv)

	for i := len(a)/2 - 1; i >= 0; i-- {
		opp := len(a) - 1 - i
		a[i], a[opp] = a[opp], a[i]
	}

	return a
}

// RunActivationsResult supports the 'activations result' command
func RunActivationsResult(c *CmdConfig) error {
	argCount := len(c.Args)
	if argCount > 1 {
		return doctl.NewTooManyArgsErr(c.NS)
	}
	var id string
	if argCount > 0 {
		id = c.Args[0]
	}
	quietFlag, _ := c.Doit.GetBool(c.NS, flagQuiet)
	skipFlag, _ := c.Doit.GetInt(c.NS, flagSkip)   // 0 if not there
	limitFlag, _ := c.Doit.GetInt(c.NS, flagLimit) // 0 if not there
	functionFlag, _ := c.Doit.GetString(c.NS, flagFunction)
	limit := 1
	if limitFlag > 200 {
		limit = 200
	} else if limitFlag > 0 {
		limit = limitFlag
	}
	options := whisk.ActivationListOptions{Limit: limit, Skip: skipFlag}
	sls := c.Serverless()
	var activations []whisk.Activation
	if id == "" {
		if functionFlag != "" {
			options.Name = functionFlag
		}
		actv, err := sls.ListActivations(options)
		if err != nil {
			return err
		}
		activations = actv
	} else {
		activations = []whisk.Activation{
			{ActivationID: id},
		}
	}
	reversed := make([]whisk.Activation, len(activations))
	for i, activation := range activations {
		response, err := sls.GetActivationResult(activation.ActivationID)
		if err != nil {
			return err
		}
		activation.Result = response.Result
		reversed[len(activations)-i-1] = activation
	}
	for _, activation := range reversed {
		if !quietFlag && id == "" {
			makeBanner(c.Out, activation)
		}
		printResult(c.Out, activation.Result)
	}
	return nil
}

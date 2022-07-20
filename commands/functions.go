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
	"strings"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/commands/displayers"
	"github.com/digitalocean/doctl/do"
	"github.com/spf13/cobra"
)

// Functions generates the serverless 'functions' subtree for addition to the doctl command
func Functions() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:   "functions",
			Short: "Work with the functions in your namespace",
			Long: `The subcommands of ` + "`" + `doctl serverless functions` + "`" + ` operate on your functions namespace. 
You are able to inspect and list these functions to know what is deployed.  You can also invoke functions to test them.`,
			Aliases: []string{"fn"},
		},
	}

	get := CmdBuilder(cmd, RunFunctionsGet, "get <functionName>", "Retrieves the deployed copy of a function (code or metadata)",
		`Use `+"`"+`doctl serverless functions get`+"`"+` to obtain the code or metadata of a deployed function.
This allows you to inspect the deployed copy and ascertain whether it corresponds to what
is in your functions project in the local file system.`,
		Writer)
	AddBoolFlag(get, "url", "r", false, "get function url")
	AddBoolFlag(get, "code", "", false, "show function code (only works if code is not a zip file)")
	AddStringFlag(get, "save-env", "E", "", "save environment variables to FILE as key-value pairs")
	AddStringFlag(get, "save-env-json", "J", "", "save environment variables to FILE as JSON")
	AddBoolFlag(get, "save", "", false, "save function code to file corresponding to the function name")
	AddStringFlag(get, "save-as", "", "", "file to save function code to")

	invoke := CmdBuilder(cmd, RunFunctionsInvoke, "invoke <functionName>", "Invokes a function",
		`Use `+"`"+`doctl serverless functions invoke`+"`"+` to invoke a function in your functions namespace.
You can provide inputs and inspect outputs.`,
		Writer)
	AddBoolFlag(invoke, "web", "", false, "Invoke as a web function, show result as web page")
	AddStringSliceFlag(invoke, "param", "p", []string{}, "parameter values in KEY:VALUE format, list allowed")
	AddStringFlag(invoke, "param-file", "P", "", "FILE containing parameter values in JSON format")
	AddBoolFlag(invoke, "full", "f", false, "wait for full activation record")
	AddBoolFlag(invoke, "no-wait", "n", false, "fire and forget (asynchronous invoke, does not wait for the result)")

	list := CmdBuilder(cmd, RunFunctionsList, "list [<packageName>]", "Lists the functions in your functions namespace",
		`Use `+"`"+`doctl serverless functions list`+"`"+` to list the functions in your functions namespace.`,
		Writer, displayerType(&displayers.Functions{}))
	AddStringFlag(list, "limit", "l", "", "only return LIMIT number of functions (default 30, max 200)")
	AddStringFlag(list, "skip", "s", "", "exclude the first SKIP number of functions from the result")
	AddBoolFlag(list, "count", "", false, "show only the total number of functions")
	AddBoolFlag(list, "name-sort", "", false, "sort results by name")
	AddBoolFlag(list, "name", "n", false, "sort results by name")

	return cmd
}

// RunFunctionsGet supports the 'serverless functions get' command
func RunFunctionsGet(c *CmdConfig) error {
	err := ensureOneArg(c)
	if err != nil {
		return err
	}
	output, err := RunServerlessExec(actionGet, c, []string{flagURL, flagCode, flagSave}, []string{flagSaveEnv, flagSaveEnvJSON, flagSaveAs})
	if err != nil {
		return err
	}
	return c.PrintServerlessTextOutput(output)
}

// RunFunctionsInvoke supports the 'serverless functions invoke' command
func RunFunctionsInvoke(c *CmdConfig) error {
	err := ensureOneArg(c)
	if err != nil {
		return err
	}
	// Assemble args and flags except for "param"
	args := getFlatArgsArray(c, []string{flagWeb, flagFull, flagNoWait, flagResult}, []string{flagParamFile})
	// Add "param" with special handling if present
	args, err = appendParams(c, args)
	if err != nil {
		return err
	}
	output, err := ServerlessExec(c, actionInvoke, args...)
	if err != nil {
		return err
	}

	return c.PrintServerlessTextOutput(output)
}

// RunFunctionsList supports the 'serverless functions list' command
func RunFunctionsList(c *CmdConfig) error {
	argCount := len(c.Args)
	if argCount > 1 {
		return doctl.NewTooManyArgsErr(c.NS)
	}
	// Determine if '--count' is requested since we will use simple text output in that case.
	// Count is mutually exclusive with the global format flag.
	count, _ := c.Doit.GetBool(c.NS, flagCount)
	if count && c.Doit.IsSet("format") {
		return errors.New("the --count and --format flags are mutually exclusive")
	}
	// Add JSON flag so we can control output format
	if !count {
		c.Doit.Set(c.NS, flagJSON, true)
	}
	output, err := RunServerlessExec(actionList, c, []string{flagCount, flagNameSort, flagNameName, flagJSON}, []string{flagLimit, flagSkip})
	if err != nil {
		return err
	}
	if count {
		return c.PrintServerlessTextOutput(output)
	}
	// Reparse the output to use a more specific type, which can then be passed to the displayer
	rawOutput, err := json.Marshal(output.Entity)
	if err != nil {
		return err
	}
	var formatted []do.FunctionInfo
	err = json.Unmarshal(rawOutput, &formatted)
	if err != nil {
		return err
	}
	return c.Display(&displayers.Functions{Info: formatted})
}

// appendParams determines if there is a 'param' flag (value is a slice, elements
// of the slice should be in KEY:VALUE form), if so, transforms it into the form
// expected by 'nim' (each param is its own --param flag, KEY and VALUE are separate
// tokens).   The 'args' argument is the result of getFlatArgsArray and is appended
// to.
func appendParams(c *CmdConfig, args []string) ([]string, error) {
	params, err := c.Doit.GetStringSlice(c.NS, flagParam)
	if err != nil || len(params) == 0 {
		return args, nil // error here is not considered an error (and probably won't occur)
	}
	for _, param := range params {
		parts := strings.Split(param, ":")
		if len(parts) < 2 {
			return args, errors.New("values for --params must have KEY:VALUE form")
		}
		parts1 := strings.Join(parts[1:], ":")
		args = append(args, dashdashParam, parts[0], parts1)
	}
	return args, nil
}

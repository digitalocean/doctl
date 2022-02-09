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

func Functions() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:   "functions",
			Short: "Work with the functions of your sandbox",
			Long: `The subcommands of ` + "`" + `doctl sandbox functions` + "`" + ` operate on the deployed (cloud-resident) functions of your sandbox.
You are able to inspect and list these functions to know what is deployed.  You can also invoke functions to test them.`,
			Aliases: []string{"fn"},
		},
	}

	get := CmdBuilder(cmd, RunFunctionsGet, "get <functionName>", "Retrieves the deployed copy of a function (code or metadata)",
		`Use `+"`"+`doctl sandbox functions get`+"`"+` to obtain the code or metadata of a deployed function.
This allows you to inspect the deployed copy and ascertain whether it corresponds to what
is in your sandbox area in the local file system.`,
		Writer)
	AddBoolFlag(get, "url", "r", false, "get function url")
	AddBoolFlag(get, "code", "", false, "show function code (only works if code is not a zip file)")
	AddStringFlag(get, "save-env", "E", "", "save environment variables to FILE as key-value pairs")
	AddStringFlag(get, "save-env-json", "J", "", "save environment variables to FILE as JSON")
	AddBoolFlag(get, "save", "", false, "save function code to file corresponding to the function name")
	AddStringFlag(get, "save-as", "", "", "file to save function code to")

	invoke := CmdBuilder(cmd, RunFunctionsInvoke, "invoke <functionName>", "Invokes a function",
		`Use `+"`"+`doctl sandbox functions invoke`+"`"+` to invoke a function of your sandbox that has been deployed
to the cloud.  You can provide inputs and inspect outputs.`,
		Writer)
	AddBoolFlag(invoke, "web", "", false, "Invoke as a web function, show result as web page")
	AddStringFlag(invoke, "param", "p", "", "parameter values in KEY VALUE format")
	AddStringFlag(invoke, "param-file", "P", "", "FILE containing parameter values in JSON format")
	AddBoolFlag(invoke, "full", "f", false, "wait for full activation record")
	AddBoolFlag(invoke, "no-wait", "n", false, "fire and forget (asynchronous invoke, does not wait for the result)")

	list := CmdBuilder(cmd, RunFunctionsList, "list [<packageName>]", "Lists all the functions",
		`Use `+"`"+`doctl sandbox functions list`+"`"+` to list the functions of your sandbox that are deployed
to the cloud.`,
		Writer)
	AddStringFlag(list, "limit", "l", "", "only return LIMIT number of functions (default 30, max 200)")
	AddStringFlag(list, "skip", "s", "", "exclude the first SKIP number of functions from the result")
	AddBoolFlag(list, "count", "", false, "show only the total number of functions")
	AddBoolFlag(list, "name-sort", "", false, "sort results by name")
	AddBoolFlag(list, "name", "n", false, "sort results by name")

	return cmd
}

func RunFunctionsGet(c *CmdConfig) error {
	err := ensureOneArg(c)
	if err != nil {
		return err
	}
	output, err := RunSandboxExec("action/get", c, []string{"url", "code", "save"}, []string{"save-env", "save-env-json", "save-as"})
	if err != nil {
		return err
	}
	PrintSandboxTextOutput(output)
	return nil
}

func RunFunctionsInvoke(c *CmdConfig) error {
	err := ensureOneArg(c)
	if err != nil {
		return err
	}
	output, err := RunSandboxExec("action/invoke", c, []string{"web", "full", "no-wait", "result"}, []string{"param", "param-file"})
	if err != nil {
		return err
	}

	PrintSandboxTextOutput(output)
	return nil
}

func RunFunctionsList(c *CmdConfig) error {
	argCount := len(c.Args)
	if argCount > 1 {
		return doctl.NewTooManyArgsErr(c.NS)
	}
	output, err := RunSandboxExec("action/list", c, []string{"count", "name-sort", "name"}, []string{"limit", "skip"})
	if err != nil {
		return err
	}
	PrintSandboxTextOutput(output)
	return nil
}

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
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/apache/openwhisk-client-go/whisk"
	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/commands/charm/template"
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
			Aliases: []string{"function", "fn"},
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
		Writer, aliasOpt("ls"), displayerType(&displayers.Functions{}))
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
	urlFlag, _ := c.Doit.GetBool(c.NS, flagURL)
	codeFlag, _ := c.Doit.GetBool(c.NS, flagCode)
	saveFlag, _ := c.Doit.GetBool(c.NS, flagSave)
	saveAsFlag, _ := c.Doit.GetString(c.NS, flagSaveAs)
	saveEnvFlag, _ := c.Doit.GetString(c.NS, flagSaveEnv)
	saveEnvJSONFlag, _ := c.Doit.GetString(c.NS, flagSaveEnvJSON)
	fetchCode := codeFlag || saveFlag || saveAsFlag != ""

	sls := c.Serverless()
	action, parms, err := sls.GetFunction(c.Args[0], fetchCode)
	if err != nil {
		return err
	}

	if urlFlag {
		host, err := sls.GetConnectedAPIHost()
		if err != nil {
			return err
		}
		_, err = fmt.Fprintln(c.Out, computeURL(action, host))
		return err
	}

	if saveFlag || saveAsFlag != "" {
		return doSaveFunctionCode(action, saveFlag, saveAsFlag)
	}

	if saveEnvFlag != "" || saveEnvJSONFlag != "" {
		return doSaveFunctionEnvironment(saveEnvFlag, saveEnvJSONFlag, parms)
	}

	if codeFlag {
		if !*action.Exec.Binary {
			_, err = fmt.Fprintln(c.Out, *action.Exec.Code)
			return err
		}
		return errors.New("Binary code cannot be displayed on the console")
	}

	output := do.ServerlessOutput{Entity: action}
	return c.PrintServerlessTextOutput(output)
}

// doSaveFunctionCode performs the save operations for code
func doSaveFunctionCode(action whisk.Action, save bool, saveAs string) error {
	var extension string // used only when save and !saveAs
	var data []byte
	if *action.Exec.Binary {
		extension = ".zip"
		decoded, err := base64.StdEncoding.DecodeString(*action.Exec.Code)
		if err != nil {
			return err
		}
		data = decoded
	} else {
		extension = fileExtensionForKind(action.Exec.Kind) // find equivalent
		data = []byte(*action.Exec.Code)
	}
	if save && saveAs == "" {
		saveAs = action.Name + extension
	}
	if saveAs != "" {
		err := os.WriteFile(saveAs, data, 0666)
		if err != nil {
			return err
		}
	}

	return nil
}

// doSaveFunctionEnvironment saves the environment variables for a function to file,
// either as key-value pairs or JSON.  Could do both if both are specified.
func doSaveFunctionEnvironment(saveEnv string, saveEnvJSON string, parms []do.FunctionParameter) error {
	keyVals := []string{}
	envMap := map[string]string{}
	for _, parm := range parms {
		if parm.Init {
			keyVal := parm.Key + "=" + parm.Value
			keyVals = append(keyVals, keyVal)
			envMap[parm.Key] = parm.Value
		}
	}

	if saveEnv != "" {
		data := []byte(strings.Join(keyVals, "\n"))
		err := os.WriteFile(saveEnv, data, 0666)
		if err != nil {
			return err
		}
	}

	if saveEnvJSON != "" {
		data, err := json.MarshalIndent(&envMap, "", "  ")
		if err != nil {
			return err
		}
		err = os.WriteFile(saveEnvJSON, data, 0666)
		if err != nil {
			return err
		}
	}

	return nil
}

// fileExtensionforKind finds the right file extension for a given runtime 'kind'.
// This code will require modification when the repertoire of runtimes is extended.
func fileExtensionForKind(kind string) string {
	lang := strings.Split(kind, ":")[0]
	switch strings.ToLower(lang) {
	case "go":
		return ".go"
	case "nodejs":
		return ".js"
	case "php":
		return ".php"
	case "python":
		return ".py"
	}
	return ""
}

// computeURL determines the URL string based on the action get output.
// Based on code in aio-cli-plugin-runtime, src/commands/runtime/action/get.js
func computeURL(action whisk.Action, host string) string {
	nameParts := strings.Split(action.Namespace, "/")
	namespace := nameParts[0]
	var packageName string
	if len(nameParts) > 1 {
		packageName = nameParts[1]
	}
	if action.WebAction() {
		if packageName == "" {
			packageName = "default"
		}
		return fmt.Sprintf("%s/api/v1/web/%s/%s/%s", host, namespace, packageName, action.Name)
	}
	if packageName != "" {
		packageName += "/"
	}
	return fmt.Sprintf("%s/api/v1/namespaces/%s/actions/%s%s", host, namespace, packageName, action.Name)
}

// RunFunctionsInvoke supports the 'serverless functions invoke' command
func RunFunctionsInvoke(c *CmdConfig) error {
	err := ensureOneArg(c)
	if err != nil {
		return err
	}
	paramFile, _ := c.Doit.GetString(c.NS, flagParamFile)
	paramFlags, _ := c.Doit.GetStringSlice(c.NS, flagParam)
	params, err := consolidateParams(paramFile, paramFlags)
	if err != nil {
		return err
	}
	web, _ := c.Doit.GetBool(c.NS, flagWeb)
	if web {
		var mapParams map[string]interface{} = nil
		if params != nil {
			p, ok := params.(map[string]interface{})
			if !ok {
				return fmt.Errorf("cannot invoke via web: parameters do not form a dictionary")
			}
			mapParams = p
		}
		return c.Serverless().InvokeFunctionViaWeb(c.Args[0], mapParams)
	}
	full, _ := c.Doit.GetBool(c.NS, flagFull)
	noWait, _ := c.Doit.GetBool(c.NS, flagNoWait)
	blocking := !noWait
	result := blocking && !full
	response, err := c.Serverless().InvokeFunction(c.Args[0], params, blocking, result)

	if err != nil {
		if response != nil {
			activationResponse := response.(map[string]interface{})
			template.Print(`Request accepted, but processing not completed yet. {{nl}}All functions invocation >= 30s will get demoted to an asynchronous invocation. Use {{highlight "--no-wait"}} flag to immediately return the activation id. {{nl}}
Use this command to view the results.
{{bold "doctl sls activations result" }} {{bold .}} {{nl 2}}`, activationResponse["activationId"])
			return nil
		}
		return err
	}

	output := do.ServerlessOutput{Entity: response}
	return c.PrintServerlessTextOutput(output)
}

// RunFunctionsList supports the 'serverless functions list' command
func RunFunctionsList(c *CmdConfig) error {
	argCount := len(c.Args)
	if argCount > 1 {
		return doctl.NewTooManyArgsErr(c.NS)
	}
	var pkg string
	if argCount == 1 {
		pkg = c.Args[0]
	}
	// Determine if '--count' is requested since we will use simple text output in that case.
	// Count is mutually exclusive with the global format flag.
	count, _ := c.Doit.GetBool(c.NS, flagCount)
	if count && c.Doit.IsSet("format") {
		return errors.New("the --count and --format flags are mutually exclusive")
	}
	// Retrieve other flags
	skip, _ := c.Doit.GetInt(c.NS, flagSkip)
	limit, _ := c.Doit.GetInt(c.NS, flagLimit)
	nameSort, _ := c.Doit.GetBool(c.NS, flagNameSort)
	nameName, _ := c.Doit.GetBool(c.NS, flagNameName)
	// Get information from backend
	list, err := c.Serverless().ListFunctions(pkg, skip, limit)
	if err != nil {
		return err
	}
	if count {
		plural := "s"
		are := "are"
		if len(list) == 1 {
			plural = ""
			are = "is"
		}
		fmt.Fprintf(c.Out, "There %s %d function%s in this namespace.\n", are, len(list), plural)
		return nil
	}
	if nameSort || nameName {
		sortFunctionList(list)
	}
	return c.Display(&displayers.Functions{Info: list})
}

// sortFunctionList performs a sort of a function list (by name)
func sortFunctionList(list []whisk.Action) {
	isLess := func(i, j int) bool {
		return list[i].Name < list[j].Name
	}
	sort.Slice(list, isLess)
}

// consolidateParams accepts parameters from a file, the command line, or both, and consolidates all
// such parameters into a simple dictionary.
func consolidateParams(paramFile string, params []string) (interface{}, error) {
	consolidated := map[string]interface{}{}
	if len(paramFile) > 0 {
		contents, err := os.ReadFile(paramFile)
		if err != nil {
			return nil, err
		}
		err = json.Unmarshal(contents, &consolidated)
		if err != nil {
			return nil, err
		}
	}
	for _, param := range params {
		parts := strings.Split(param, ":")
		if len(parts) < 2 {
			return nil, fmt.Errorf("values for --params must have KEY:VALUE form")
		}
		parts1 := strings.Join(parts[1:], ":")
		consolidated[parts[0]] = parts1
	}
	if len(consolidated) > 0 {
		return consolidated, nil
	}
	return nil, nil
}

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
			Use:     "functions",
			Short:   "Work with the functions in your namespace",
			Long:    `The subcommands of ` + "`" + `doctl serverless functions` + "`" + ` manage your functions namespace, such as listing, reviewing, and invoking your functions.`,
			Aliases: []string{"function", "fn"},
		},
	}

	get := CmdBuilder(cmd, RunFunctionsGet, "get <functionName>", "Retrieve the metadata or code of a deployed function",
		`Retrieves the code or metadata of a deployed function.`,
		Writer)
	AddBoolFlag(get, "url", "r", false, "Retrieves function URL")
	AddBoolFlag(get, "code", "", false, "Retrieves the functions code. This does not work if the function is saved as a zip file.")
	AddStringFlag(get, "save-env", "E", "", "Saves the function's environment variables to a local file as key-value pairs")
	AddStringFlag(get, "save-env-json", "J", "", "Saves the function's environment variables to a local file as JSON")
	AddBoolFlag(get, "save", "", false, "Saves the function's code to a local file")
	AddStringFlag(get, "save-as", "", "", "Saves the file as the specified name")
	get.Example = `The following example retrieves the code for a function named "example/helloWorld" and saves it to a file named ` + "`" + `local-helloWorld.py` + "`" + `: doctl serverless functions get example/helloWorld --code --save-as local-helloWorld.py`

	invoke := CmdBuilder(cmd, RunFunctionsInvoke, "invoke <functionName>", "Invokes a function",
		`Invokes a function in your functions namespace.
You can provide inputs and inspect outputs.`,
		Writer)
	AddBoolFlag(invoke, "web", "", false, "Invokes the function as a web function and displays the result in your browser")
	AddStringSliceFlag(invoke, "param", "p", []string{}, "Key-value pairs of input parameters. For example, if your function takes two parameters called `name` and `place`, you would provide them as `name:John,place:NY`.")
	AddStringFlag(invoke, "param-file", "P", "", "A path to a file containing parameter values in JSON format, such as `path/to/file.json`.")
	AddBoolFlag(invoke, "full", "f", false, "Waits for the function to complete and then outputs the function's response along with its complete activation record. The record contains log information, duration time, and other information about the function's execution.")
	AddBoolFlag(invoke, "no-wait", "n", false, "Asynchronously invokes the function and does not wait for the result to be returned. An activation ID is returned in the response, instead.")
	invoke.Example = `The following example invokes a function named "example/helloWorld" with the parameters ` + "`" + `name:John,place:NY` + "`" + `: doctl serverless functions invoke example/helloWorld --param name:John,place:NY`

	list := CmdBuilder(cmd, RunFunctionsList, "list [<packageName>]", "Lists the functions in your functions namespace",
		`Lists the functions in your functions namespace.`,
		Writer, aliasOpt("ls"), displayerType(&displayers.Functions{}))
	AddStringFlag(list, "limit", "l", "", "Returns the specified number of functions in the result, starting with the most recently updated function.")
	AddStringFlag(list, "skip", "s", "", "Excludes the specified number of functions from the result, starting with the most recently updated function. For example, if you specify `2`, the most recently updated function and the function updated before that are excluded from the result.")
	AddBoolFlag(list, "count", "", false, "Returns only the total number of functions in the namespace")
	AddBoolFlag(list, "name-sort", "", false, "Sorts results by name in alphabetical order")
	AddBoolFlag(list, "name", "n", false, "Sorts results by name in alphabetical order")
	list.Example = `The following example lists the three most recently updated functions in the ` + "`" + `example` + "`" + ` package: doctl serverless functions list example --limit 3`

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
		var mapParams map[string]any = nil
		if params != nil {
			p, ok := params.(map[string]any)
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
			activationResponse := response.(map[string]any)
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
func consolidateParams(paramFile string, params []string) (any, error) {
	consolidated := map[string]any{}
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

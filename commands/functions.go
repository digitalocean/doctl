package commands

import (
	"github.com/digitalocean/doctl"
	"github.com/spf13/cobra"
)

func Functions() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:   "functions",
			Short: "This is the functions subtree",
			Long:  `This is more information about the functions subtree`,
		},
	}

	Create := cmdBuilderWithInit(cmd, RunFunctionsCreate, "create <actionName> [<actionPath>]", "Creates an Action",
		`More information about 'functions create'`,
		Writer, false)
	AddStringFlag(Create, "cert", "", "", "client cert")
	AddStringFlag(Create, "key", "", "", "client key")
	AddStringFlag(Create, "apiversion", "", "", "whisk API version")
	AddStringFlag(Create, "apihost", "", "", "whisk API host")
	AddStringFlag(Create, "auth", "", "", "whisk auth")
	AddBoolFlag(Create, "insecure", "i", false, "bypass certificate check")
	AddStringFlag(Create, "debug", "", "", "Debug level output")
	AddStringFlag(Create, "useragent", "", "aio-cli-plugin-runtime@4.0.0", "Use custom user-agent string")
	Create.Flags().MarkHidden("useragent")
	AddStringFlag(Create, "param", "p", "", "parameter values in KEY VALUE format")
	AddStringFlag(Create, "env", "e", "", "environment values in KEY VALUE format")
	AddStringFlag(Create, "web", "", "", "treat ACTION as a web action or as a raw HTTP web action")
	AddStringFlag(Create, "web-secure", "", "", "secure the web action (valid values are true, false, or any string)")
	AddStringFlag(Create, "param-file", "P", "", "FILE containing parameter values in JSON format")
	AddStringFlag(Create, "env-file", "E", "", "FILE containing environment variables in JSON format")
	AddStringFlag(Create, "timeout", "", "", "Timeout LIMIT in milliseconds after which the Action is terminated")
	AddStringFlag(Create, "memory", "m", "", "Maximum memory LIMIT in MB for the Action")
	AddStringFlag(Create, "logsize", "l", "", "Maximum log size LIMIT in KB for the Action")
	AddStringFlag(Create, "kind", "", "", "the KIND of the action runtime (example: swift:default, nodejs:default)")
	AddStringFlag(Create, "annotation", "a", "", "annotation values in KEY VALUE format")
	AddStringFlag(Create, "annotation-file", "A", "", "FILE containing annotation values in JSON format")
	AddStringFlag(Create, "sequence", "", "", "treat ACTION as comma separated sequence of actions to invoke")
	AddStringFlag(Create, "docker", "", "", "use provided Docker image (a path on DockerHub) to run the action")
	AddBoolFlag(Create, "native", "", false, "use default skeleton runtime where code artifact provides actual executable for the action")
	AddStringFlag(Create, "main", "", "", "the name of the action entry point (function or fully-qualified method name when applicable)")
	AddBoolFlag(Create, "binary", "", false, "treat code artifact as binary")
	AddBoolFlag(Create, "json", "", false, "output raw json")

	Delete := cmdBuilderWithInit(cmd, RunFunctionsDelete, "delete <actionName>", "Deletes an Action",
		`More information about 'functions delete'`,
		Writer, false)
	AddBoolFlag(Delete, "force", "f", false, "Just do it, omitting confirmatory prompt")
	AddStringFlag(Delete, "cert", "", "", "client cert")
	AddStringFlag(Delete, "key", "", "", "client key")
	AddStringFlag(Delete, "apiversion", "", "", "whisk API version")
	AddStringFlag(Delete, "apihost", "", "", "whisk API host")
	AddStringFlag(Delete, "auth", "", "", "whisk auth")
	AddBoolFlag(Delete, "insecure", "i", false, "bypass certificate check")
	AddStringFlag(Delete, "debug", "", "", "Debug level output")
	AddStringFlag(Delete, "useragent", "", "aio-cli-plugin-runtime@4.0.0", "Use custom user-agent string")
	Delete.Flags().MarkHidden("useragent")
	AddBoolFlag(Delete, "json", "", false, "output raw json")

	Get := cmdBuilderWithInit(cmd, RunFunctionsGet, "get <actionName>", "Retrieves an Action",
		`More information about 'functions get'`,
		Writer, false)
	AddStringFlag(Get, "cert", "", "", "client cert")
	AddStringFlag(Get, "key", "", "", "client key")
	AddStringFlag(Get, "apiversion", "", "", "whisk API version")
	AddStringFlag(Get, "apihost", "", "", "whisk API host")
	AddStringFlag(Get, "auth", "", "", "whisk auth")
	AddBoolFlag(Get, "insecure", "i", false, "bypass certificate check")
	AddStringFlag(Get, "debug", "", "", "Debug level output")
	AddStringFlag(Get, "useragent", "", "aio-cli-plugin-runtime@4.0.0", "Use custom user-agent string")
	Get.Flags().MarkHidden("useragent")
	AddBoolFlag(Get, "url", "r", false, "get action url")
	AddBoolFlag(Get, "code", "", false, "show action code (only works if code is not a zip file)")
	AddStringFlag(Get, "save-env", "E", "", "save environment variables to FILE as key-value pairs")
	AddStringFlag(Get, "save-env-json", "J", "", "save environment variables to FILE as JSON")
	AddBoolFlag(Get, "save", "", false, "save action code to file corresponding with action name")
	AddStringFlag(Get, "save-as", "", "", "file to save action code to")

	Invoke := cmdBuilderWithInit(cmd, RunFunctionsInvoke, "invoke <actionName>", "Invokes an Action",
		`More information about 'functions invoke'`,
		Writer, false)
	AddBoolFlag(Invoke, "web", "", false, "Invoke as a web action, show result as web page")
	AddStringFlag(Invoke, "cert", "", "", "client cert")
	AddStringFlag(Invoke, "key", "", "", "client key")
	AddStringFlag(Invoke, "apiversion", "", "", "whisk API version")
	AddStringFlag(Invoke, "apihost", "", "", "whisk API host")
	AddStringFlag(Invoke, "auth", "", "", "whisk auth")
	AddBoolFlag(Invoke, "insecure", "i", false, "bypass certificate check")
	AddStringFlag(Invoke, "debug", "", "", "Debug level output")
	AddStringFlag(Invoke, "useragent", "", "aio-cli-plugin-runtime@4.0.0", "Use custom user-agent string")
	Invoke.Flags().MarkHidden("useragent")
	AddStringFlag(Invoke, "param", "p", "", "parameter values in KEY VALUE format")
	AddStringFlag(Invoke, "param-file", "P", "", "FILE containing parameter values in JSON format")
	AddBoolFlag(Invoke, "full", "f", false, "wait for full activation record")
	AddBoolFlag(Invoke, "no-wait", "n", false, "fire and forget (asynchronous invoke, does not wait for the result)")
	AddBoolFlag(Invoke, "result", "r", false, "invoke action and wait for the result (default)")
	Invoke.Flags().MarkHidden("result")

	List := cmdBuilderWithInit(cmd, RunFunctionsList, "list [<packageName>]", "Lists all the Actions",
		`More information about 'functions list'`,
		Writer, false)
	AddStringFlag(List, "cert", "", "", "client cert")
	AddStringFlag(List, "key", "", "", "client key")
	AddStringFlag(List, "apiversion", "", "", "whisk API version")
	AddStringFlag(List, "apihost", "", "", "whisk API host")
	AddStringFlag(List, "auth", "", "", "whisk auth")
	AddBoolFlag(List, "insecure", "i", false, "bypass certificate check")
	AddStringFlag(List, "debug", "", "", "Debug level output")
	AddStringFlag(List, "useragent", "", "aio-cli-plugin-runtime@4.0.0", "Use custom user-agent string")
	List.Flags().MarkHidden("useragent")
	AddStringFlag(List, "limit", "l", "", "only return LIMIT number of actions")
	AddStringFlag(List, "skip", "s", "", "exclude the first SKIP number of actions from the result")
	AddBoolFlag(List, "count", "", false, "show only the total number of actions")
	AddBoolFlag(List, "json", "", false, "output raw json")
	AddBoolFlag(List, "name-sort", "", false, "sort results by name")
	AddBoolFlag(List, "name", "n", false, "sort results by name")

	Update := cmdBuilderWithInit(cmd, RunFunctionsUpdate, "update <actionName> [<actionPath>]", "Updates an Action",
		`More information about 'functions update'`,
		Writer, false)
	AddStringFlag(Update, "cert", "", "", "client cert")
	AddStringFlag(Update, "key", "", "", "client key")
	AddStringFlag(Update, "apiversion", "", "", "whisk API version")
	AddStringFlag(Update, "apihost", "", "", "whisk API host")
	AddStringFlag(Update, "auth", "", "", "whisk auth")
	AddBoolFlag(Update, "insecure", "i", false, "bypass certificate check")
	AddStringFlag(Update, "debug", "", "", "Debug level output")
	AddStringFlag(Update, "useragent", "", "aio-cli-plugin-runtime@4.0.0", "Use custom user-agent string")
	Update.Flags().MarkHidden("useragent")
	AddStringFlag(Update, "param", "p", "", "parameter values in KEY VALUE format")
	AddStringFlag(Update, "env", "e", "", "environment values in KEY VALUE format")
	AddStringFlag(Update, "web", "", "", "treat ACTION as a web action or as a raw HTTP web action")
	AddStringFlag(Update, "web-secure", "", "", "secure the web action (valid values are true, false, or any string)")
	AddStringFlag(Update, "param-file", "P", "", "FILE containing parameter values in JSON format")
	AddStringFlag(Update, "env-file", "E", "", "FILE containing environment variables in JSON format")
	AddStringFlag(Update, "timeout", "", "", "Timeout LIMIT in milliseconds after which the Action is terminated")
	AddStringFlag(Update, "memory", "m", "", "Maximum memory LIMIT in MB for the Action")
	AddStringFlag(Update, "logsize", "l", "", "Maximum log size LIMIT in KB for the Action")
	AddStringFlag(Update, "kind", "", "", "the KIND of the action runtime (example: swift:default, nodejs:default)")
	AddStringFlag(Update, "annotation", "a", "", "annotation values in KEY VALUE format")
	AddStringFlag(Update, "annotation-file", "A", "", "FILE containing annotation values in JSON format")
	AddStringFlag(Update, "sequence", "", "", "treat ACTION as comma separated sequence of actions to invoke")
	AddStringFlag(Update, "docker", "", "", "use provided Docker image (a path on DockerHub) to run the action")
	AddBoolFlag(Update, "native", "", false, "use default skeleton runtime where code artifact provides actual executable for the action")
	AddStringFlag(Update, "main", "", "", "the name of the action entry point (function or fully-qualified method name when applicable)")
	AddBoolFlag(Update, "binary", "", false, "treat code artifact as binary")
	AddBoolFlag(Update, "json", "", false, "output raw json")

	return cmd
}
func RunFunctionsCreate(c *CmdConfig) error {
	argCount := len(c.Args)
	if argCount < 1 {
		return doctl.NewMissingArgsErr(c.NS)
	}
	if argCount > 2 {
		return doctl.NewTooManyArgsErr(c.NS)
	}
	output, err := RunSandboxExec("action/create", c, []string{"insecure", "native", "binary", "json"}, []string{"cert", "key", "apiversion", "apihost", "auth", "debug", "useragent", "param", "env", "web", "web-secure", "param-file", "env-file", "timeout", "memory", "logsize", "kind", "annotation", "annotation-file", "sequence", "docker", "main"})
	if err != nil {
		return err
	}
	PrintSandboxTextOutput(output)
	return nil
}

func RunFunctionsDelete(c *CmdConfig) error {
	err := ensureOneArg(c)
	if err != nil {
		return err
	}
	output, err := RunSandboxExec("action/delete", c, []string{"force", "insecure", "json"}, []string{"cert", "key", "apiversion", "apihost", "auth", "debug", "useragent"})
	if err != nil {
		return err
	}
	PrintSandboxTextOutput(output)
	return nil
}

func RunFunctionsGet(c *CmdConfig) error {
	err := ensureOneArg(c)
	if err != nil {
		return err
	}
	output, err := RunSandboxExec("action/get", c, []string{"insecure", "url", "code", "save"}, []string{"cert", "key", "apiversion", "apihost", "auth", "debug", "useragent", "save-env", "save-env-json", "save-as"})
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
	output, err := RunSandboxExec("action/invoke", c, []string{"web", "insecure", "full", "no-wait", "result"}, []string{"cert", "key", "apiversion", "apihost", "auth", "debug", "useragent", "param", "param-file"})
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
	output, err := RunSandboxExec("action/list", c, []string{"insecure", "count", "json", "name-sort", "name"}, []string{"cert", "key", "apiversion", "apihost", "auth", "debug", "useragent", "limit", "skip"})
	if err != nil {
		return err
	}
	PrintSandboxTextOutput(output)
	return nil
}

func RunFunctionsUpdate(c *CmdConfig) error {
	argCount := len(c.Args)
	if argCount < 1 {
		return doctl.NewMissingArgsErr(c.NS)
	}
	if argCount > 2 {
		return doctl.NewTooManyArgsErr(c.NS)
	}
	output, err := RunSandboxExec("action/update", c, []string{"insecure", "native", "binary", "json"}, []string{"cert", "key", "apiversion", "apihost", "auth", "debug", "useragent", "param", "env", "web", "web-secure", "param-file", "env-file", "timeout", "memory", "logsize", "kind", "annotation", "annotation-file", "sequence", "docker", "main"})
	if err != nil {
		return err
	}
	PrintSandboxTextOutput(output)
	return nil
}

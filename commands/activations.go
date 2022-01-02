package commands

import (
	"github.com/digitalocean/doctl"
	"github.com/spf13/cobra"
)

func Activations() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:   "activations",
			Short: "This is the activations subtree",
			Long:  `This is more information about the activations subtree`,
		},
	}

	get := cmdBuilderWithInit(cmd, RunActivationsGet, "get [<activationId>]", "Retrieves an Activation",
		`More information about 'activations get'`,
		Writer, false)
	AddStringFlag(get, "cert", "", "", "client cert")
	AddStringFlag(get, "key", "", "", "client key")
	AddStringFlag(get, "apiversion", "", "", "whisk API version")
	AddStringFlag(get, "apihost", "", "", "whisk API host")
	AddStringFlag(get, "auth", "", "", "whisk auth")
	AddBoolFlag(get, "insecure", "i", false, "bypass certificate check")
	AddStringFlag(get, "debug", "", "", "Debug level output")
	AddStringFlag(get, "useragent", "", "aio-cli-plugin-runtime@4.0.0", "Use custom user-agent string")
	get.Flags().MarkHidden("useragent")
	AddBoolFlag(get, "last", "l", false, "Fetch the most recent activation (default)")
	AddIntFlag(get, "skip", "s", 0, "SKIP number of activations")
	AddBoolFlag(get, "logs", "g", false, "Emit only the logs, stripped of time stamps and stream identifier")
	AddBoolFlag(get, "result", "r", false, "Emit only the result")
	AddStringFlag(get, "action", "a", "", "Fetch logs for a specific action")
	AddBoolFlag(get, "quiet", "q", false, "Suppress last activation information header")

	list := cmdBuilderWithInit(cmd, RunActivationsList, "list [<activation_name>]", "Lists all the Activations",
		`More information about 'activations list'`,
		Writer, false)
	AddStringFlag(list, "cert", "", "", "client cert")
	AddStringFlag(list, "key", "", "", "client key")
	AddStringFlag(list, "apiversion", "", "", "whisk API version")
	AddStringFlag(list, "apihost", "", "", "whisk API host")
	AddStringFlag(list, "auth", "", "", "whisk auth")
	AddBoolFlag(list, "insecure", "i", false, "bypass certificate check")
	AddStringFlag(list, "debug", "", "", "Debug level output")
	AddStringFlag(list, "useragent", "", "aio-cli-plugin-runtime@4.0.0", "Use custom user-agent string")
	list.Flags().MarkHidden("useragent")
	AddStringFlag(list, "limit", "l", "", "only return LIMIT number of activations")
	AddStringFlag(list, "skip", "s", "", "exclude the first SKIP number of activations from the result")
	AddStringFlag(list, "since", "", "", "return activations with timestamps later than SINCE; measured in milliseconds since Th, 01, Jan 1970")
	AddStringFlag(list, "upto", "", "", "return activations with timestamps earlier than UPTO; measured in milliseconds since Th, 01, Jan 1970")
	AddBoolFlag(list, "count", "", false, "show only the total number of activations")
	AddBoolFlag(list, "json", "", false, "output raw json")
	AddBoolFlag(list, "full", "f", false, "include full activation description")

	logs := cmdBuilderWithInit(cmd, RunActivationsLogs, "logs [<activationId>]", "Retrieves the Logs for an Activation",
		`More information about 'activations logs'`,
		Writer, false)
	AddStringFlag(logs, "cert", "", "", "client cert")
	AddStringFlag(logs, "key", "", "", "client key")
	AddStringFlag(logs, "apiversion", "", "", "whisk API version")
	AddStringFlag(logs, "apihost", "", "", "whisk API host")
	AddStringFlag(logs, "auth", "", "", "whisk auth")
	AddBoolFlag(logs, "insecure", "i", false, "bypass certificate check")
	AddStringFlag(logs, "debug", "", "", "Debug level output")
	AddStringFlag(logs, "useragent", "", "aio-cli-plugin-runtime@4.0.0", "Use custom user-agent string")
	logs.Flags().MarkHidden("useragent")
	AddStringFlag(logs, "action", "a", "", "Fetch logs for a specific action")
	AddBoolFlag(logs, "manifest", "m", false, "Fetch logs for all actions in the manifest")
	AddStringFlag(logs, "package", "p", "", "Fetch logs for a specific package in the manifest")
	AddBoolFlag(logs, "deployed", "d", false, "Fetch logs for all actions deployed under a specific package")
	AddBoolFlag(logs, "last", "l", false, "Fetch the most recent activation logs (default)")
	AddIntFlag(logs, "limit", "n", 1, "Fetch the last LIMIT activation logs (up to 200)")
	AddBoolFlag(logs, "strip", "r", false, "strip timestamp information and output first line only")
	AddBoolFlag(logs, "tail", "", false, "Fetch logs continuously")
	AddBoolFlag(logs, "watch", "w", false, "Fetch logs continuously")
	AddBoolFlag(logs, "poll", "", false, "Fetch logs continuously")

	result := cmdBuilderWithInit(cmd, RunActivationsResult, "result [<activationId>]", "Retrieves the Results for an Activation",
		`More information about 'activations result'`,
		Writer, false)
	AddStringFlag(result, "cert", "", "", "client cert")
	AddStringFlag(result, "key", "", "", "client key")
	AddStringFlag(result, "apiversion", "", "", "whisk API version")
	AddStringFlag(result, "apihost", "", "", "whisk API host")
	AddStringFlag(result, "auth", "", "", "whisk auth")
	AddBoolFlag(result, "insecure", "i", false, "bypass certificate check")
	AddStringFlag(result, "debug", "", "", "Debug level output")
	AddStringFlag(result, "useragent", "", "aio-cli-plugin-runtime@4.0.0", "Use custom user-agent string")
	result.Flags().MarkHidden("useragent")
	AddBoolFlag(result, "last", "l", false, "Fetch the most recent activation result (default)")
	AddIntFlag(result, "limit", "n", 1, "Fetch the last LIMIT activation results (up to 200)")
	AddIntFlag(result, "skip", "s", 0, "SKIP number of activations")
	AddStringFlag(result, "action", "a", "", "Fetch results for a specific action")
	AddBoolFlag(result, "quiet", "q", false, "Suppress last activation information header")

	return cmd
}
func RunActivationsGet(c *CmdConfig) error {
	argCount := len(c.Args)
	if argCount > 1 {
		return doctl.NewTooManyArgsErr(c.NS)
	}
	output, err := RunSandboxExec("activation/get", c, []string{"insecure", "last", "logs", "result", "quiet"}, []string{"cert", "key", "apiversion", "apihost", "auth", "debug", "useragent", "skip", "action"})
	if err != nil {
		return err
	}
	PrintSandboxTextOutput(output)
	return nil
}

func RunActivationsList(c *CmdConfig) error {
	argCount := len(c.Args)
	if argCount > 1 {
		return doctl.NewTooManyArgsErr(c.NS)
	}
	output, err := RunSandboxExec("activation/list", c, []string{"insecure", "count", "json", "full"}, []string{"cert", "key", "apiversion", "apihost", "auth", "debug", "useragent", "limit", "skip", "since", "upto"})
	if err != nil {
		return err
	}
	PrintSandboxTextOutput(output)
	return nil
}

func RunActivationsLogs(c *CmdConfig) error {
	argCount := len(c.Args)
	if argCount > 1 {
		return doctl.NewTooManyArgsErr(c.NS)
	}
	output, err := RunSandboxExec("activation/logs", c, []string{"insecure", "manifest", "deployed", "last", "strip", "tail", "watch", "poll"}, []string{"cert", "key", "apiversion", "apihost", "auth", "debug", "useragent", "action", "package", "limit"})
	if err != nil {
		return err
	}
	PrintSandboxTextOutput(output)
	return nil
}

func RunActivationsResult(c *CmdConfig) error {
	argCount := len(c.Args)
	if argCount > 1 {
		return doctl.NewTooManyArgsErr(c.NS)
	}
	output, err := RunSandboxExec("activation/result", c, []string{"insecure", "last", "quiet"}, []string{"cert", "key", "apiversion", "apihost", "auth", "debug", "useragent", "limit", "skip", "action"})
	if err != nil {
		return err
	}
	PrintSandboxTextOutput(output)
	return nil
}

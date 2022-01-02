package commands

import (
	"github.com/digitalocean/doctl"
	"github.com/spf13/cobra"
)

func Packages() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:   "packages",
			Short: "This is the packages subtree",
			Long:  `This is more information about the packages subtree`,
		},
	}

	bind := cmdBuilderWithInit(cmd, RunPackagesBind, "bind <packageName> <bindPackageName>", "Bind parameters to a package",
		`More information about 'packages bind'`,
		Writer, false)
	AddStringFlag(bind, "cert", "", "", "client cert")
	AddStringFlag(bind, "key", "", "", "client key")
	AddStringFlag(bind, "apiversion", "", "", "whisk API version")
	AddStringFlag(bind, "apihost", "", "", "whisk API host")
	AddStringFlag(bind, "auth", "", "", "whisk auth")
	AddBoolFlag(bind, "insecure", "i", false, "bypass certificate check")
	AddStringFlag(bind, "debug", "", "", "Debug level output")
	AddStringFlag(bind, "useragent", "", "aio-cli-plugin-runtime@4.0.0", "Use custom user-agent string")
	bind.Flags().MarkHidden("useragent")
	AddStringFlag(bind, "param", "p", "", "parameters in key value pairs to be passed to the package")
	AddStringFlag(bind, "param-file", "P", "", "parameter to be passed to the package for json file")
	AddStringFlag(bind, "annotation", "a", "", "annotation values in KEY VALUE format")
	AddStringFlag(bind, "annotation-file", "A", "", "FILE containing annotation values in JSON format")
	AddBoolFlag(bind, "json", "", false, "output raw json")

	create := cmdBuilderWithInit(cmd, RunPackagesCreate, "create <packageName>", "Creates a Package",
		`More information about 'packages create'`,
		Writer, false)
	AddStringFlag(create, "cert", "", "", "client cert")
	AddStringFlag(create, "key", "", "", "client key")
	AddStringFlag(create, "apiversion", "", "", "whisk API version")
	AddStringFlag(create, "apihost", "", "", "whisk API host")
	AddStringFlag(create, "auth", "", "", "whisk auth")
	AddBoolFlag(create, "insecure", "i", false, "bypass certificate check")
	AddStringFlag(create, "debug", "", "", "Debug level output")
	AddStringFlag(create, "useragent", "", "aio-cli-plugin-runtime@4.0.0", "Use custom user-agent string")
	create.Flags().MarkHidden("useragent")
	AddStringFlag(create, "param", "p", "", "parameters in key value pairs to be passed to the package")
	AddStringFlag(create, "param-file", "P", "", "parameter to be passed to the package for json file")
	AddStringFlag(create, "shared", "", "", "parameter to be passed to indicate whether package is shared or private")
	AddStringFlag(create, "annotation", "a", "", "annotation values in KEY VALUE format")
	AddStringFlag(create, "annotation-file", "A", "", "FILE containing annotation values in JSON format")
	AddBoolFlag(create, "json", "", false, "output raw json")

	delete := cmdBuilderWithInit(cmd, RunPackagesDelete, "delete <packageName>", "Deletes a Package",
		`More information about 'packages delete'`,
		Writer, false)
	AddBoolFlag(delete, "recursive", "r", false, "Delete the contained actions")
	AddStringFlag(delete, "apihost", "", "", "Whisk API host")
	AddStringFlag(delete, "auth", "", "", "Whisk auth")
	AddBoolFlag(delete, "force", "f", false, "Just do it, omitting confirmatory prompt")
	AddBoolFlag(delete, "json", "", false, "output raw json")

	get := cmdBuilderWithInit(cmd, RunPackagesGet, "get <packageName>", "Retrieves a Package",
		`More information about 'packages get'`,
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

	list := cmdBuilderWithInit(cmd, RunPackagesList, "list [<namespace>]", "Lists all the Packages",
		`More information about 'packages list'`,
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
	AddStringFlag(list, "limit", "l", "", "only return LIMIT number of packages")
	AddStringFlag(list, "skip", "s", "", "exclude the first SKIP number of packages from the result")
	AddBoolFlag(list, "count", "", false, "show only the total number of packages")
	AddBoolFlag(list, "json", "", false, "output raw json")
	AddBoolFlag(list, "name-sort", "", false, "sort results by name")
	AddBoolFlag(list, "name", "n", false, "sort results by name")

	update := cmdBuilderWithInit(cmd, RunPackagesUpdate, "update <packageName>", "Updates a Package",
		`More information about 'packages update'`,
		Writer, false)
	AddStringFlag(update, "cert", "", "", "client cert")
	AddStringFlag(update, "key", "", "", "client key")
	AddStringFlag(update, "apiversion", "", "", "whisk API version")
	AddStringFlag(update, "apihost", "", "", "whisk API host")
	AddStringFlag(update, "auth", "", "", "whisk auth")
	AddBoolFlag(update, "insecure", "i", false, "bypass certificate check")
	AddStringFlag(update, "debug", "", "", "Debug level output")
	AddStringFlag(update, "useragent", "", "aio-cli-plugin-runtime@4.0.0", "Use custom user-agent string")
	update.Flags().MarkHidden("useragent")
	AddStringFlag(update, "param", "p", "", "parameters in key value pairs to be passed to the package")
	AddStringFlag(update, "param-file", "P", "", "parameter to be passed to the package for json file")
	AddStringFlag(update, "shared", "", "", "parameter to be passed to indicate whether package is shared or private")
	AddStringFlag(update, "annotation", "a", "", "annotation values in KEY VALUE format")
	AddStringFlag(update, "annotation-file", "A", "", "FILE containing annotation values in JSON format")
	AddBoolFlag(update, "json", "", false, "output raw json")

	return cmd
}
func RunPackagesBind(c *CmdConfig) error {
	argCount := len(c.Args)
	if argCount < 2 {
		return doctl.NewMissingArgsErr(c.NS)
	}
	if argCount > 2 {
		return doctl.NewTooManyArgsErr(c.NS)
	}
	output, err := RunSandboxExec("package/bind", c, []string{"insecure", "json"}, []string{"cert", "key", "apiversion", "apihost", "auth", "debug", "useragent", "param", "param-file", "annotation", "annotation-file"})
	if err != nil {
		return err
	}
	PrintSandboxTextOutput(output)
	return nil
}

func RunPackagesCreate(c *CmdConfig) error {
	err := ensureOneArg(c)
	if err != nil {
		return err
	}
	output, err := RunSandboxExec("package/create", c, []string{"insecure", "json"}, []string{"cert", "key", "apiversion", "apihost", "auth", "debug", "useragent", "param", "param-file", "shared", "annotation", "annotation-file"})
	if err != nil {
		return err
	}
	PrintSandboxTextOutput(output)
	return nil
}

func RunPackagesDelete(c *CmdConfig) error {
	err := ensureOneArg(c)
	if err != nil {
		return err
	}
	output, err := RunSandboxExec("package/delete", c, []string{"recursive", "force", "json"}, []string{"apihost", "auth"})
	if err != nil {
		return err
	}
	PrintSandboxTextOutput(output)
	return nil
}

func RunPackagesGet(c *CmdConfig) error {
	err := ensureOneArg(c)
	if err != nil {
		return err
	}
	output, err := RunSandboxExec("package/get", c, []string{"insecure"}, []string{"cert", "key", "apiversion", "apihost", "auth", "debug", "useragent"})
	if err != nil {
		return err
	}
	PrintSandboxTextOutput(output)
	return nil
}

func RunPackagesList(c *CmdConfig) error {
	argCount := len(c.Args)
	if argCount > 1 {
		return doctl.NewTooManyArgsErr(c.NS)
	}
	output, err := RunSandboxExec("package/list", c, []string{"insecure", "count", "json", "name-sort", "name"}, []string{"cert", "key", "apiversion", "apihost", "auth", "debug", "useragent", "limit", "skip"})
	if err != nil {
		return err
	}
	PrintSandboxTextOutput(output)
	return nil
}

func RunPackagesUpdate(c *CmdConfig) error {
	err := ensureOneArg(c)
	if err != nil {
		return err
	}
	output, err := RunSandboxExec("package/update", c, []string{"insecure", "json"}, []string{"cert", "key", "apiversion", "apihost", "auth", "debug", "useragent", "param", "param-file", "shared", "annotation", "annotation-file"})
	if err != nil {
		return err
	}
	PrintSandboxTextOutput(output)
	return nil
}

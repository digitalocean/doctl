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
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"

	"github.com/digitalocean/doctl"

	"golang.org/x/crypto/ssh/terminal"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	yaml "gopkg.in/yaml.v2"
)

// ErrUnknownTerminal signifies an unknown terminal. It is returned when doit
// can't ascertain the current terminal type with requesting an auth token.
var (
	ErrUnknownTerminal = errors.New("unknown terminal")
	cfgFileWriter      = defaultConfigFileWriter // create default cfgFileWriter
)

// retrieveUserTokenFromCommandLine is a function that can retrieve a token. By default,
// it will prompt the user. In test, you can replace this with code that returns the appropriate response.
func retrieveUserTokenFromCommandLine() (string, error) {
	if !terminal.IsTerminal(int(os.Stdout.Fd())) {
		return "", ErrUnknownTerminal
	}

	fmt.Print("Please authenticate doctl for use with your DigitalOcean account. You can generate a token in the control panel at https://cloud.digitalocean.com/account/api/tokens\n\n")
	fmt.Print("Enter your access token: ")
	passwdBytes, err := terminal.ReadPassword(int(syscall.Stdin))
	if err != nil {
		return "", err
	}

	return string(passwdBytes), nil
}

// UnknownSchemeError signifies an unknown HTTP scheme.
type UnknownSchemeError struct {
	Scheme string
}

var _ error = &UnknownSchemeError{}

func (use *UnknownSchemeError) Error() string {
	return "Unknown scheme: " + use.Scheme
}

// Auth creates auth commands for doctl.
func Auth() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:   "auth",
			Short: "Display commands for authenticating doctl with an account",
			Long: `The ` + "`" + `doctl auth` + "`" + ` commands allow you to authenticate doctl for use with your DigitalOcean account using tokens that you generate in the control panel at https://cloud.digitalocean.com/account/api/tokens.

If you work with a just one account, you can call ` + "`" + `doctl auth init` + "`" + ` and supply the token when prompted. This creates an authentication context named ` + "`" + `default` + "`" + `.

To switch between multiple DigitalOcean accounts, including team accounts, you can create named contexts by using ` + "`" + `doctl auth init --context <name>` + "`" + `, then providing a token when prompted. This saves the token under the name you provide. To switch between accounts, use ` + "`" + `doctl auth switch --context <name>` + "`" + `.,

To remove accounts from the configuration file, you can run ` + "`" + `doctl auth remove --context <name>` + "`" + `. This removes the token under the name you provide.`,
		},
	}

	cmdBuilderWithInit(cmd, RunAuthInit(retrieveUserTokenFromCommandLine), "init", "Initialize doctl to use a specific account", `This command allows you to initialize doctl with a token that allows it to query and manage your account details and resources.

You will need an API token, which you can generate in the control panel at https://cloud.digitalocean.com/account/api/tokens.

You can provide a name to this initialization via the `+"`"+`--context`+"`"+` flag, and then it will be saved as an "authentication context". Authentication contexts are accessible via `+"`"+`doctl auth switch`+"`"+`, which re-initializes doctl, or by providing the `+"`"+`--context`+"`"+` flag when using any doctl command (to specify that auth context for just one command). This enables you to use multiple DigitalOcean accounts with doctl, or tokens that have different authentication scopes.

If the `+"`"+`--context`+"`"+` flag is not specified, a default authentication context will be created during initialization.

If doctl is never initialized, you will need to specify an API token whenever you use a `+"`"+`doctl`+"`"+` command via the `+"`"+`--access-token`+"`"+` flag.`, Writer, false)
	cmdAuthSwitch := cmdBuilderWithInit(cmd, RunAuthSwitch, "switch", "Switches between authentication contexts", `This command allows you to switch between accounts with authentication contexts you've already created.

To see a list of available authentication contexts, call `+"`"+`doctl auth list`+"`"+`.

For details on creating an authentication context, see the help for `+"`"+`doctl auth init`+"`"+`.`, Writer, false)
	cmdAuthSwitch.AddValidArgsFunc(authContextListValidArgsFunc)

	cmdAuthRemove := cmdBuilderWithInit(cmd, RunAuthRemove, "remove", "Remove authentication contexts ", `This command allows you to remove authentication contexts you've already created.

To see a list of available authentication contexts, call `+"`"+`doctl auth list`+"`"+`.

For details on creating an authentication context, see the help for `+"`"+`doctl auth init`+"`"+`.`, Writer, false)
	cmdAuthRemove.AddValidArgsFunc(authContextListValidArgsFunc)

	cmdAuthList := cmdBuilderWithInit(cmd, RunAuthList, "list", "List available authentication contexts", `List named authentication contexts that you created with `+"`"+`doctl auth init`+"`"+`.

To switch between the contexts use `+"`"+`doctl auth switch --context <name>`+"`"+`, where `+"`"+`<name>`+"`"+` is one of the contexts listed.

To create new contexts, see the help for `+"`"+`doctl auth init`+"`"+`.`, Writer, false, aliasOpt("ls"))
	// The command runner expects that any command named "list" accepts a
	// format flag, so we include here despite only supporting text output for
	// this command.
	AddStringFlag(cmdAuthList, doctl.ArgFormat, "", "", "Columns for output in a comma-separated list. Possible values: `text`")

	return cmd
}

// RunAuthInit initializes the doctl config. Configuration is stored in $XDG_CONFIG_HOME/doctl. On Unix, if
// XDG_CONFIG_HOME is not set, use $HOME/.config. On Windows use %APPDATA%/doctl/config.
func RunAuthInit(retrieveUserTokenFunc func() (string, error)) func(c *CmdConfig) error {
	return func(c *CmdConfig) error {
		token := c.getContextAccessToken()

		if token == "" {
			in, err := retrieveUserTokenFunc()
			if err != nil {
				return fmt.Errorf("Unable to read DigitalOcean access token: %s", err)
			}
			token = strings.TrimSpace(in)
		} else {
			fmt.Fprintf(c.Out, "Using token [%v]", token)
			fmt.Fprintln(c.Out)
		}

		c.setContextAccessToken(token)

		fmt.Fprintln(c.Out)
		fmt.Fprint(c.Out, "Validating token... ")

		// need to initial the godo client since we've changed the configuration.
		if err := c.initServices(c); err != nil {
			return fmt.Errorf("Unable to initialize DigitalOcean API client with new token: %s", err)
		}

		if _, err := c.Account().Get(); err != nil {
			fmt.Fprintln(c.Out, "invalid token")
			fmt.Fprintln(c.Out)
			return fmt.Errorf("Unable to use supplied token to access API: %s", err)
		}

		fmt.Fprintln(c.Out, "OK")
		fmt.Fprintln(c.Out)

		return writeConfig()
	}
}

// RunAuthRemove remove available auth contexts from the user's doctl config.
func RunAuthRemove(c *CmdConfig) error {
	context := Context

	if context == "" {
		return fmt.Errorf("You must provide a context name")
	}

	err := c.removeContext(context)

	if err != nil {
		return fmt.Errorf("Context not found")
	}

	fmt.Println("Context deleted successfully")

	return writeConfig()
}

// RunAuthList lists all available auth contexts from the user's doctl config.
func RunAuthList(c *CmdConfig) error {
	context := Context
	if context == "" {
		context = viper.GetString("context")
	}
	contexts := viper.GetStringMap("auth-contexts")

	displayAuthContexts(c.Out, context, contexts)
	return nil
}

func displayAuthContexts(out io.Writer, currentContext string, contexts map[string]interface{}) {
	// Because the default context isn't present on the auth-contexts field,
	// we add it manually so that it's always included in the output, and so
	// we can check if it's the current context.
	contexts[doctl.ArgDefaultContext] = true

	// Extract and sort the map keys so that the order that we display the
	// auth contexts is consistent.
	keys := make([]string, 0)
	for ctx := range contexts {
		keys = append(keys, ctx)
	}
	sort.Strings(keys)

	for _, ctx := range keys {
		if ctx == currentContext {
			fmt.Fprintln(out, ctx, "(current)")
			continue
		}
		fmt.Fprintln(out, ctx)
	}
}

// RunAuthSwitch changes the default context and writes it to the
// configuration.
func RunAuthSwitch(c *CmdConfig) error {
	context := Context
	if context == "" {
		context = viper.GetString("context")
	}

	// The two lines below aren't required for doctl specific functionality,
	// but somehow magically fixes an issue
	// (https://github.com/digitalocean/doctl/issues/996) where auth-contexts
	// are mangled when running this command.
	contexts := viper.GetStringMapString("auth-contexts")
	viper.Set("auth-contexts", contexts)

	viper.Set("context", context)

	fmt.Printf("Now using context [%s] by default\n", context)
	return writeConfig()
}

func writeConfig() error {
	f, err := cfgFileWriter()
	if err != nil {
		return err
	}

	defer f.Close()

	b, err := yaml.Marshal(viper.AllSettings())
	if err != nil {
		return errors.New("Unable to encode configuration to YAML format.")
	}

	_, err = f.Write(b)
	if err != nil {
		return errors.New("Unable to write configuration.")
	}

	return nil
}

// defaultConfigFileWriter returns a writer to a newly created config.yaml file in the default config home.
// When using the default config file path the default config home directory will be created; Otherwise
// the custom config home directory must exist and be writable to the user issuing the auth command.
func defaultConfigFileWriter() (io.WriteCloser, error) {
	cfgFile := viper.GetString("config")

	defaultCfgFile := filepath.Join(defaultConfigHome(), defaultConfigName)
	if cfgFile == defaultCfgFile {
		configHome()
	}

	f, err := os.Create(cfgFile)
	if err != nil {
		return nil, err
	}
	if err := os.Chmod(cfgFile, 0600); err != nil {
		return nil, err
	}

	return f, nil
}

func getAuthContextList() []string {
	contexts := []string{"default"}
	cfgContexts := viper.GetStringMap("auth-contexts")

	for k := range cfgContexts {
		contexts = append(contexts, k)
	}

	return contexts
}

func authContextListValidArgsFunc(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) != 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	return getAuthContextList(), cobra.ShellCompDirectiveNoFileComp
}

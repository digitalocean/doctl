package commands

import (
	"fmt"
	"io"
	"os"
	"os/user"
	"path/filepath"

	"github.com/bryanl/doit"
	"github.com/bryanl/doit/Godeps/_workspace/src/github.com/digitalocean/godo"
	"github.com/bryanl/doit/Godeps/_workspace/src/github.com/fatih/color"
	"github.com/bryanl/doit/Godeps/_workspace/src/github.com/spf13/cobra"
	"github.com/bryanl/doit/Godeps/_workspace/src/github.com/spf13/viper"
)

const (
	configFile = ".doitcfg"
)

var (
	// DoitCmd is the base command.
	DoitCmd = &cobra.Command{
		Use: "doit",
	}

	// Token holds the global authorization token.
	Token string

	// Output holds the global output format.
	Output string

	requiredColor = color.New(color.Bold, color.FgWhite).SprintfFunc()

	writer = os.Stdout
)

func init() {
	viper.SetConfigType("yaml")

	DoitCmd.PersistentFlags().StringVarP(&Token, "access-token", "t", "", "DigtialOcean API V2 Access Token")
	DoitCmd.PersistentFlags().StringVarP(&Output, "output", "o", "text", "output formt [text|json]")
}

// LoadConfig loads out configuration.
func LoadConfig() error {
	fp, err := configFilePath()
	if err != nil {
		return fmt.Errorf("can't find home directory: %v", err)
	}
	if _, err := os.Stat(fp); err == nil {
		file, err := os.Open(fp)
		if err != nil {
			return fmt.Errorf("can't open configuration file %q: %v", fp, err)
		}
		viper.ReadConfig(file)
	}

	return nil
}

// Execute executes the base command.
func Execute() {
	initializeConfig()
	addCommands()
	DoitCmd.Execute()
}

// AddCommands adds sub commands to the base command.
func addCommands() {
	DoitCmd.AddCommand(Account())
	DoitCmd.AddCommand(Actions())
	DoitCmd.AddCommand(Domain())
	DoitCmd.AddCommand(DropletAction())
	DoitCmd.AddCommand(Droplet())
	DoitCmd.AddCommand(FloatingIP())
	DoitCmd.AddCommand(FloatingIPAction())
	DoitCmd.AddCommand(Images())
	DoitCmd.AddCommand(Region())
	DoitCmd.AddCommand(Size())
	DoitCmd.AddCommand(SSHKeys())
	DoitCmd.AddCommand(SSH())
}

func initFlags() {
	viper.SetEnvPrefix("DIGITALOCEAN")
	viper.BindEnv("access-token", "DIGITALOCEAN_ACCESS_TOKEN")
	viper.BindPFlag("access-token", DoitCmd.PersistentFlags().Lookup("access-token"))
	viper.BindPFlag("output", DoitCmd.PersistentFlags().Lookup("output"))
}

func loadDefaultSettings() {
	viper.SetDefault("output", "text")
}

// InitializeConfig initializes the doit configuration.
func initializeConfig() {
	loadDefaultSettings()
	LoadConfig()
	initFlags()

	if DoitCmd.PersistentFlags().Lookup("access-token").Changed {
		viper.Set("access-token", Token)
	}

	if DoitCmd.PersistentFlags().Lookup("output").Changed {
		viper.Set("output", Output)
	}
}

func configFilePath() (string, error) {
	usr, err := user.Current()
	if err != nil {
		return "", err
	}

	dir := filepath.Join(usr.HomeDir, configFile)
	return dir, nil
}

type flagOpt func(c *cobra.Command, name, key string)

func requiredOpt() flagOpt {
	return func(c *cobra.Command, name, key string) {
		c.MarkFlagRequired(key)
		key = requiredKey(key)
		viper.Set(key, true)

		u := c.Flag(name).Usage
		c.Flag(name).Usage = fmt.Sprintf("%s %s", u, requiredColor("(required)"))
	}
}

func requiredKey(key string) string {
	return fmt.Sprintf("%s.required", key)
}

func addStringFlag(cmd *cobra.Command, name, dflt, desc string, opts ...flagOpt) {
	fn := flagName(cmd, name)
	cmd.Flags().String(name, dflt, desc)
	viper.BindPFlag(fn, cmd.Flags().Lookup(name))

	for _, o := range opts {
		o(cmd, name, fn)
	}
}

func addIntFlag(cmd *cobra.Command, name string, def int, desc string, opts ...flagOpt) {
	fn := flagName(cmd, name)
	cmd.Flags().Int(name, def, desc)
	viper.BindPFlag(fn, cmd.Flags().Lookup(name))

	for _, o := range opts {
		o(cmd, name, fn)
	}
}

func addBoolFlag(cmd *cobra.Command, name string, def bool, desc string, opts ...flagOpt) {
	fn := flagName(cmd, name)
	cmd.Flags().Bool(name, def, desc)
	viper.BindPFlag(fn, cmd.Flags().Lookup(name))

	for _, o := range opts {
		o(cmd, name, fn)
	}
}

func addStringSliceFlag(cmd *cobra.Command, name string, def []string, desc string, opts ...flagOpt) {
	fn := flagName(cmd, name)
	cmd.Flags().StringSlice(name, def, desc)
	viper.BindPFlag(fn, cmd.Flags().Lookup(name))

	for _, o := range opts {
		o(cmd, name, fn)
	}
}

func flagName(cmd *cobra.Command, name string) string {
	parentName := doit.NSRoot
	if cmd.Parent() != nil {
		parentName = cmd.Parent().Name()
	}

	return fmt.Sprintf("%s.%s.%s", parentName, cmd.Name(), name)
}

func cmdNS(cmd *cobra.Command) string {
	parentName := doit.NSRoot
	if cmd.Parent() != nil {
		parentName = cmd.Parent().Name()
	}

	return fmt.Sprintf("%s.%s", parentName, cmd.Name())
}

type cmdRunner func(ns string, config doit.Config, out io.Writer) error

type cmdOption func(*cobra.Command)

func aliasOpt(aliases ...string) cmdOption {
	return func(c *cobra.Command) {
		if c.Aliases == nil {
			c.Aliases = []string{}
		}

		for _, a := range aliases {
			c.Aliases = append(c.Aliases, a)
		}
	}
}

func cmdBuilder(cr cmdRunner, cliText, desc string, out io.Writer, options ...cmdOption) *cobra.Command {
	c := &cobra.Command{
		Use:   cliText,
		Short: desc,
		Long:  desc,
		Run: func(cmd *cobra.Command, args []string) {
			checkErr(cr(cmdNS(cmd), doit.DoitConfig, out), cmd)
		},
	}

	for _, co := range options {
		co(c)
	}

	return c
}

func listDroplets(client *godo.Client) ([]godo.Droplet, error) {
	f := func(opt *godo.ListOptions) ([]interface{}, *godo.Response, error) {
		list, resp, err := client.Droplets.List(opt)
		if err != nil {
			return nil, nil, err
		}

		si := make([]interface{}, len(list))
		for i := range list {
			si[i] = list[i]
		}

		return si, resp, err
	}

	si, err := doit.PaginateResp(f)
	if err != nil {
		return nil, err
	}

	list := make([]godo.Droplet, len(si))
	for i := range si {
		list[i] = si[i].(godo.Droplet)
	}

	return list, nil
}

func extractDropletPublicIP(droplet *godo.Droplet) string {
	for _, in := range droplet.Networks.V4 {
		if in.Type == "public" {
			return in.IPAddress
		}
	}

	return ""

}

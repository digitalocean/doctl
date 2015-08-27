package commands

import (
	"fmt"
	"io"
	"os"
	"os/user"
	"path/filepath"

	"github.com/bryanl/doit"
	"github.com/digitalocean/godo"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
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

	writer = os.Stdout
)

func init() {
	viper.SetConfigType("yaml")

	DoitCmd.PersistentFlags().StringVarP(&Token, "token", "t", "", "DigtialOcean API V2 Token")
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
	DoitCmd.AddCommand(Images())
	DoitCmd.AddCommand(Region())
	DoitCmd.AddCommand(Plugin())
	DoitCmd.AddCommand(Size())
	DoitCmd.AddCommand(SSHKeys())
	DoitCmd.AddCommand(SSH())
}

func initFlags() {
	viper.SetEnvPrefix("DIGITALOCEAN")
	viper.BindEnv("token", "ACCESS_TOKEN")
	viper.BindPFlag("token", DoitCmd.PersistentFlags().Lookup("token"))
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

	if DoitCmd.PersistentFlags().Lookup("token").Changed {
		viper.Set("token", Token)
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

func addStringFlag(cmd *cobra.Command, name, dflt, desc string) {
	fn := flagName(cmd, name)
	cmd.Flags().String(name, dflt, desc)
	viper.BindPFlag(fn, cmd.Flags().Lookup(name))
}

func addIntFlag(cmd *cobra.Command, name string, def int, desc string) {
	fn := flagName(cmd, name)
	cmd.Flags().Int(name, def, desc)
	viper.BindPFlag(fn, cmd.Flags().Lookup(name))
}

func addBoolFlag(cmd *cobra.Command, name string, def bool, desc string) {
	fn := flagName(cmd, name)
	cmd.Flags().Bool(name, def, desc)
	viper.BindPFlag(fn, cmd.Flags().Lookup(name))
}

func addStringSliceFlag(cmd *cobra.Command, name string, def []string, desc string) {
	fn := flagName(cmd, name)
	cmd.Flags().StringSlice(name, def, desc)
	viper.BindPFlag(fn, cmd.Flags().Lookup(name))
}

func flagName(cmd *cobra.Command, name string) string {
	parentName := doit.NSRoot
	if cmd.Parent() != nil {
		parentName = cmd.Parent().Name()
	}

	return fmt.Sprintf("%s-%s-%s", parentName, cmd.Name(), name)
}

func cmdNS(cmd *cobra.Command) string {
	parentName := doit.NSRoot
	if cmd.Parent() != nil {
		parentName = cmd.Parent().Name()
	}

	return fmt.Sprintf("%s-%s", parentName, cmd.Name())
}

type cmdRunner func(ns string, out io.Writer) error

func cmdBuilder(cr cmdRunner, cliText, desc string, out io.Writer) *cobra.Command {
	return &cobra.Command{
		Use:   cliText,
		Short: desc,
		Long:  desc,
		Run: func(cmd *cobra.Command, args []string) {
			checkErr(cr(cmdNS(cmd), out), cmd)
		},
	}
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

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
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/digitalocean/doctl"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	defaultConfigName = "config.yaml" // default name of config file
)

var (
	//DoitCmd is the root level doctl command that all other commands attach to
	DoitCmd = &Command{ // base command
		Command: &cobra.Command{
			Use:   "doctl",
			Short: "doctl is a command line interface (CLI) for the DigitalOcean API.",
		},
	}

	//Writer wires up stdout for all commands to write to
	Writer = os.Stdout
	//APIURL customize API base URL
	APIURL string
	//Context current auth context
	Context string
	//Output global output format
	Output string
	//Token global authorization token
	Token string
	//Trace toggles http tracing output
	Trace bool
	//Verbose toggle verbose output on and off
	Verbose bool

	requiredColor = color.New(color.Bold).SprintfFunc()
)

func init() {
	var cfgFile string

	initConfig()

	rootPFlagSet := DoitCmd.PersistentFlags()
	rootPFlagSet.StringVarP(&cfgFile, "config", "c",
		filepath.Join(defaultConfigHome(), defaultConfigName), "Specify a custom config file")
	viper.BindPFlag("config", rootPFlagSet.Lookup("config"))

	rootPFlagSet.StringVarP(&APIURL, "api-url", "u", "", "Override default API endpoint")
	viper.BindPFlag("api-url", rootPFlagSet.Lookup("api-url"))

	rootPFlagSet.StringVarP(&Token, doctl.ArgAccessToken, "t", "", "API V2 access token")
	viper.BindPFlag(doctl.ArgAccessToken, rootPFlagSet.Lookup(doctl.ArgAccessToken))

	rootPFlagSet.StringVarP(&Output, doctl.ArgOutput, "o", "text", "Desired output format [text|json]")
	viper.BindPFlag("output", rootPFlagSet.Lookup(doctl.ArgOutput))

	rootPFlagSet.StringVarP(&Context, doctl.ArgContext, "", "", "Specify a custom authentication context name")
	DoitCmd.RegisterFlagCompletionFunc(doctl.ArgContext, func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return getAuthContextList(), cobra.ShellCompDirectiveNoFileComp
	})

	rootPFlagSet.BoolVarP(&Trace, "trace", "", false, "Show a log of network activity while performing a command")
	rootPFlagSet.BoolVarP(&Verbose, doctl.ArgVerbose, "v", false, "Enable verbose output")

	addCommands()

	cobra.OnInitialize(initConfig)
}

func initConfig() {
	viper.SetEnvPrefix("DIGITALOCEAN")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.SetConfigType("yaml")

	cfgFile := viper.GetString("config")
	viper.SetConfigFile(cfgFile)

	viper.SetDefault("output", "text")
	viper.SetDefault(doctl.ArgContext, doctl.ArgDefaultContext)

	if _, err := os.Stat(cfgFile); err == nil {
		if err := viper.ReadInConfig(); err != nil {
			log.Fatalln("Config initialization failed:", err)
		}
	}
}

// in case we ever want to change this, or let folks configure it...
func defaultConfigHome() string {
	cfgDir, err := os.UserConfigDir()
	checkErr(err)

	return filepath.Join(cfgDir, "doctl")
}

func configHome() string {
	ch := defaultConfigHome()
	err := os.MkdirAll(ch, 0755)
	checkErr(err)

	return ch
}

// Execute executes the current command using DoitCmd.
func Execute() {
	if err := DoitCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
}

// AddCommands adds sub commands to the base command.
func addCommands() {
	DoitCmd.AddCommand(Account())
	DoitCmd.AddCommand(Apps())
	DoitCmd.AddCommand(Auth())
	DoitCmd.AddCommand(Balance())
	DoitCmd.AddCommand(BillingHistory())
	DoitCmd.AddCommand(Invoices())
	DoitCmd.AddCommand(computeCmd())
	DoitCmd.AddCommand(Kubernetes())
	DoitCmd.AddCommand(Databases())
	DoitCmd.AddCommand(Projects())
	DoitCmd.AddCommand(Version())
	DoitCmd.AddCommand(Registry())
	DoitCmd.AddCommand(VPCs())
	DoitCmd.AddCommand(OneClicks())
	DoitCmd.AddCommand(Monitoring())
}

func computeCmd() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:   "compute",
			Short: "Display commands that manage infrastructure",
			Long:  `The subcommands under ` + "`" + `doctl compute` + "`" + ` are for managing DigitalOcean resources.`,
		},
	}

	cmd.AddCommand(Actions())
	cmd.AddCommand(CDN())
	cmd.AddCommand(Certificate())
	cmd.AddCommand(DropletAction())
	cmd.AddCommand(Droplet())
	cmd.AddCommand(Domain())
	cmd.AddCommand(Firewall())
	cmd.AddCommand(FloatingIP())
	cmd.AddCommand(FloatingIPAction())
	cmd.AddCommand(Images())
	cmd.AddCommand(ImageAction())
	cmd.AddCommand(LoadBalancer())
	cmd.AddCommand(Plugin())
	cmd.AddCommand(Region())
	cmd.AddCommand(Size())
	cmd.AddCommand(Snapshot())
	cmd.AddCommand(SSHKeys())
	cmd.AddCommand(Tags())
	cmd.AddCommand(Volume())
	cmd.AddCommand(VolumeAction())

	// SSH is different since it doesn't have any subcommands. In this case, let's
	// give it a parent at init time.
	SSH(cmd)

	return cmd
}

type flagOpt func(c *Command, name, key string)

func requiredOpt() flagOpt {
	return func(c *Command, name, key string) {
		c.MarkFlagRequired(key)

		key = fmt.Sprintf("required.%s", key)
		viper.Set(key, true)

		u := c.Flag(name).Usage
		c.Flag(name).Usage = fmt.Sprintf("%s %s", u, requiredColor("(required)"))
	}
}

// AddStringFlag adds a string flag to a command.
func AddStringFlag(cmd *Command, name, shorthand, dflt, desc string, opts ...flagOpt) {
	fn := flagName(cmd, name)
	cmd.Flags().StringP(name, shorthand, dflt, desc)

	for _, o := range opts {
		o(cmd, name, fn)
	}

	viper.BindPFlag(fn, cmd.Flags().Lookup(name))
}

// AddIntFlag adds an integr flag to a command.
func AddIntFlag(cmd *Command, name, shorthand string, def int, desc string, opts ...flagOpt) {
	fn := flagName(cmd, name)
	cmd.Flags().IntP(name, shorthand, def, desc)
	viper.BindPFlag(fn, cmd.Flags().Lookup(name))

	for _, o := range opts {
		o(cmd, name, fn)
	}
}

// AddBoolFlag adds a boolean flag to a command.
func AddBoolFlag(cmd *Command, name, shorthand string, def bool, desc string, opts ...flagOpt) {
	fn := flagName(cmd, name)
	cmd.Flags().BoolP(name, shorthand, def, desc)
	viper.BindPFlag(fn, cmd.Flags().Lookup(name))

	for _, o := range opts {
		o(cmd, name, fn)
	}
}

// AddStringSliceFlag adds a string slice flag to a command.
func AddStringSliceFlag(cmd *Command, name, shorthand string, def []string, desc string, opts ...flagOpt) {
	fn := flagName(cmd, name)
	cmd.Flags().StringSliceP(name, shorthand, def, desc)
	viper.BindPFlag(fn, cmd.Flags().Lookup(name))

	for _, o := range opts {
		o(cmd, name, fn)
	}
}

// AddStringMapStringFlag adds a map of strings by strings flag to a command.
func AddStringMapStringFlag(cmd *Command, name, shorthand string, def map[string]string, desc string, opts ...flagOpt) {
	fn := flagName(cmd, name)
	cmd.Flags().StringToStringP(name, shorthand, def, desc)
	viper.BindPFlag(fn, cmd.Flags().Lookup(name))

	for _, o := range opts {
		o(cmd, name, fn)
	}
}

func flagName(cmd *Command, name string) string {
	if cmd.Parent() != nil {
		return fmt.Sprintf("%s.%s.%s", cmd.Parent().Name(), cmd.Name(), name)
	}
	return fmt.Sprintf("%s.%s", cmd.Name(), name)
}

func cmdNS(cmd *cobra.Command) string {
	if cmd.Parent() != nil {
		return fmt.Sprintf("%s.%s", cmd.Parent().Name(), cmd.Name())
	}
	return fmt.Sprintf("%s", cmd.Name())
}

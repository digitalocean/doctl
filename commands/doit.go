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
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/commands/displayers"
	"github.com/digitalocean/doctl/do"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v2"
)

const (
	// defaultConfigName is the name of the config file when no alternative is supplied.
	defaultConfigName = "config.yaml"
)

// DoitCmd is the base command.
var DoitCmd = &Command{
	Command: &cobra.Command{
		Use:   "doctl",
		Short: "doctl is a command line interface for the DigitalOcean API.",
	},
}

// Context holds the current auth context
var Context string

// ApiURL holds the API URL to use.
var ApiURL string

// Token holds the global authorization token.
var Token string

// Output holds the global output format.
var Output string

// Verbose toggles verbose output.
var Verbose bool

var requiredColor = color.New(color.Bold).SprintfFunc()

// Writer is where output should be written to.
var Writer = os.Stdout

// Trace toggles http tracing output.
var Trace bool

// cfgFile is the location of the config file
var cfgFile string

// cfgFileWriter is the config file writer
var cfgFileWriter = defaultConfigFileWriter

// ErrNoAccessToken is an error for when there is no access token.
var ErrNoAccessToken = errors.New("no access token has been configured")

func init() {
	cobra.OnInitialize(initConfig)

	DoitCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "config file (default is $HOME/.config/doctl/config.yaml)")
	DoitCmd.PersistentFlags().StringVarP(&Token, doctl.ArgAccessToken, "t", "", "API V2 Access Token")
	DoitCmd.PersistentFlags().StringVarP(&Output, "output", "o", "text", "output format [text|json]")
	DoitCmd.PersistentFlags().StringVarP(&ApiURL, "api-url", "u", "", "Override default API V2 endpoint")
	DoitCmd.PersistentFlags().BoolVarP(&Verbose, doctl.ArgVerbose, "v", false, "verbose output")
	DoitCmd.PersistentFlags().BoolVarP(&Trace, "trace", "", false, "trace api access")

	DoitCmd.PersistentFlags().StringVarP(&Context, doctl.ArgContext, "", "", "authentication context name")

	viper.SetEnvPrefix("DIGITALOCEAN")
	viper.BindEnv(doctl.ArgAccessToken, "DIGITALOCEAN_ACCESS_TOKEN")
	viper.BindPFlag(doctl.ArgAccessToken, DoitCmd.PersistentFlags().Lookup("access-token"))
	viper.BindEnv("api-url", "DIGITALOCEAN_API_URL")
	viper.BindPFlag("api-url", DoitCmd.PersistentFlags().Lookup("api-url"))
	viper.BindPFlag("output", DoitCmd.PersistentFlags().Lookup("output"))
	viper.BindEnv("enable-beta", "DIGITALOCEAN_ENABLE_BETA")

	addCommands()
}

func initConfig() {
	var err error
	cfgFile, err = findConfig()
	if err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}

	legacyConfigCheck()

	viper.SetConfigType("yaml")
	viper.SetConfigFile(cfgFile)

	viper.AutomaticEnv()

	if _, err := os.Stat(cfgFile); err == nil {
		if err := viper.ReadInConfig(); err != nil {
			log.Fatalln("reading initialization failed:", err)
		}
	}

	viper.SetDefault("output", "text")
	viper.SetDefault("context", "default")
}

func findConfig() (string, error) {
	if cfgFile != "" {
		return cfgFile, nil
	}

	if homeDir() != "" {
		legacyConfigPath := filepath.Join(homeDir(), ".doctlcfg")
		if _, err := os.Stat(legacyConfigPath); err == nil {
			msg := fmt.Sprintf("Configuration detected at %q. Please move .doctlcfg to %s",
				legacyConfigPath, configPath())
			warn(msg)
		}
	}

	if os.Getenv("XDG_CONFIG_HOME") != "" {
		legacyXDGPath := filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "config.yaml")
		if _, err := os.Stat(legacyXDGPath); err == nil {
			msg := fmt.Sprintf("Configuration detected at %q. Please move config.yaml to %s",
				legacyXDGPath, configPath())
			warn(msg)
		}
	}

	ch := configHome()
	if err := os.MkdirAll(ch, 0755); err != nil {
		return "", err
	}

	return filepath.Join(ch, defaultConfigName), nil
}

func configPath() string {
	return fmt.Sprintf("%s/%s", configHome(), defaultConfigName)
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
	DoitCmd.AddCommand(Auth())
	DoitCmd.AddCommand(Completion())
	DoitCmd.AddCommand(computeCmd())
	DoitCmd.AddCommand(Kubernetes())
	DoitCmd.AddCommand(Projects())
	DoitCmd.AddCommand(Version())
}

func computeCmd() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:   "compute",
			Short: "compute commands",
			Long:  "compute commands are for controlling and managing infrastructure",
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

		key = requiredKey(key)
		viper.Set(key, true)

		u := c.Flag(name).Usage
		c.Flag(name).Usage = fmt.Sprintf("%s %s", u, requiredColor("(required)"))
	}
}

func requiredKey(key string) string {
	return fmt.Sprintf("required.%s", key)
}

func betaOpt() flagOpt {
	return func(c *Command, name, key string) {
		c.Flag(name).Hidden = !isBeta()
	}
}

func isBeta() bool {
	return viper.GetBool("enable-beta")
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

func flagName(cmd *Command, name string) string {
	parentName := doctl.NSRoot
	if cmd.Parent() != nil {
		parentName = cmd.Parent().Name()
	}

	return fmt.Sprintf("%s.%s.%s", parentName, cmd.Name(), name)
}

func cmdNS(cmd *cobra.Command) string {
	parentName := doctl.NSRoot
	if cmd.Parent() != nil {
		parentName = cmd.Parent().Name()
	}

	return fmt.Sprintf("%s.%s", parentName, cmd.Name())
}

// CmdRunner runs a command and passes in a cmdConfig.
type CmdRunner func(*CmdConfig) error

// CmdConfig is a command configuration.
type CmdConfig struct {
	NS   string
	Doit doctl.Config
	Out  io.Writer
	Args []string

	initServices          func(*CmdConfig) error
	getContextAccessToken func() string
	setContextAccessToken func(string)

	// services
	Keys              func() do.KeysService
	Sizes             func() do.SizesService
	Regions           func() do.RegionsService
	Images            func() do.ImagesService
	ImageActions      func() do.ImageActionsService
	LoadBalancers     func() do.LoadBalancersService
	FloatingIPs       func() do.FloatingIPsService
	FloatingIPActions func() do.FloatingIPActionsService
	Droplets          func() do.DropletsService
	DropletActions    func() do.DropletActionsService
	Domains           func() do.DomainsService
	Actions           func() do.ActionsService
	Account           func() do.AccountService
	Tags              func() do.TagsService
	Volumes           func() do.VolumesService
	VolumeActions     func() do.VolumeActionsService
	Snapshots         func() do.SnapshotsService
	Certificates      func() do.CertificatesService
	Firewalls         func() do.FirewallsService
	CDNs              func() do.CDNsService
	Projects          func() do.ProjectsService
	Kubernetes        func() do.KubernetesService
}

// NewCmdConfig creates an instance of a CmdConfig.
func NewCmdConfig(ns string, dc doctl.Config, out io.Writer, args []string, initGodo bool) (*CmdConfig, error) {

	cmdConfig := &CmdConfig{
		NS:   ns,
		Doit: dc,
		Out:  out,
		Args: args,

		initServices: func(c *CmdConfig) error {
			accessToken := c.getContextAccessToken()
			godoClient, err := c.Doit.GetGodoClient(Trace, accessToken)
			if err != nil {
				return fmt.Errorf("unable to initialize DigitalOcean api client: %s", err)
			}

			c.Keys = func() do.KeysService { return do.NewKeysService(godoClient) }
			c.Sizes = func() do.SizesService { return do.NewSizesService(godoClient) }
			c.Regions = func() do.RegionsService { return do.NewRegionsService(godoClient) }
			c.Images = func() do.ImagesService { return do.NewImagesService(godoClient) }
			c.ImageActions = func() do.ImageActionsService { return do.NewImageActionsService(godoClient) }
			c.FloatingIPs = func() do.FloatingIPsService { return do.NewFloatingIPsService(godoClient) }
			c.FloatingIPActions = func() do.FloatingIPActionsService { return do.NewFloatingIPActionsService(godoClient) }
			c.Droplets = func() do.DropletsService { return do.NewDropletsService(godoClient) }
			c.DropletActions = func() do.DropletActionsService { return do.NewDropletActionsService(godoClient) }
			c.Domains = func() do.DomainsService { return do.NewDomainsService(godoClient) }
			c.Actions = func() do.ActionsService { return do.NewActionsService(godoClient) }
			c.Account = func() do.AccountService { return do.NewAccountService(godoClient) }
			c.Tags = func() do.TagsService { return do.NewTagsService(godoClient) }
			c.Volumes = func() do.VolumesService { return do.NewVolumesService(godoClient) }
			c.VolumeActions = func() do.VolumeActionsService { return do.NewVolumeActionsService(godoClient) }
			c.Snapshots = func() do.SnapshotsService { return do.NewSnapshotsService(godoClient) }
			c.Certificates = func() do.CertificatesService { return do.NewCertificatesService(godoClient) }
			c.LoadBalancers = func() do.LoadBalancersService { return do.NewLoadBalancersService(godoClient) }
			c.Firewalls = func() do.FirewallsService { return do.NewFirewallsService(godoClient) }
			c.CDNs = func() do.CDNsService { return do.NewCDNsService(godoClient) }
			c.Projects = func() do.ProjectsService { return do.NewProjectsService(godoClient) }
			c.Kubernetes = func() do.KubernetesService { return do.NewKubernetesService(godoClient) }

			return nil
		},

		getContextAccessToken: func() string {
			context := Context
			if context == "" {
				context = viper.GetString("context")
			}
			token := ""

			switch context {
			case "default":
				token = viper.GetString(doctl.ArgAccessToken)
			default:
				contexts := viper.GetStringMapString("auth-contexts")

				token = contexts[context]
			}

			return token
		},

		setContextAccessToken: func(token string) {
			context := Context
			if context == "" {
				context = viper.GetString("context")
			}

			switch context {
			case "default":
				viper.Set(doctl.ArgAccessToken, token)
			default:
				contexts := viper.GetStringMapString("auth-contexts")
				contexts[context] = token

				viper.Set("auth-contexts", contexts)
			}
		},
	}

	if initGodo {
		if err := cmdConfig.initServices(cmdConfig); err != nil {
			return nil, err
		}
	}

	return cmdConfig, nil
}

// Display displays the output from a command.
func (c *CmdConfig) Display(d displayers.Displayable) error {
	dc := &displayers.Displayer{
		NS:     c.NS,
		Config: c.Doit,
		Item:   d,
		Out:    c.Out,
	}

	return dc.Display()
}

// CmdBuilder builds a new command.
func CmdBuilder(parent *Command, cr CmdRunner, cliText, desc string, out io.Writer, options ...cmdOption) *Command {
	return cmdBuilderWithInit(parent, cr, cliText, desc, out, true, options...)
}

func cmdBuilderWithInit(parent *Command, cr CmdRunner, cliText, desc string, out io.Writer, initCmd bool, options ...cmdOption) *Command {
	cc := &cobra.Command{
		Use:   cliText,
		Short: desc,
		Long:  desc,
		Run: func(cmd *cobra.Command, args []string) {
			c, err := NewCmdConfig(
				cmdNS(cmd),
				doctl.DoitConfig,
				out,
				args,
				initCmd,
			)
			checkErr(err, cmd)

			err = cr(c)
			checkErr(err, cmd)
		},
	}

	c := &Command{Command: cc}

	if parent != nil {
		parent.AddCommand(c)
	}

	for _, co := range options {
		co(c)
	}

	if cols := c.fmtCols; cols != nil {
		formatHelp := fmt.Sprintf("Columns for output in a comma separated list. Possible values: %s",
			strings.Join(cols, ","))
		AddStringFlag(c, doctl.ArgFormat, "", "", formatHelp)
		AddBoolFlag(c, doctl.ArgNoHeader, "", false, "hide headers")
	}

	return c

}

func writeConfig() error {
	f, err := cfgFileWriter()
	if err != nil {
		return err
	}

	defer f.Close()

	b, err := yaml.Marshal(viper.AllSettings())
	if err != nil {
		return errors.New("unable to encode configuration to YAML format")
	}

	_, err = f.Write(b)
	if err != nil {
		return errors.New("unable to write configuration")
	}

	return nil
}

func defaultConfigFileWriter() (io.WriteCloser, error) {
	f, err := os.Create(cfgFile)
	if err != nil {
		return nil, err
	}
	if err := os.Chmod(cfgFile, 0600); err != nil {
		return nil, err
	}

	return f, nil
}

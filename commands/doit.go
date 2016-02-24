package commands

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/bryanl/doit"
	"github.com/bryanl/doit/do"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	// DoitCmd is the base command.
	DoitCmd = &cobra.Command{
		Use:   "doit",
		Short: "doit is a command line interface for the DigitalOcean API.",
	}

	// Token holds the global authorization token.
	Token string

	// Output holds the global output format.
	Output string

	// Verbose toggles verbose output.
	Verbose bool

	requiredColor = color.New(color.Bold, color.FgWhite).SprintfFunc()

	writer = os.Stdout
)

func init() {
	viper.SetConfigType("yaml")

	DoitCmd.PersistentFlags().StringVarP(&Token, "access-token", "t", "", "DigitalOcean API V2 Access Token")
	DoitCmd.PersistentFlags().StringVarP(&Output, "output", "o", "text", "output formt [text|json]")
	DoitCmd.PersistentFlags().BoolVarP(&Verbose, "verbose", "v", false, "verbose output")
}

// LoadConfig loads out configuration.
func LoadConfig() error {
	cf, err := doit.NewConfigFile()
	if err != nil {
		return err
	}

	r, err := cf.Open()
	if err != nil {
		return fmt.Errorf("can't open configuration file: %v", err)
	}

	return viper.ReadConfig(r)
}

// Init initializes the root command.
func Init() *cobra.Command {
	initializeConfig()
	addCommands()

	return DoitCmd
}

// AddCommands adds sub commands to the base command.
func addCommands() {
	DoitCmd.AddCommand(Account())
	DoitCmd.AddCommand(Actions())
	DoitCmd.AddCommand(Auth())
	DoitCmd.AddCommand(Domain())
	DoitCmd.AddCommand(DropletAction())
	DoitCmd.AddCommand(Droplet())
	DoitCmd.AddCommand(FloatingIP())
	DoitCmd.AddCommand(FloatingIPAction())
	DoitCmd.AddCommand(Images())
	DoitCmd.AddCommand(Plugin())
	DoitCmd.AddCommand(Region())
	DoitCmd.AddCommand(Size())
	DoitCmd.AddCommand(SSHKeys())
	DoitCmd.AddCommand(SSH())
	DoitCmd.AddCommand(Version())
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

type flagOpt func(c *command, name, key string)

func requiredOpt() flagOpt {
	return func(c *command, name, key string) {
		c.MarkFlagRequired(key)
		key = requiredKey(key)
		viper.Set(key, true)

		u := c.Flag(name).Usage
		c.Flag(name).Usage = fmt.Sprintf("%s %s", u, requiredColor("(required)"))
	}
}

func shortFlag(f string) flagOpt {
	return func(c *command, name, key string) {
		c.Flag(name).Shorthand = f
	}
}

func requiredKey(key string) string {
	return fmt.Sprintf("%s.required", key)
}

func addStringFlag(cmd *command, name, dflt, desc string, opts ...flagOpt) {
	fn := flagName(cmd, name)
	cmd.Flags().String(name, dflt, desc)
	viper.BindPFlag(fn, cmd.Flags().Lookup(name))

	for _, o := range opts {
		o(cmd, name, fn)
	}
}

func addIntFlag(cmd *command, name string, def int, desc string, opts ...flagOpt) {
	fn := flagName(cmd, name)
	cmd.Flags().Int(name, def, desc)
	viper.BindPFlag(fn, cmd.Flags().Lookup(name))

	for _, o := range opts {
		o(cmd, name, fn)
	}
}

func addBoolFlag(cmd *command, name string, def bool, desc string, opts ...flagOpt) {
	fn := flagName(cmd, name)
	cmd.Flags().Bool(name, def, desc)
	viper.BindPFlag(fn, cmd.Flags().Lookup(name))

	for _, o := range opts {
		o(cmd, name, fn)
	}
}

func addStringSliceFlag(cmd *command, name string, def []string, desc string, opts ...flagOpt) {
	fn := flagName(cmd, name)
	cmd.Flags().StringSlice(name, def, desc)
	viper.BindPFlag(fn, cmd.Flags().Lookup(name))

	for _, o := range opts {
		o(cmd, name, fn)
	}
}

func flagName(cmd *command, name string) string {
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

// cmdRunner runs a command and passes in a cmdConfig.
type cmdRunner func(*cmdConfig) error

type command struct {
	*cobra.Command

	fmtCols []string
}

type cmdConfig struct {
	ns         string
	doitConfig doit.Config
	out        io.Writer
	args       []string

	// services
	ks   do.KeysService
	ss   do.SizesService
	rs   do.RegionsService
	is   do.ImagesService
	ias  do.ImageActionsService
	fis  do.FloatingIPsService
	fias do.FloatingIPActionsService
	ds   do.DropletsService
	das  do.DropletActionsService
	dos  do.DomainsService
	acts do.ActionsService
	as   do.AccountService
}

func newCmdConfig(ns string, dc doit.Config, out io.Writer, args []string) *cmdConfig {
	return &cmdConfig{
		ns:         ns,
		doitConfig: dc,
		out:        out,
		args:       args,
	}
}

func (c *cmdConfig) display(d displayable) error {
	dc := &displayer{
		ns:     c.ns,
		config: c.doitConfig,
		item:   d,
		out:    c.out,
	}

	return dc.Display()
}

func (c *cmdConfig) accountService() do.AccountService {
	return c.as
}

func (c *cmdConfig) actionsService() do.ActionsService {
	return c.acts
}

func (c *cmdConfig) domainsService() do.DomainsService {
	return c.dos
}

func (c *cmdConfig) dropletActionsService() do.DropletActionsService {
	return c.das
}

func (c *cmdConfig) dropletsService() do.DropletsService {
	return c.ds
}

func (c *cmdConfig) floatingIPActionsService() do.FloatingIPActionsService {
	return c.fias
}

func (c *cmdConfig) floatingIPsService() do.FloatingIPsService {
	return c.fis
}

func (c *cmdConfig) imageActionsService() do.ImageActionsService {
	return c.ias
}

func (c *cmdConfig) imagesService() do.ImagesService {
	return c.is
}

func (c *cmdConfig) regionsService() do.RegionsService {
	return c.rs
}

func (c *cmdConfig) sizesService() do.SizesService {
	return c.ss
}

func (c *cmdConfig) keysService() do.KeysService {
	return c.ks
}

func cmdBuilder(parent *cobra.Command, cr cmdRunner, cliText, desc string, out io.Writer, options ...cmdOption) *command {
	cc := &cobra.Command{
		Use:   cliText,
		Short: desc,
		Long:  desc,
		Run: func(cmd *cobra.Command, args []string) {
			c := newCmdConfig(
				cmdNS(cmd),
				doit.DoitConfig,
				out,
				args,
			)

			err := cr(c)
			checkErr(err, cmd)
		},
	}

	if parent != nil {
		parent.AddCommand(cc)
	}

	c := &command{Command: cc}

	for _, co := range options {
		co(c)
	}

	if cols := c.fmtCols; cols != nil {
		formatHelp := fmt.Sprintf("Columns for output in a comma seperated list. Possible values: %s",
			strings.Join(cols, ","))
		addStringFlag(c, doit.ArgFormat, "", formatHelp)
		addBoolFlag(c, doit.ArgNoHeader, false, "hide headers")
	}

	return c
}

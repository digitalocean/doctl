package commands

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	configFile = ".doitcfg"
)

var (
	DoitCmd = &cobra.Command{
		Use: "doit",
	}

	Token, Output string
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

func Execute() {
	InitializeConfig()
	AddCommands()
	DoitCmd.Execute()
}

func AddCommands() {
	DoitCmd.AddCommand(Account())
	DoitCmd.AddCommand(Actions())
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
func InitializeConfig() {
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

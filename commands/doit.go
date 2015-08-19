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

func init() {
	viper.SetConfigType("yaml")
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

// Root creates the root command for doit.
func Root() *cobra.Command {
	rootCmd := &cobra.Command{Use: "doit"}

	rootCmd.PersistentFlags().String("token", "", "DigitalOcean API V2 Token")
	viper.SetEnvPrefix("DIGITALOCEAN")
	viper.BindEnv("token", "ACCESS_TOKEN")
	viper.BindPFlag("token", rootCmd.Flags().Lookup("token"))

	rootCmd.AddCommand(Account())

	return rootCmd
}

func configFilePath() (string, error) {
	usr, err := user.Current()
	if err != nil {
		return "", err
	}

	dir := filepath.Join(usr.HomeDir, configFile)
	return dir, nil
}

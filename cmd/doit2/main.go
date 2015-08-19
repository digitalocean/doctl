package main

import (
	"os"
	"os/user"
	"path/filepath"

	"github.com/Sirupsen/logrus"
	"github.com/bryanl/doit"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	configFile = ".doitcfg"
)

func init() {
	logrus.SetOutput(os.Stderr)
	logrus.SetLevel(logrus.InfoLevel)

	doit.Bail = func(err error, msg string) {
		logrus.WithField("err", err).Fatal(msg)
	}

	viper.SetConfigType("yaml")
}

func main() {
	fp, err := configFilePath()
	if err != nil {
		logrus.WithField("err", err).Fatal("can't find home directory")
	}
	if _, err := os.Stat(fp); err == nil {
		file, err := os.Open(fp)
		if err != nil {
			logrus.WithFields(logrus.Fields{
				"err":  err,
				"path": fp,
			}).Fatal("can't open configuration file")
		}
		viper.ReadConfig(file)
	}

	rootCmd := &cobra.Command{Use: "doit"}

	rootCmd.PersistentFlags().String("token", "", "DigitalOcean API V2 Token")
	viper.SetEnvPrefix("DIGITALOCEAN")
	viper.BindEnv("token", "ACCESS_TOKEN")
	viper.BindPFlag("token", rootCmd.Flags().Lookup("token"))

	rootCmd.AddCommand(account())
	rootCmd.Execute()
}

func account() *cobra.Command {
	cmdAccount := &cobra.Command{
		Use:   "account",
		Short: "account commands",
		Long:  "account is used to access account commands",
	}

	cmdAccountGet := &cobra.Command{
		Use:   "get",
		Short: "account info",
		Long:  "get account details",
		Run: func(cmd *cobra.Command, args []string) {
			doit.NewAccountGet()
		},
	}

	cmdAccount.AddCommand(cmdAccountGet)

	return cmdAccount
}

func configFilePath() (string, error) {
	usr, err := user.Current()
	if err != nil {
		return "", err
	}

	dir := filepath.Join(usr.HomeDir, configFile)
	return dir, nil
}

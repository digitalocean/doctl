package doit

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/Sirupsen/logrus"
	"github.com/digitalocean/godo"
	"github.com/spf13/viper"
	"golang.org/x/oauth2"
)

var (
	// DoitConfig holds the app's current configuration.
	DoitConfig Config = &LiveConfig{}
)

// Config is an interface that represent doit's config.
type Config interface {
	GetGodoClient() *godo.Client
	SSH(user, host string, options []string) Runner
	Set(ns, key string, val interface{})
	GetString(ns, key string) string
	GetBool(ns, key string) bool
	GetInt(ns, key string) int
	GetStringSlice(ns, key string) []string
}

// LiveConfig is an implementation of Config for live values.
type LiveConfig struct{}

var _ Config = &LiveConfig{}

// GetGodoClient returns a GodoClient.
func (c *LiveConfig) GetGodoClient() *godo.Client {
	token := viper.GetString("access-token")
	tokenSource := &TokenSource{AccessToken: token}
	oauthClient := oauth2.NewClient(oauth2.NoContext, tokenSource)
	return godo.NewClient(oauthClient)
}

// SSH creates a ssh connection to a host.
func (c *LiveConfig) SSH(user, host string, options []string) Runner {
	logrus.WithFields(logrus.Fields{
		"user": user,
		"host": host,
	}).Info("ssh")

	sshHost := fmt.Sprintf("%s@%s", user, host)

	args := []string{sshHost}
	for _, o := range options {
		args = append(args, "-o", o)
	}

	cmd := exec.Command("ssh", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd
}

// Set sets a config key.
func (c *LiveConfig) Set(ns, key string, val interface{}) {
	nskey := fmt.Sprintf("%s-%s", ns, key)
	viper.Set(nskey, val)
}

// GetString returns a config value as a string.
func (c *LiveConfig) GetString(ns, key string) string {
	if ns == NSRoot {
		return viper.GetString(key)
	}

	nskey := fmt.Sprintf("%s-%s", ns, key)
	return viper.GetString(nskey)
}

// GetBool returns a config value as a bool.
func (c *LiveConfig) GetBool(ns, key string) bool {
	if ns == NSRoot {
		return viper.GetBool(key)
	}

	nskey := fmt.Sprintf("%s-%s", ns, key)
	return viper.GetBool(nskey)
}

// GetInt returns a config value as an int.
func (c *LiveConfig) GetInt(ns, key string) int {
	if ns == NSRoot {
		return viper.GetInt(key)
	}

	nskey := fmt.Sprintf("%s-%s", ns, key)
	return viper.GetInt(nskey)
}

// GetStringSlice returns a config value as a string slice.
func (c *LiveConfig) GetStringSlice(ns, key string) []string {
	if ns == NSRoot {
		return viper.GetStringSlice(key)
	}

	nskey := fmt.Sprintf("%s-%s", ns, key)
	return viper.GetStringSlice(nskey)
}

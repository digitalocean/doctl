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
	Bail    func(err error, msg string)
	VConfig ViperConfig = &LiveViperConfig{}
)

type ViperConfig interface {
	GetGodoClient() *godo.Client
	SSH(user, host string, options []string) Runner
	Set(ns, key string, val interface{})
	GetString(ns, key string) string
	GetBool(ns, key string) bool
	GetInt(ns, key string) int
	GetStringSlice(ns, key string) []string
}

type LiveViperConfig struct {
}

var _ ViperConfig = &LiveViperConfig{}

func (c *LiveViperConfig) GetGodoClient() *godo.Client {
	token := viper.GetString("token")
	tokenSource := &TokenSource{AccessToken: token}
	oauthClient := oauth2.NewClient(oauth2.NoContext, tokenSource)
	return godo.NewClient(oauthClient)
}

func (c *LiveViperConfig) SSH(user, host string, options []string) Runner {
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

func (c *LiveViperConfig) Set(ns, key string, val interface{}) {
	nskey := fmt.Sprintf("%s-%s", ns, key)
	viper.Set(nskey, val)
}

func (c *LiveViperConfig) GetString(ns, key string) string {
	if ns == NSRoot {
		return viper.GetString(key)
	}

	nskey := fmt.Sprintf("%s-%s", ns, key)
	return viper.GetString(nskey)
}

func (c *LiveViperConfig) GetBool(ns, key string) bool {
	if ns == NSRoot {
		return viper.GetBool(key)
	}

	nskey := fmt.Sprintf("%s-%s", ns, key)
	return viper.GetBool(nskey)
}

func (c *LiveViperConfig) GetInt(ns, key string) int {
	if ns == NSRoot {
		return viper.GetInt(key)
	}

	nskey := fmt.Sprintf("%s-%s", ns, key)
	return viper.GetInt(nskey)
}

func (c *LiveViperConfig) GetStringSlice(ns, key string) []string {
	if ns == NSRoot {
		return viper.GetStringSlice(key)
	}

	nskey := fmt.Sprintf("%s-%s", ns, key)
	return viper.GetStringSlice(nskey)
}

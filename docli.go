package doit

import (
	"fmt"

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
	Set(ns, key string, val interface{})
	GetString(ns, key string) string
	GetBool(ns, key string) bool
	GetInt(ns, key string) int
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

type TestViperConfig struct {
	Client *godo.Client
	v      *viper.Viper
}

func NewTestViperConfig(client *godo.Client) *TestViperConfig {
	return &TestViperConfig{
		Client: client,
		v:      viper.New(),
	}
}

var _ ViperConfig = &TestViperConfig{}

func (c *TestViperConfig) GetGodoClient() *godo.Client {
	return c.Client
}

func (c *TestViperConfig) Set(ns, key string, val interface{}) {
	nskey := fmt.Sprintf("%s-%s", ns, key)
	c.v.Set(nskey, val)
}

func (c *TestViperConfig) GetString(ns, key string) string {
	nskey := fmt.Sprintf("%s-%s", ns, key)
	return c.v.GetString(nskey)
}

func (c *TestViperConfig) GetInt(ns, key string) int {
	nskey := fmt.Sprintf("%s-%s", ns, key)
	return c.v.GetInt(nskey)
}

func (c *TestViperConfig) GetBool(ns, key string) bool {
	nskey := fmt.Sprintf("%s-%s", ns, key)
	return c.v.GetBool(nskey)
}

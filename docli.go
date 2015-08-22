package doit

import (
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
	Set(key string, val interface{})
	GetString(key string) string
	GetInt(key string) int
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

func (c *LiveViperConfig) Set(key string, val interface{}) {
	viper.Set(key, val)
}

func (c *LiveViperConfig) GetString(key string) string {
	return viper.GetString(key)
}

func (c *LiveViperConfig) GetInt(key string) int {
	return viper.GetInt(key)
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

func (c *TestViperConfig) Set(key string, val interface{}) {
	c.v.Set(key, val)
}

func (c *TestViperConfig) GetString(key string) string {
	return c.v.GetString(key)
}

func (c *TestViperConfig) GetInt(key string) int {
	return c.v.GetInt(key)
}

package commands

import (
	"fmt"

	"github.com/Sirupsen/logrus"
	"github.com/bryanl/doit"
	"github.com/digitalocean/godo"
	"github.com/spf13/viper"
)

var (
	testDroplet = godo.Droplet{
		ID: 1,
		Image: &godo.Image{
			ID:           1,
			Name:         "an-image",
			Distribution: "DOOS",
		},
		Name: "a-droplet",
		Networks: &godo.Networks{
			V4: []godo.NetworkV4{
				{IPAddress: "8.8.8.8", Type: "public"},
				{IPAddress: "172.16.1.2", Type: "private"},
			},
		},
		Region: &godo.Region{
			Slug: "test0",
			Name: "test 0",
		},
	}
	testDropletList = []godo.Droplet{testDroplet}
	testKernel      = godo.Kernel{ID: 1}
	testKernelList  = []godo.Kernel{testKernel}
)

type testFn func(c *TestViperConfig)

func withTestClient(client *godo.Client, tFn testFn) {
	ogConfig := doit.VConfig
	defer func() {
		doit.VConfig = ogConfig
	}()

	cfg := NewTestViperConfig(client)
	doit.VConfig = cfg

	tFn(cfg)
}

type TestViperConfig struct {
	Client *godo.Client
	SSHFn  func(user, host string, options []string) doit.Runner
	v      *viper.Viper
}

func NewTestViperConfig(client *godo.Client) *TestViperConfig {
	return &TestViperConfig{
		Client: client,
		SSHFn: func(u, h string, o []string) doit.Runner {
			logrus.WithFields(logrus.Fields{
				"user":    u,
				"host":    h,
				"options": o,
			}).Info("ssh")
			return &doit.MockRunner{}
		},
		v: viper.New(),
	}
}

var _ doit.ViperConfig = &TestViperConfig{}

func (c *TestViperConfig) GetGodoClient() *godo.Client {
	return c.Client
}

func (c *TestViperConfig) SSH(user, host string, options []string) doit.Runner {
	return c.SSHFn(user, host, options)
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

func (c *TestViperConfig) GetStringSlice(ns, key string) []string {
	nskey := fmt.Sprintf("%s-%s", ns, key)
	return c.v.GetStringSlice(nskey)
}

func (c *TestViperConfig) GetBool(ns, key string) bool {
	nskey := fmt.Sprintf("%s-%s", ns, key)
	return c.v.GetBool(nskey)
}

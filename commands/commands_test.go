package commands

import (
	"fmt"
	"sort"
	"testing"

	"github.com/bryanl/doit"
	"github.com/digitalocean/godo"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
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

	testPrivateDroplet = godo.Droplet{
		ID: 1,
		Image: &godo.Image{
			ID:           1,
			Name:         "an-image",
			Distribution: "DOOS",
		},
		Name: "a-droplet",
		Networks: &godo.Networks{
			V4: []godo.NetworkV4{
				{IPAddress: "172.16.1.2", Type: "private"},
			},
		},
		Region: &godo.Region{
			Slug: "test0",
			Name: "test 0",
		},
	}

	testDropletList        = []godo.Droplet{testDroplet}
	testPrivateDropletList = []godo.Droplet{testPrivateDroplet}
	testKernel             = godo.Kernel{ID: 1}
	testKernelList         = []godo.Kernel{testKernel}
	testFloatingIP         = godo.FloatingIP{
		Droplet: &testDroplet,
		Region:  testDroplet.Region,
		IP:      "127.0.0.1",
	}
	testFloatingIPList = []godo.FloatingIP{testFloatingIP}
)

func assertCommandNames(t *testing.T, cmd *cobra.Command, expected ...string) {
	var names []string

	for _, c := range cmd.Commands() {
		names = append(names, c.Name())
	}

	sort.Strings(expected)
	sort.Strings(names)
	assert.Equal(t, expected, names)
}

type testFn func(c *TestConfig)

func withTestClient(client *godo.Client, tFn testFn) {
	ogConfig := doit.DoitConfig
	defer func() {
		doit.DoitConfig = ogConfig
	}()

	cfg := NewTestConfig(client)
	doit.DoitConfig = cfg

	tFn(cfg)
}

type TestConfig struct {
	Client *godo.Client
	SSHFn  func(user, host, keyPath string, port int) doit.Runner
	v      *viper.Viper
}

var _ doit.Config = &TestConfig{}

func NewTestConfig(client *godo.Client) *TestConfig {
	return &TestConfig{
		Client: client,
		SSHFn: func(u, h, kp string, p int) doit.Runner {
			return &doit.MockRunner{}
		},
		v: viper.New(),
	}
}

var _ doit.Config = &TestConfig{}

func (c *TestConfig) GetGodoClient() *godo.Client {
	return c.Client
}

func (c *TestConfig) SSH(user, host, keyPath string, port int) doit.Runner {
	return c.SSHFn(user, host, keyPath, port)
}

func (c *TestConfig) Set(ns, key string, val interface{}) {
	nskey := fmt.Sprintf("%s-%s", ns, key)
	c.v.Set(nskey, val)
}

func (c *TestConfig) GetString(ns, key string) (string, error) {
	nskey := fmt.Sprintf("%s-%s", ns, key)
	return c.v.GetString(nskey), nil
}

func (c *TestConfig) GetInt(ns, key string) (int, error) {
	nskey := fmt.Sprintf("%s-%s", ns, key)
	return c.v.GetInt(nskey), nil
}

func (c *TestConfig) GetStringSlice(ns, key string) ([]string, error) {
	nskey := fmt.Sprintf("%s-%s", ns, key)
	return c.v.GetStringSlice(nskey), nil
}

func (c *TestConfig) GetBool(ns, key string) (bool, error) {
	nskey := fmt.Sprintf("%s-%s", ns, key)
	return c.v.GetBool(nskey), nil
}

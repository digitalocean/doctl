package commands

import (
	"fmt"
	"io/ioutil"
	"sort"
	"testing"

	"github.com/bryanl/doit"
	"github.com/bryanl/doit/do"
	domocks "github.com/bryanl/doit/do/mocks"
	"github.com/bryanl/doit/pkg/runner"
	"github.com/digitalocean/godo"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

var (
	testDroplet = do.Droplet{
		Droplet: &godo.Droplet{
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
		},
	}

	anotherTestDroplet = do.Droplet{
		Droplet: &godo.Droplet{
			ID: 3,
			Image: &godo.Image{
				ID:           1,
				Name:         "an-image",
				Distribution: "DOOS",
			},
			Name: "another-droplet",
			Networks: &godo.Networks{
				V4: []godo.NetworkV4{
					{IPAddress: "8.8.8.9", Type: "public"},
					{IPAddress: "172.16.1.4", Type: "private"},
				},
			},
			Region: &godo.Region{
				Slug: "test0",
				Name: "test 0",
			},
		},
	}

	testPrivateDroplet = do.Droplet{
		Droplet: &godo.Droplet{
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
		},
	}

	testDropletList        = do.Droplets{testDroplet, anotherTestDroplet}
	testPrivateDropletList = do.Droplets{testPrivateDroplet}
	testKernel             = do.Kernel{Kernel: &godo.Kernel{ID: 1}}
	testKernelList         = do.Kernels{testKernel}
	testFloatingIP         = do.FloatingIP{
		FloatingIP: &godo.FloatingIP{
			Droplet: testDroplet.Droplet,
			Region:  testDroplet.Region,
			IP:      "127.0.0.1",
		},
	}
	testFloatingIPList = do.FloatingIPs{testFloatingIP}
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

type testFn func(c *CmdConfig, tm *tcMocks)

type testCmdConfig struct {
	*CmdConfig

	doitConfig *TestConfig
}

type tcMocks struct {
	keys              domocks.KeysService
	sizes             domocks.SizesService
	regions           domocks.RegionsService
	images            domocks.ImagesService
	imageActions      domocks.ImageActionsService
	floatingIPs       domocks.FloatingIPsService
	floatingIPActions domocks.FloatingIPActionsService
	droplets          domocks.DropletsService
	dropletActions    domocks.DropletActionsService
	domains           domocks.DomainsService
	actions           domocks.ActionsService
	account           domocks.AccountService
}

func withTestClient(t *testing.T, tFn testFn) {
	ogConfig := doit.DoitConfig
	defer func() {
		doit.DoitConfig = ogConfig
	}()

	cfg := NewTestConfig()
	doit.DoitConfig = cfg

	tm := &tcMocks{}

	config := &CmdConfig{
		NS:   "test",
		Doit: cfg,
		Out:  ioutil.Discard,

		Keys:              func() do.KeysService { return &tm.keys },
		Sizes:             func() do.SizesService { return &tm.sizes },
		Regions:           func() do.RegionsService { return &tm.regions },
		Images:            func() do.ImagesService { return &tm.images },
		ImageActions:      func() do.ImageActionsService { return &tm.imageActions },
		FloatingIPs:       func() do.FloatingIPsService { return &tm.floatingIPs },
		FloatingIPActions: func() do.FloatingIPActionsService { return &tm.floatingIPActions },
		Droplets:          func() do.DropletsService { return &tm.droplets },
		DropletActions:    func() do.DropletActionsService { return &tm.dropletActions },
		Domains:           func() do.DomainsService { return &tm.domains },
		Actions:           func() do.ActionsService { return &tm.actions },
		Account:           func() do.AccountService { return &tm.account },
	}

	tFn(config, tm)

	assert.True(t, tm.account.AssertExpectations(t))
	assert.True(t, tm.actions.AssertExpectations(t))
	assert.True(t, tm.domains.AssertExpectations(t))
	assert.True(t, tm.dropletActions.AssertExpectations(t))
	assert.True(t, tm.droplets.AssertExpectations(t))
	assert.True(t, tm.floatingIPActions.AssertExpectations(t))
	assert.True(t, tm.floatingIPs.AssertExpectations(t))
	assert.True(t, tm.imageActions.AssertExpectations(t))
	assert.True(t, tm.images.AssertExpectations(t))
	assert.True(t, tm.regions.AssertExpectations(t))
	assert.True(t, tm.sizes.AssertExpectations(t))
	assert.True(t, tm.keys.AssertExpectations(t))
}

type TestConfig struct {
	SSHFn func(user, host, keyPath string, port int) runner.Runner
	v     *viper.Viper
}

var _ doit.Config = &TestConfig{}

func NewTestConfig() *TestConfig {
	return &TestConfig{
		SSHFn: func(u, h, kp string, p int) runner.Runner {
			return &doit.MockRunner{}
		},
		v: viper.New(),
	}
}

var _ doit.Config = &TestConfig{}

func (c *TestConfig) GetGodoClient() *godo.Client {
	return &godo.Client{}
}

func (c *TestConfig) SSH(user, host, keyPath string, port int) runner.Runner {
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

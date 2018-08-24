/*
Copyright 2016 The Doctl Authors All rights reserved.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package commands

import (
	"fmt"
	"io/ioutil"
	"sort"
	"testing"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/do"
	domocks "github.com/digitalocean/doctl/do/mocks"
	"github.com/digitalocean/doctl/pkg/runner"
	"github.com/digitalocean/doctl/pkg/ssh"
	"github.com/digitalocean/godo"
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

	testSnapshot = do.Snapshot{
		Snapshot: &godo.Snapshot{
			ID:      "1",
			Name:    "test-snapshot",
			Regions: []string{"dev0"},
		},
	}
	testSnapshotSecondary = do.Snapshot{
		Snapshot: &godo.Snapshot{
			ID:      "2",
			Name:    "test-snapshot-2",
			Regions: []string{"dev1", "dev2"},
		},
	}

	testSnapshotList = do.Snapshots{testSnapshot, testSnapshotSecondary}
)

func assertCommandNames(t *testing.T, cmd *Command, expected ...string) {
	var names []string

	for _, c := range cmd.Commands() {
		names = append(names, c.Name())
		if c.Name() == "list" {
			assert.Contains(t, c.Aliases, "ls", "Missing 'ls' alias for 'list' command.")
		}
	}

	sort.Strings(expected)
	sort.Strings(names)
	assert.Equal(t, expected, names)
}

type testFn func(c *CmdConfig, tm *tcMocks)

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
	volumes           domocks.VolumesService
	volumeActions     domocks.VolumeActionsService
	actions           domocks.ActionsService
	account           domocks.AccountService
	tags              domocks.TagsService
	snapshots         domocks.SnapshotsService
	certificates      domocks.CertificatesService
	loadBalancers     domocks.LoadBalancersService
	firewalls         domocks.FirewallsService
	cdns              domocks.CDNsService
}

func withTestClient(t *testing.T, tFn testFn) {
	ogConfig := doctl.DoitConfig
	defer func() {
		doctl.DoitConfig = ogConfig
	}()

	cfg := NewTestConfig()
	doctl.DoitConfig = cfg

	tm := &tcMocks{}

	config := &CmdConfig{
		NS:   "test",
		Doit: cfg,
		Out:  ioutil.Discard,

		// can stub this out, since the return is dictated by the mocks.
		initServices: func(c *CmdConfig) error { return nil },

		getContextAccessToken: func() string {
			return viper.GetString(doctl.ArgAccessToken)
		},

		setContextAccessToken: func(token string) {},

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
		Tags:              func() do.TagsService { return &tm.tags },
		Volumes:           func() do.VolumesService { return &tm.volumes },
		VolumeActions:     func() do.VolumeActionsService { return &tm.volumeActions },
		Snapshots:         func() do.SnapshotsService { return &tm.snapshots },
		Certificates:      func() do.CertificatesService { return &tm.certificates },
		LoadBalancers:     func() do.LoadBalancersService { return &tm.loadBalancers },
		Firewalls:         func() do.FirewallsService { return &tm.firewalls },
		CDNs:              func() do.CDNsService { return &tm.cdns },
	}

	tFn(config, tm)

	assert.True(t, tm.account.AssertExpectations(t))
	assert.True(t, tm.actions.AssertExpectations(t))
	assert.True(t, tm.certificates.AssertExpectations(t))
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
	assert.True(t, tm.tags.AssertExpectations(t))
	assert.True(t, tm.volumes.AssertExpectations(t))
	assert.True(t, tm.volumeActions.AssertExpectations(t))
	assert.True(t, tm.snapshots.AssertExpectations(t))
	assert.True(t, tm.loadBalancers.AssertExpectations(t))
	assert.True(t, tm.firewalls.AssertExpectations(t))
	assert.True(t, tm.cdns.AssertExpectations(t))
}

type TestConfig struct {
	SSHFn func(user, host, keyPath string, port int, opts ssh.Options) runner.Runner
	v     *viper.Viper
}

var _ doctl.Config = &TestConfig{}

func NewTestConfig() *TestConfig {
	return &TestConfig{
		SSHFn: func(u, h, kp string, p int, opts ssh.Options) runner.Runner {
			return &doctl.MockRunner{}
		},
		v: viper.New(),
	}
}

var _ doctl.Config = &TestConfig{}

func (c *TestConfig) GetGodoClient(trace bool, accessToken string) (*godo.Client, error) {
	return &godo.Client{}, nil
}

func (c *TestConfig) SSH(user, host, keyPath string, port int, opts ssh.Options) runner.Runner {
	return c.SSHFn(user, host, keyPath, port, opts)
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

/*
Copyright 2018 The Doctl Authors All rights reserved.
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
	"io/ioutil"
	"sort"
	"testing"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/do"
	domocks "github.com/digitalocean/doctl/do/mocks"
	"github.com/digitalocean/godo"
	"github.com/golang/mock/gomock"
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
			assert.NotNil(t, c.Flags().Lookup("format"), "Missing 'format' flag for 'list' command.")
		}
		if c.Name() == "get" {
			assert.NotNil(t, c.Flags().Lookup("format"), "Missing 'format' flag for 'get' command.")
		}
	}

	sort.Strings(expected)
	sort.Strings(names)
	assert.Equal(t, expected, names)
}

type testFn func(c *CmdConfig, tm *tcMocks)

type tcMocks struct {
	account           *domocks.MockAccountService
	actions           *domocks.MockActionsService
	apps              *domocks.MockAppsService
	balance           *domocks.MockBalanceService
	billingHistory    *domocks.MockBillingHistoryService
	databases         *domocks.MockDatabasesService
	dropletActions    *domocks.MockDropletActionsService
	droplets          *domocks.MockDropletsService
	keys              *domocks.MockKeysService
	sizes             *domocks.MockSizesService
	regions           *domocks.MockRegionsService
	images            *domocks.MockImagesService
	imageActions      *domocks.MockImageActionsService
	invoices          *domocks.MockInvoicesService
	floatingIPs       *domocks.MockFloatingIPsService
	floatingIPActions *domocks.MockFloatingIPActionsService
	domains           *domocks.MockDomainsService
	volumes           *domocks.MockVolumesService
	volumeActions     *domocks.MockVolumeActionsService
	tags              *domocks.MockTagsService
	snapshots         *domocks.MockSnapshotsService
	certificates      *domocks.MockCertificatesService
	loadBalancers     *domocks.MockLoadBalancersService
	firewalls         *domocks.MockFirewallsService
	cdns              *domocks.MockCDNsService
	projects          *domocks.MockProjectsService
	kubernetes        *domocks.MockKubernetesService
	registry          *domocks.MockRegistryService
	sshRunner         *domocks.MockRunner
	vpcs              *domocks.MockVPCsService
	oneClick          *domocks.MockOneClickService
	listen            *domocks.MockListenerService
	monitoring        *domocks.MockMonitoringService
}

func withTestClient(t *testing.T, tFn testFn) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tm := &tcMocks{
		account:           domocks.NewMockAccountService(ctrl),
		actions:           domocks.NewMockActionsService(ctrl),
		apps:              domocks.NewMockAppsService(ctrl),
		balance:           domocks.NewMockBalanceService(ctrl),
		billingHistory:    domocks.NewMockBillingHistoryService(ctrl),
		keys:              domocks.NewMockKeysService(ctrl),
		sizes:             domocks.NewMockSizesService(ctrl),
		regions:           domocks.NewMockRegionsService(ctrl),
		images:            domocks.NewMockImagesService(ctrl),
		imageActions:      domocks.NewMockImageActionsService(ctrl),
		invoices:          domocks.NewMockInvoicesService(ctrl),
		floatingIPs:       domocks.NewMockFloatingIPsService(ctrl),
		floatingIPActions: domocks.NewMockFloatingIPActionsService(ctrl),
		droplets:          domocks.NewMockDropletsService(ctrl),
		dropletActions:    domocks.NewMockDropletActionsService(ctrl),
		domains:           domocks.NewMockDomainsService(ctrl),
		tags:              domocks.NewMockTagsService(ctrl),
		volumes:           domocks.NewMockVolumesService(ctrl),
		volumeActions:     domocks.NewMockVolumeActionsService(ctrl),
		snapshots:         domocks.NewMockSnapshotsService(ctrl),
		certificates:      domocks.NewMockCertificatesService(ctrl),
		loadBalancers:     domocks.NewMockLoadBalancersService(ctrl),
		firewalls:         domocks.NewMockFirewallsService(ctrl),
		cdns:              domocks.NewMockCDNsService(ctrl),
		projects:          domocks.NewMockProjectsService(ctrl),
		kubernetes:        domocks.NewMockKubernetesService(ctrl),
		databases:         domocks.NewMockDatabasesService(ctrl),
		registry:          domocks.NewMockRegistryService(ctrl),
		sshRunner:         domocks.NewMockRunner(ctrl),
		vpcs:              domocks.NewMockVPCsService(ctrl),
		oneClick:          domocks.NewMockOneClickService(ctrl),
		listen:            domocks.NewMockListenerService(ctrl),
		monitoring:        domocks.NewMockMonitoringService(ctrl),
	}

	config := &CmdConfig{
		NS:   "test",
		Doit: doctl.NewTestConfig(),
		Out:  ioutil.Discard,

		// can stub this out, since the return is dictated by the mocks.
		initServices: func(c *CmdConfig) error { return nil },

		getContextAccessToken: func() string {
			return viper.GetString(doctl.ArgAccessToken)
		},

		setContextAccessToken: func(token string) {},

		Keys:              func() do.KeysService { return tm.keys },
		Sizes:             func() do.SizesService { return tm.sizes },
		Regions:           func() do.RegionsService { return tm.regions },
		Images:            func() do.ImagesService { return tm.images },
		ImageActions:      func() do.ImageActionsService { return tm.imageActions },
		FloatingIPs:       func() do.FloatingIPsService { return tm.floatingIPs },
		FloatingIPActions: func() do.FloatingIPActionsService { return tm.floatingIPActions },
		Droplets:          func() do.DropletsService { return tm.droplets },
		DropletActions:    func() do.DropletActionsService { return tm.dropletActions },
		Domains:           func() do.DomainsService { return tm.domains },
		Actions:           func() do.ActionsService { return tm.actions },
		Account:           func() do.AccountService { return tm.account },
		Balance:           func() do.BalanceService { return tm.balance },
		BillingHistory:    func() do.BillingHistoryService { return tm.billingHistory },
		Invoices:          func() do.InvoicesService { return tm.invoices },
		Tags:              func() do.TagsService { return tm.tags },
		Volumes:           func() do.VolumesService { return tm.volumes },
		VolumeActions:     func() do.VolumeActionsService { return tm.volumeActions },
		Snapshots:         func() do.SnapshotsService { return tm.snapshots },
		Certificates:      func() do.CertificatesService { return tm.certificates },
		LoadBalancers:     func() do.LoadBalancersService { return tm.loadBalancers },
		Firewalls:         func() do.FirewallsService { return tm.firewalls },
		CDNs:              func() do.CDNsService { return tm.cdns },
		Projects:          func() do.ProjectsService { return tm.projects },
		Kubernetes:        func() do.KubernetesService { return tm.kubernetes },
		Databases:         func() do.DatabasesService { return tm.databases },
		Registry:          func() do.RegistryService { return tm.registry },
		VPCs:              func() do.VPCsService { return tm.vpcs },
		OneClicks:         func() do.OneClickService { return tm.oneClick },
		Apps:              func() do.AppsService { return tm.apps },
		Monitoring:        func() do.MonitoringService { return tm.monitoring },
	}

	tFn(config, tm)
}

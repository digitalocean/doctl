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
	"io"
	"testing"
	"time"

	"github.com/digitalocean/godo"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/do"
	domocks "github.com/digitalocean/doctl/do/mocks"
	"github.com/digitalocean/doctl/internal/apps/builder"
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
	testReservedIP         = do.ReservedIP{
		ReservedIP: &godo.ReservedIP{
			Droplet: testDroplet.Droplet,
			Region:  testDroplet.Region,
			IP:      "127.0.0.1",
		},
	}
	testReservedIPList = do.ReservedIPs{testReservedIP}

	testReservedIPv6 = do.ReservedIPv6{
		ReservedIPV6: &godo.ReservedIPV6{
			Droplet:    testDroplet.Droplet,
			RegionSlug: testDroplet.Region.Slug,
			IP:         "5a11:a:b0a7",
		},
	}
	testReservedIPv6List = do.ReservedIPv6s{testReservedIPv6}

	testBYOIPPrefixCreate = &godo.BYOIPPrefixCreateResp{
		UUID:   "123e4567-e89b-12d3-a456-426614174000",
		Region: "nyc1",
		Status: "active",
	}

	testBYOIPPrefix = do.BYOIPPrefix{
		BYOIPPrefix: &godo.BYOIPPrefix{
			Prefix:        "10.1.1.1/24",
			Region:        "nyc1",
			Status:        "active",
			UUID:          "123e4567-e89b-12d3-a456-426614174000",
			FailureReason: "",
			Validations:   nil,
		},
	}

	testBYOIPPrefixList = do.BYOIPPrefixes{testBYOIPPrefix}

	testBYOIPPrefixGetResources = do.BYOIPPrefixResources{
		{
			BYOIPPrefixResource: &godo.BYOIPPrefixResource{
				ID:         1234,
				BYOIP:      "10.1.1.23",
				Resource:   "do:droplet:b5816aae-437a-4d82-b440-656af71a2ad4",
				Region:     "nyc1",
				AssignedAt: time.Now(),
			},
		},
	}

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

	testDropletBackupPolicy = do.DropletBackupPolicy{
		DropletBackupPolicy: &godo.DropletBackupPolicy{
			DropletID: 123,
			BackupPolicy: &godo.DropletBackupPolicyConfig{
				Plan:                "weekly",
				Weekday:             "MON",
				Hour:                0,
				WindowLengthHours:   4,
				RetentionPeriodDays: 28,
			},
			NextBackupWindow: &godo.BackupWindow{
				Start: &godo.Timestamp{Time: time.Date(2024, time.January, 1, 12, 0, 0, 0, time.UTC)},
				End:   &godo.Timestamp{Time: time.Date(2024, time.February, 1, 12, 0, 0, 0, time.UTC)},
			},
		},
	}

	anotherTestDropletBackupPolicy = do.DropletBackupPolicy{
		DropletBackupPolicy: &godo.DropletBackupPolicy{
			DropletID: 123,
			BackupPolicy: &godo.DropletBackupPolicyConfig{
				Plan:                "daily",
				Hour:                12,
				WindowLengthHours:   4,
				RetentionPeriodDays: 7,
			},
			NextBackupWindow: &godo.BackupWindow{
				Start: &godo.Timestamp{Time: time.Date(2024, time.January, 1, 12, 0, 0, 0, time.UTC)},
				End:   &godo.Timestamp{Time: time.Date(2024, time.February, 1, 12, 0, 0, 0, time.UTC)},
			},
		},
	}

	testDropletBackupPolicies = do.DropletBackupPolicies{testDropletBackupPolicy, anotherTestDropletBackupPolicy}

	testDropletSupportedBackupPolicy = do.DropletSupportedBackupPolicy{
		SupportedBackupPolicy: &godo.SupportedBackupPolicy{
			Name:                 "daily",
			PossibleWindowStarts: []int{0, 4, 8, 12, 16, 20},
			WindowLengthHours:    4,
			RetentionPeriodDays:  7,
			PossibleDays:         []string{},
		},
	}

	anotherTestDropletSupportedBackupPolicy = do.DropletSupportedBackupPolicy{
		SupportedBackupPolicy: &godo.SupportedBackupPolicy{
			Name:                 "weekly",
			PossibleWindowStarts: []int{0, 4, 8, 12, 16, 20},
			WindowLengthHours:    4,
			RetentionPeriodDays:  28,
			PossibleDays:         []string{"SUN", "MON", "TUE", "WED", "THU", "FRI", "SAT"},
		},
	}

	testDropletSupportedBackupPolicies = do.DropletSupportedBackupPolicies{testDropletSupportedBackupPolicy, anotherTestDropletSupportedBackupPolicy}
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

	assert.ElementsMatch(t, expected, names)
}

type testFn func(c *CmdConfig, tm *tcMocks)

type tcMocks struct {
	account               *domocks.MockAccountService
	actions               *domocks.MockActionsService
	apps                  *domocks.MockAppsService
	balance               *domocks.MockBalanceService
	billingHistory        *domocks.MockBillingHistoryService
	databases             *domocks.MockDatabasesService
	dropletActions        *domocks.MockDropletActionsService
	dropletAutoscale      *domocks.MockDropletAutoscaleService
	droplets              *domocks.MockDropletsService
	keys                  *domocks.MockKeysService
	sizes                 *domocks.MockSizesService
	regions               *domocks.MockRegionsService
	images                *domocks.MockImagesService
	imageActions          *domocks.MockImageActionsService
	invoices              *domocks.MockInvoicesService
	reservedIPs           *domocks.MockReservedIPsService
	reservedIPActions     *domocks.MockReservedIPActionsService
	reservedIPv6s         *domocks.MockReservedIPv6sService
	byoipPrefixes         *domocks.MockBYOIPPrefixsService
	domains               *domocks.MockDomainsService
	uptimeChecks          *domocks.MockUptimeChecksService
	volumes               *domocks.MockVolumesService
	volumeActions         *domocks.MockVolumeActionsService
	tags                  *domocks.MockTagsService
	snapshots             *domocks.MockSnapshotsService
	certificates          *domocks.MockCertificatesService
	loadBalancers         *domocks.MockLoadBalancersService
	firewalls             *domocks.MockFirewallsService
	cdns                  *domocks.MockCDNsService
	projects              *domocks.MockProjectsService
	kubernetes            *domocks.MockKubernetesService
	registry              *domocks.MockRegistryService
	registries            *domocks.MockRegistriesService
	sshRunner             *domocks.MockRunner
	vpcs                  *domocks.MockVPCsService
	vpcNatGateways        *domocks.MockVPCNATGatewaysService
	oneClick              *domocks.MockOneClickService
	listen                *domocks.MockListenerService
	terminal              *domocks.MockTerminal
	monitoring            *domocks.MockMonitoringService
	serverless            *domocks.MockServerlessService
	appBuilderFactory     *builder.MockComponentBuilderFactory
	appBuilder            *builder.MockComponentBuilder
	appDockerEngineClient *builder.MockDockerEngineClient
	oauth                 *domocks.MockOAuthService
	partnerAttachments    *domocks.MockPartnerAttachmentsService
	spacesKeys            *domocks.MockSpacesKeysService
	genAI                 *domocks.MockGenAIService
	nfs                   *domocks.MockNfsService
	nfsActions            *domocks.MockNfsActionsService
}

func withTestClient(t *testing.T, tFn testFn) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tm := &tcMocks{
		account:               domocks.NewMockAccountService(ctrl),
		actions:               domocks.NewMockActionsService(ctrl),
		apps:                  domocks.NewMockAppsService(ctrl),
		balance:               domocks.NewMockBalanceService(ctrl),
		billingHistory:        domocks.NewMockBillingHistoryService(ctrl),
		keys:                  domocks.NewMockKeysService(ctrl),
		sizes:                 domocks.NewMockSizesService(ctrl),
		regions:               domocks.NewMockRegionsService(ctrl),
		images:                domocks.NewMockImagesService(ctrl),
		imageActions:          domocks.NewMockImageActionsService(ctrl),
		invoices:              domocks.NewMockInvoicesService(ctrl),
		reservedIPs:           domocks.NewMockReservedIPsService(ctrl),
		reservedIPActions:     domocks.NewMockReservedIPActionsService(ctrl),
		reservedIPv6s:         domocks.NewMockReservedIPv6sService(ctrl),
		byoipPrefixes:         domocks.NewMockBYOIPPrefixsService(ctrl),
		droplets:              domocks.NewMockDropletsService(ctrl),
		dropletActions:        domocks.NewMockDropletActionsService(ctrl),
		dropletAutoscale:      domocks.NewMockDropletAutoscaleService(ctrl),
		domains:               domocks.NewMockDomainsService(ctrl),
		tags:                  domocks.NewMockTagsService(ctrl),
		uptimeChecks:          domocks.NewMockUptimeChecksService(ctrl),
		volumes:               domocks.NewMockVolumesService(ctrl),
		volumeActions:         domocks.NewMockVolumeActionsService(ctrl),
		vpcNatGateways:        domocks.NewMockVPCNATGatewaysService(ctrl),
		snapshots:             domocks.NewMockSnapshotsService(ctrl),
		certificates:          domocks.NewMockCertificatesService(ctrl),
		loadBalancers:         domocks.NewMockLoadBalancersService(ctrl),
		firewalls:             domocks.NewMockFirewallsService(ctrl),
		cdns:                  domocks.NewMockCDNsService(ctrl),
		projects:              domocks.NewMockProjectsService(ctrl),
		kubernetes:            domocks.NewMockKubernetesService(ctrl),
		databases:             domocks.NewMockDatabasesService(ctrl),
		registry:              domocks.NewMockRegistryService(ctrl),
		registries:            domocks.NewMockRegistriesService(ctrl),
		sshRunner:             domocks.NewMockRunner(ctrl),
		vpcs:                  domocks.NewMockVPCsService(ctrl),
		oneClick:              domocks.NewMockOneClickService(ctrl),
		listen:                domocks.NewMockListenerService(ctrl),
		terminal:              domocks.NewMockTerminal(ctrl),
		monitoring:            domocks.NewMockMonitoringService(ctrl),
		serverless:            domocks.NewMockServerlessService(ctrl),
		appBuilderFactory:     builder.NewMockComponentBuilderFactory(ctrl),
		appBuilder:            builder.NewMockComponentBuilder(ctrl),
		appDockerEngineClient: builder.NewMockDockerEngineClient(ctrl),
		oauth:                 domocks.NewMockOAuthService(ctrl),
		partnerAttachments:    domocks.NewMockPartnerAttachmentsService(ctrl),
		spacesKeys:            domocks.NewMockSpacesKeysService(ctrl),
		genAI:                 domocks.NewMockGenAIService(ctrl),
		nfs:                   domocks.NewMockNfsService(ctrl),
		nfsActions:            domocks.NewMockNfsActionsService(ctrl),
	}

	testConfig := doctl.NewTestConfig()
	testConfig.DockerEngineClient = tm.appDockerEngineClient

	config := &CmdConfig{
		NS:   "test",
		Doit: testConfig,
		Out:  io.Discard,

		// can stub this out, since the return is dictated by the mocks.
		initServices: func(c *CmdConfig) error { return nil },

		getContextAccessToken: func() string {
			return viper.GetString(doctl.ArgAccessToken)
		},

		setContextAccessToken: func(token string) {},

		componentBuilderFactory: tm.appBuilderFactory,

		Keys:               func() do.KeysService { return tm.keys },
		Sizes:              func() do.SizesService { return tm.sizes },
		Regions:            func() do.RegionsService { return tm.regions },
		Images:             func() do.ImagesService { return tm.images },
		ImageActions:       func() do.ImageActionsService { return tm.imageActions },
		ReservedIPs:        func() do.ReservedIPsService { return tm.reservedIPs },
		ReservedIPActions:  func() do.ReservedIPActionsService { return tm.reservedIPActions },
		ReservedIPv6s:      func() do.ReservedIPv6sService { return tm.reservedIPv6s },
		BYOIPPrefixes:      func() do.BYOIPPrefixsService { return tm.byoipPrefixes },
		Droplets:           func() do.DropletsService { return tm.droplets },
		DropletActions:     func() do.DropletActionsService { return tm.dropletActions },
		DropletAutoscale:   func() do.DropletAutoscaleService { return tm.dropletAutoscale },
		Domains:            func() do.DomainsService { return tm.domains },
		Actions:            func() do.ActionsService { return tm.actions },
		Account:            func() do.AccountService { return tm.account },
		Balance:            func() do.BalanceService { return tm.balance },
		BillingHistory:     func() do.BillingHistoryService { return tm.billingHistory },
		Invoices:           func() do.InvoicesService { return tm.invoices },
		Tags:               func() do.TagsService { return tm.tags },
		UptimeChecks:       func() do.UptimeChecksService { return tm.uptimeChecks },
		Volumes:            func() do.VolumesService { return tm.volumes },
		VolumeActions:      func() do.VolumeActionsService { return tm.volumeActions },
		VPCNATGateways:     func() do.VPCNATGatewaysService { return tm.vpcNatGateways },
		Snapshots:          func() do.SnapshotsService { return tm.snapshots },
		Certificates:       func() do.CertificatesService { return tm.certificates },
		LoadBalancers:      func() do.LoadBalancersService { return tm.loadBalancers },
		Firewalls:          func() do.FirewallsService { return tm.firewalls },
		CDNs:               func() do.CDNsService { return tm.cdns },
		Projects:           func() do.ProjectsService { return tm.projects },
		Kubernetes:         func() do.KubernetesService { return tm.kubernetes },
		Databases:          func() do.DatabasesService { return tm.databases },
		Registry:           func() do.RegistryService { return tm.registry },
		Registries:         func() do.RegistriesService { return tm.registries },
		VPCs:               func() do.VPCsService { return tm.vpcs },
		OneClicks:          func() do.OneClickService { return tm.oneClick },
		Apps:               func() do.AppsService { return tm.apps },
		Monitoring:         func() do.MonitoringService { return tm.monitoring },
		Serverless:         func() do.ServerlessService { return tm.serverless },
		OAuth:              func() do.OAuthService { return tm.oauth },
		PartnerAttachments: func() do.PartnerAttachmentsService { return tm.partnerAttachments },
		SpacesKeys:         func() do.SpacesKeysService { return tm.spacesKeys },
		GenAI:              func() do.GenAIService { return tm.genAI },
		Nfs:                func() do.NfsService { return tm.nfs },
		NfsActions:         func() do.NfsActionsService { return tm.nfsActions },
	}

	tFn(config, tm)
}

func assertCommandAliases(t *testing.T, cmd *cobra.Command) {
	for _, c := range cmd.Commands() {
		if c.Name() == "list" {
			assert.Contains(t, c.Aliases, "ls", "Missing 'ls' alias for 'list' command.")
		}
		if c.Name() == "delete" {
			assert.Contains(t, c.Aliases, "rm", "Missing 'rm' alias for 'delete' command.")
		}
	}
}

func recurseCommand(t *testing.T, cmd *cobra.Command) {
	t.Run(cmd.Name(), func(t *testing.T) {
		assertCommandAliases(t, cmd)
	})
	for _, c := range cmd.Commands() {
		recurseCommand(t, c)
	}
}

func TestCommandAliases(t *testing.T) {
	for _, cmd := range DoitCmd.Commands() {
		recurseCommand(t, cmd)
	}
}

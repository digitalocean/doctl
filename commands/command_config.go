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
	"fmt"
	"io"

	"github.com/spf13/viper"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/commands/displayers"
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/doctl/internal/apps/builder"
)

// CmdConfig is a command configuration.
type CmdConfig struct {
	NS   string
	Doit doctl.Config
	Out  io.Writer
	Args []string

	initServices            func(*CmdConfig) error
	getContextAccessToken   func() string
	setContextAccessToken   func(string)
	removeContext           func(string) error
	componentBuilderFactory builder.ComponentBuilderFactory

	// services
	Keys              func() do.KeysService
	Sizes             func() do.SizesService
	Regions           func() do.RegionsService
	Images            func() do.ImagesService
	ImageActions      func() do.ImageActionsService
	LoadBalancers     func() do.LoadBalancersService
	ReservedIPs       func() do.ReservedIPsService
	ReservedIPActions func() do.ReservedIPActionsService
	ReservedIPv6s     func() do.ReservedIPv6sService
	BYOIPPrefixes     func() do.BYOIPPrefixsService

	Droplets           func() do.DropletsService
	DropletActions     func() do.DropletActionsService
	DropletAutoscale   func() do.DropletAutoscaleService
	Domains            func() do.DomainsService
	VPCNATGateways     func() do.VPCNATGatewaysService
	Actions            func() do.ActionsService
	Account            func() do.AccountService
	Balance            func() do.BalanceService
	BillingHistory     func() do.BillingHistoryService
	Invoices           func() do.InvoicesService
	Tags               func() do.TagsService
	UptimeChecks       func() do.UptimeChecksService
	Volumes            func() do.VolumesService
	VolumeActions      func() do.VolumeActionsService
	Snapshots          func() do.SnapshotsService
	Certificates       func() do.CertificatesService
	Firewalls          func() do.FirewallsService
	CDNs               func() do.CDNsService
	Projects           func() do.ProjectsService
	Kubernetes         func() do.KubernetesService
	Databases          func() do.DatabasesService
	Registry           func() do.RegistryService
	VPCs               func() do.VPCsService
	OneClicks          func() do.OneClickService
	Apps               func() do.AppsService
	Monitoring         func() do.MonitoringService
	Serverless         func() do.ServerlessService
	OAuth              func() do.OAuthService
	PartnerAttachments func() do.PartnerAttachmentsService
	SpacesKeys         func() do.SpacesKeysService
	GenAI              func() do.GenAIService
}

// NewCmdConfig creates an instance of a CmdConfig.
func NewCmdConfig(ns string, dc doctl.Config, out io.Writer, args []string, initGodo bool) (*CmdConfig, error) {

	cmdConfig := &CmdConfig{
		NS:   ns,
		Doit: dc,
		Out:  out,
		Args: args,

		initServices: func(c *CmdConfig) error {
			accessToken := c.getContextAccessToken()
			godoClient, err := c.Doit.GetGodoClient(Trace, true, accessToken)
			if err != nil {
				return fmt.Errorf("Unable to initialize DigitalOcean API client: %s", err)
			}

			c.Keys = func() do.KeysService { return do.NewKeysService(godoClient) }
			c.Sizes = func() do.SizesService { return do.NewSizesService(godoClient) }
			c.Regions = func() do.RegionsService { return do.NewRegionsService(godoClient) }
			c.Images = func() do.ImagesService { return do.NewImagesService(godoClient) }
			c.ImageActions = func() do.ImageActionsService { return do.NewImageActionsService(godoClient) }
			c.ReservedIPs = func() do.ReservedIPsService { return do.NewReservedIPsService(godoClient) }
			c.ReservedIPActions = func() do.ReservedIPActionsService { return do.NewReservedIPActionsService(godoClient) }
			c.ReservedIPv6s = func() do.ReservedIPv6sService { return do.NewReservedIPv6sService(godoClient) }
			c.BYOIPPrefixes = func() do.BYOIPPrefixsService { return do.NewBYOIPPrefixService(godoClient) }
			c.Droplets = func() do.DropletsService { return do.NewDropletsService(godoClient) }
			c.DropletActions = func() do.DropletActionsService { return do.NewDropletActionsService(godoClient) }
			c.DropletAutoscale = func() do.DropletAutoscaleService { return do.NewDropletAutoscaleService(godoClient) }
			c.Domains = func() do.DomainsService { return do.NewDomainsService(godoClient) }
			c.VPCNATGateways = func() do.VPCNATGatewaysService { return do.NewVPCNATGatewaysService(godoClient) }
			c.Actions = func() do.ActionsService { return do.NewActionsService(godoClient) }
			c.Account = func() do.AccountService { return do.NewAccountService(godoClient) }
			c.Balance = func() do.BalanceService { return do.NewBalanceService(godoClient) }
			c.BillingHistory = func() do.BillingHistoryService { return do.NewBillingHistoryService(godoClient) }
			c.Invoices = func() do.InvoicesService { return do.NewInvoicesService(godoClient) }
			c.Tags = func() do.TagsService { return do.NewTagsService(godoClient) }
			c.UptimeChecks = func() do.UptimeChecksService { return do.NewUptimeChecksService(godoClient) }
			c.Volumes = func() do.VolumesService { return do.NewVolumesService(godoClient) }
			c.VolumeActions = func() do.VolumeActionsService { return do.NewVolumeActionsService(godoClient) }
			c.Snapshots = func() do.SnapshotsService { return do.NewSnapshotsService(godoClient) }
			c.Certificates = func() do.CertificatesService { return do.NewCertificatesService(godoClient) }
			c.LoadBalancers = func() do.LoadBalancersService { return do.NewLoadBalancersService(godoClient) }
			c.Firewalls = func() do.FirewallsService { return do.NewFirewallsService(godoClient) }
			c.CDNs = func() do.CDNsService { return do.NewCDNsService(godoClient) }
			c.Projects = func() do.ProjectsService { return do.NewProjectsService(godoClient) }
			c.Kubernetes = func() do.KubernetesService { return do.NewKubernetesService(godoClient) }
			c.Databases = func() do.DatabasesService { return do.NewDatabasesService(godoClient) }
			c.Registry = func() do.RegistryService { return do.NewRegistryService(godoClient) }
			c.VPCs = func() do.VPCsService { return do.NewVPCsService(godoClient) }
			c.OneClicks = func() do.OneClickService { return do.NewOneClickService(godoClient) }
			c.Apps = func() do.AppsService { return do.NewAppsService(godoClient) }
			c.Monitoring = func() do.MonitoringService { return do.NewMonitoringService(godoClient) }
			c.Serverless = func() do.ServerlessService {
				return do.NewServerlessService(godoClient, getServerlessDirectory(), accessToken)
			}
			c.OAuth = func() do.OAuthService { return do.NewOAuthService(godoClient) }
			c.PartnerAttachments = func() do.PartnerAttachmentsService {
				return do.NewPartnerAttachmentsService(godoClient)
			}
			c.SpacesKeys = func() do.SpacesKeysService { return do.NewSpacesKeysService(godoClient) }
			c.GenAI = func() do.GenAIService { return do.NewGenAIService(godoClient) }
			return nil
		},

		getContextAccessToken: func() string {
			context := Context
			if context == "" {
				context = viper.GetString("context")
			}
			token := ""

			switch context {
			case doctl.ArgDefaultContext:
				token = viper.GetString(doctl.ArgAccessToken)
			default:
				contexts := viper.GetStringMapString("auth-contexts")

				token = contexts[context]
			}

			return token
		},

		setContextAccessToken: func(token string) {
			context := Context
			if context == "" {
				context = viper.GetString("context")
			}

			switch context {
			case doctl.ArgDefaultContext:
				viper.Set(doctl.ArgAccessToken, token)
			default:
				contexts := viper.GetStringMapString("auth-contexts")
				contexts[context] = token

				viper.Set("auth-contexts", contexts)
			}
		},

		removeContext: func(context string) error {
			if context == "default" {
				viper.Set("access-token", "")
				return nil
			}

			contexts := viper.GetStringMapString("auth-contexts")

			_, ok := contexts[context]

			if !ok {
				return fmt.Errorf("Context not found")
			}

			delete(contexts, context)

			viper.Set("auth-contexts", contexts)

			return nil
		},

		componentBuilderFactory: &builder.DefaultComponentBuilderFactory{},
	}

	if initGodo {
		if err := cmdConfig.initServices(cmdConfig); err != nil {
			return nil, err
		}
	}

	return cmdConfig, nil
}

// CmdRunner runs a command and passes in a cmdConfig.
type CmdRunner func(*CmdConfig) error

// Display displays the output from a command.
func (c *CmdConfig) Display(d displayers.Displayable) error {
	dc := &displayers.Displayer{
		Item: d,
		Out:  c.Out,
	}

	columnList, err := c.Doit.GetString(c.NS, doctl.ArgFormat)
	if err != nil {
		return err
	}

	withHeaders, err := c.Doit.GetBool(c.NS, doctl.ArgNoHeader)
	if err != nil {
		return err
	}

	dc.NoHeaders = withHeaders
	dc.ColumnList = columnList
	dc.OutputType = Output

	return dc.Display()
}

// An urner implements the URN method, which returns a valid uniform resource
// name.
type urner interface {
	URN() string
}

// moveToProject moves the given resource to the project with the given
// project UUID.
func (c *CmdConfig) moveToProject(projectUUID string, u urner) error {
	if projectUUID == "" {
		return nil
	}
	_, err := c.Projects().AssignResources(projectUUID, []string{u.URN()})
	return err
}

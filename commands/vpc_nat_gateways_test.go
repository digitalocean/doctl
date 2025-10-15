package commands

import (
	"testing"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"
)

var (
	testVPCNATGateways = []*godo.VPCNATGateway{
		{
			ID:     "51154959-e07b-4093-98fb-828590ecc76d",
			Name:   "test-vpc-nat-gateway-01",
			Type:   "PUBLIC",
			State:  "ACTIVE",
			Region: "nyc3",
			Size:   1,
			VPCs: []*godo.IngressVPC{
				{
					VpcUUID:   "05790d02-c7e0-47d6-a917-5b4cf68cf5b7",
					GatewayIP: "10.110.0.22",
				},
			},
			Egresses: &godo.Egresses{
				PublicGateways: []*godo.PublicGateway{
					{
						IPv4: "77.38.26.185",
					},
				},
			},
			UDPTimeoutSeconds:  30,
			ICMPTimeoutSeconds: 30,
			TCPTimeoutSeconds:  30,
			ProjectID:          "0b0f3f3c-1d2e-4e5f-8f3c-0b0f3f3c1d2e",
		},
		{
			ID:     "4d99fb28-b33d-4791-aff5-bf30f8f4f917",
			Name:   "test-vpc-nat-gateway-02",
			Type:   "PUBLIC",
			State:  "ACTIVE",
			Region: "nyc3",
			Size:   1,
			VPCs: []*godo.IngressVPC{
				{
					VpcUUID:   "05790d02-c7e0-47d6-a917-5b4cf68cf5b7",
					GatewayIP: "10.110.0.23",
				},
			},
			Egresses: &godo.Egresses{
				PublicGateways: []*godo.PublicGateway{
					{
						IPv4: "151.123.18.248",
					},
				},
			},
			UDPTimeoutSeconds:  30,
			ICMPTimeoutSeconds: 30,
			TCPTimeoutSeconds:  30,
			ProjectID:          "0b0f3f3c-1d2e-4e5f-8f3c-0b0f3f3c1d2e",
		},
	}
)

func TestVPCNATGatewayCommand(t *testing.T) {
	cmd := VPCNATGateway()
	assert.NotNil(t, cmd)
	assertCommandNames(t, cmd, "create", "update", "get", "list", "delete")
}

func TestVPCNATGatewayCreate(t *testing.T) {
	withTestClient(t, func(c *CmdConfig, tm *tcMocks) {
		createReq := godo.VPCNATGatewayRequest{
			Name:   "test-vpc-nat-gateway-01",
			Type:   "PUBLIC",
			Region: "nyc3",
			Size:   1,
			VPCs: []*godo.IngressVPC{
				{
					VpcUUID: "05790d02-c7e0-47d6-a917-5b4cf68cf5b7",
				},
			},
			UDPTimeoutSeconds:  30,
			ICMPTimeoutSeconds: 30,
			TCPTimeoutSeconds:  30,
			ProjectID:          "0b0f3f3c-1d2e-4e5f-8f3c-0b0f3f3c1d2e",
		}

		tm.vpcNatGateways.EXPECT().Create(&createReq).Return(testVPCNATGateways[0], nil)

		c.Doit.Set(c.NS, doctl.ArgVPCNATGatewayName, "test-vpc-nat-gateway-01")
		c.Doit.Set(c.NS, doctl.ArgVPCNATGatewayType, "PUBLIC")
		c.Doit.Set(c.NS, doctl.ArgVPCNATGatewayRegion, "nyc3")
		c.Doit.Set(c.NS, doctl.ArgVPCNATGatewaySize, "1")
		c.Doit.Set(c.NS, doctl.ArgVPCNATGatewayVPCs, "05790d02-c7e0-47d6-a917-5b4cf68cf5b7")
		c.Doit.Set(c.NS, doctl.ArgVPCNATGatewayUDPTimeout, "30")
		c.Doit.Set(c.NS, doctl.ArgVPCNATGatewayICMPTimeout, "30")
		c.Doit.Set(c.NS, doctl.ArgVPCNATGatewayTCPTimeout, "30")
		c.Doit.Set(c.NS, doctl.ArgProjectID, "0b0f3f3c-1d2e-4e5f-8f3c-0b0f3f3c1d2e")

		err := RunVPCNATGatewayCreate(c)
		assert.NoError(t, err)
	})
}

func TestVPCNATGatewayUpdate(t *testing.T) {
	withTestClient(t, func(c *CmdConfig, tm *tcMocks) {
		gatewayID := "51154959-e07b-4093-98fb-828590ecc76d"
		updateReq := godo.VPCNATGatewayRequest{
			Name:               "test-vpc-nat-gateway-01-renamed", // update name
			Type:               "PUBLIC",
			Region:             "nyc3",
			Size:               1,
			UDPTimeoutSeconds:  50, // update timeouts
			ICMPTimeoutSeconds: 50, // update timeouts
			TCPTimeoutSeconds:  50, // update timeouts
		}

		tm.vpcNatGateways.EXPECT().Update(gatewayID, &updateReq).Return(testVPCNATGateways[0], nil)
		c.Args = append(c.Args, gatewayID)

		c.Doit.Set(c.NS, doctl.ArgVPCNATGatewayName, "test-vpc-nat-gateway-01-renamed")
		c.Doit.Set(c.NS, doctl.ArgVPCNATGatewayType, "PUBLIC")
		c.Doit.Set(c.NS, doctl.ArgVPCNATGatewayRegion, "nyc3")
		c.Doit.Set(c.NS, doctl.ArgVPCNATGatewaySize, "1")
		c.Doit.Set(c.NS, doctl.ArgVPCNATGatewayUDPTimeout, "50")
		c.Doit.Set(c.NS, doctl.ArgVPCNATGatewayICMPTimeout, "50")
		c.Doit.Set(c.NS, doctl.ArgVPCNATGatewayTCPTimeout, "50")

		err := RunVPCNATGatewayUpdate(c)
		assert.NoError(t, err)
	})
}

func TestVPCNATGatewayGet(t *testing.T) {
	withTestClient(t, func(c *CmdConfig, tm *tcMocks) {
		gatewayID := "51154959-e07b-4093-98fb-828590ecc76d"
		tm.vpcNatGateways.EXPECT().Get(gatewayID).Return(testVPCNATGateways[0], nil)
		c.Args = append(c.Args, gatewayID)

		err := RunVPCNATGatewayGet(c)
		assert.NoError(t, err)
	})
}

func TestVPCNATGatewayList(t *testing.T) {
	withTestClient(t, func(c *CmdConfig, tm *tcMocks) {
		tm.vpcNatGateways.EXPECT().List().Return(testVPCNATGateways, nil)

		err := RunVPCNATGatewayList(c)
		assert.NoError(t, err)
	})
}

func TestVPCNATGatewayDelete(t *testing.T) {
	withTestClient(t, func(c *CmdConfig, tm *tcMocks) {
		gatewayID := "51154959-e07b-4093-98fb-828590ecc76d"
		tm.vpcNatGateways.EXPECT().Delete(gatewayID).Return(nil)
		c.Args = append(c.Args, gatewayID)
		c.Doit.Set(c.NS, doctl.ArgForce, "true")

		err := RunVPCNATGatewayDelete(c)
		assert.NoError(t, err)
	})
}

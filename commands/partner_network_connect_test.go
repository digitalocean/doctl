package commands

import (
	"strings"
	"testing"
	"time"

	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/do"
)

var (
	testPartnerNetworkConnect = do.PartnerNetworkConnect{
		PartnerNetworkConnect: &godo.PartnerAttachment{
			ID:                        "test-id",
			Name:                      "doctl-pia",
			State:                     "active",
			ConnectionBandwidthInMbps: 50,
			Region:                    "stage2",
			NaaSProvider:              "MEGAPORT",
			VPCIDs:                    []string{"d35e5cb7-7957-4643-8e3a-1ab4eb3a494c"},
			CreatedAt:                 time.Date(2025, 1, 30, 0, 0, 0, 0, time.UTC),
		},
	}

	testPartnerNCList = do.PartnerNetworkConnects{
		testPartnerNetworkConnect,
	}

	testPartnerNCRoute = do.PartnerNetworkConnectRoute{
		RemoteRoute: &godo.RemoteRoute{
			ID:   "test-route-id",
			Cidr: "10.10.0.0/24",
		},
	}

	testPartnerNCRouteList = do.PartnerNetworkConnectRoutes{
		testPartnerNCRoute,
	}

	testRegenerateServiceKey = do.PartnerNetworkConnectRegenerateServiceKey{
		RegenerateServiceKey: &godo.RegenerateServiceKey{},
	}

	testBGPAuthKey = do.PartnerNetworkConnectBGPAuthKey{
		BgpAuthKey: &godo.BgpAuthKey{
			Value: "test-bgp-auth-key",
		},
	}

	testServiceKey = do.PartnerNetworkConnectServiceKey{
		ServiceKey: &godo.ServiceKey{
			Value:     "test-service-key",
			State:     "active",
			CreatedAt: time.Date(2025, 1, 30, 0, 0, 0, 0, time.UTC),
		},
	}
)

func TestPartnerNetworkConnectsCommand(t *testing.T) {
	cmd := PartnerNetworkConnects()
	assert.NotNil(t, cmd)

	assertCommandNames(t, cmd, "create", "get", "list", "delete", "update", "list-routes", "regenerate-service-key", "get-bgp-auth-key", "get-service-key")
}

func TestPartnerNCCreate(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Doit.Set(config.NS, doctl.ArgPartnerNCType, "partner")
		config.Doit.Set(config.NS, doctl.ArgPartnerNCName, "doctl-pia")
		config.Doit.Set(config.NS, doctl.ArgPartnerNCBandwidthInMbps, 50)
		config.Doit.Set(config.NS, doctl.ArgPartnerNCRegion, "stage2")
		config.Doit.Set(config.NS, doctl.ArgPartnerNCNaaSProvider, "MEGAPORT")
		config.Doit.Set(config.NS, doctl.ArgPartnerNCVPCIDs, []string{"d35e5cb7-7957-4643-8e3a-1ab4eb3a494c"})
		config.Doit.Set(config.NS, doctl.ArgPartnerNCBGPLocalASN, 65001)
		config.Doit.Set(config.NS, doctl.ArgPartnerNCBGPLocalRouterIP, "192.168.1.1")
		config.Doit.Set(config.NS, doctl.ArgPartnerNCBGPPeerASN, 65002)
		config.Doit.Set(config.NS, doctl.ArgPartnerNCBGPPeerRouterIP, "192.168.1.2")

		expectedRequest := &godo.PartnerNetworkConnectCreateRequest{
			Name:                      "doctl-pia",
			ConnectionBandwidthInMbps: 50,
			Region:                    "stage2",
			NaaSProvider:              "MEGAPORT",
			VPCIDs:                    []string{"d35e5cb7-7957-4643-8e3a-1ab4eb3a494c"},
		}

		tm.partnerNetworkConnects.EXPECT().Create(expectedRequest).Return(&testPartnerNetworkConnect, nil)

		err := RunPartnerNCCreate(config)
		assert.NoError(t, err)
	})
}

func TestPartnerNCCreateUnsupportedType(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Doit.Set(config.NS, doctl.ArgPartnerNCType, "unsupported")
		err := RunPartnerNCCreate(config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported attachment type")
	})
}

func TestPartnerNCGet(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Doit.Set(config.NS, doctl.ArgPartnerNCType, "partner")

		pncID := "e819b321-a9a1-4078-b437-8e6b8bf13530"
		tm.partnerNetworkConnects.EXPECT().GetPartnerNetworkConnect(pncID).Return(&testPartnerNetworkConnect, nil)

		config.Args = append(config.Args, pncID)

		err := RunPartnerNCGet(config)
		assert.NoError(t, err)
	})
}

func TestPartnerNCGetNoID(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Doit.Set(config.NS, doctl.ArgPartnerNCType, "partner")

		err := RunPartnerNCGet(config)
		assert.Error(t, err)
	})
}

func TestPartnerNCList(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Doit.Set(config.NS, doctl.ArgPartnerNCType, "partner")

		tm.partnerNetworkConnects.EXPECT().ListPartnerNetworkConnects().Return(testPartnerNCList, nil)

		err := RunPartnerNCList(config)
		assert.NoError(t, err)
	})
}

func TestPartnerNCDelete(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Doit.Set(config.NS, doctl.ArgPartnerNCType, "partner")

		iaID := "e819b321-a9a1-4078-b437-8e6b8bf13530"
		tm.partnerNetworkConnects.EXPECT().DeletePartnerNetworkConnect(iaID).Return(nil)

		config.Args = append(config.Args, iaID)
		config.Doit.Set(config.NS, doctl.ArgForce, true)

		err := RunPartnerNCDelete(config)
		assert.NoError(t, err)
	})
}

func TestRunPartnerNCUpdate(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Doit.Set(config.NS, doctl.ArgPartnerNCType, "partner")

		iaID := "ia-uuid1"
		iaName := "ia-name"
		vpcIDs := "f81d4fae-7dec-11d0-a765-00a0c91e6bf6,3f900b61-30d7-40d8-9711-8c5d6264b268"
		r := godo.PartnerNetworkConnectUpdateRequest{Name: iaName, VPCIDs: strings.Split(vpcIDs, ",")}
		tm.partnerNetworkConnects.EXPECT().UpdatePartnerNetworkConnect(iaID, &r).Return(&testPartnerNetworkConnect, nil)

		config.Args = append(config.Args, iaID)
		config.Doit.Set(config.NS, doctl.ArgPartnerNCName, iaName)
		config.Doit.Set(config.NS, doctl.ArgPartnerNCVPCIDs, vpcIDs)

		err := RunPartnerNCUpdate(config)
		assert.NoError(t, err)
	})
}

func TestRunPartnerNCUpdateNoID(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Doit.Set(config.NS, doctl.ArgPartnerNCType, "partner")

		err := RunPartnerNCUpdate(config)
		assert.Error(t, err)
	})
}

func TestPartnerNCRouteList(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Doit.Set(config.NS, doctl.ArgPartnerNCType, "partner")

		iaID := "ia-uuid1"
		config.Args = append(config.Args, iaID)
		tm.partnerNetworkConnects.EXPECT().ListPartnerNetworkConnectRoutes(iaID).Return(testPartnerNCRouteList, nil)

		err := RunPartnerNCRouteList(config)
		assert.NoError(t, err)
	})
}

func TestPartnerNCRegenerateServiceKey(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Doit.Set(config.NS, doctl.ArgPartnerNCType, "partner")

		iaID := "e819b321-a9a1-4078-b437-8e6b8bf13530"
		tm.partnerNetworkConnects.EXPECT().RegenerateServiceKey(iaID).Return(&testRegenerateServiceKey, nil)

		config.Args = append(config.Args, iaID)

		err := RunPartnerNCRegenerateServiceKey(config)
		assert.NoError(t, err)
	})
}

func TestGetPartnerNCBGPAuthKey(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Doit.Set(config.NS, doctl.ArgPartnerNCType, "partner")

		iaID := "e819b321-a9a1-4078-b437-8e6b8bf13530"
		tm.partnerNetworkConnects.EXPECT().GetBGPAuthKey(iaID).Return(&testBGPAuthKey, nil)

		config.Args = append(config.Args, iaID)

		err := RunGetPartnerNCBGPAuthKey(config)
		assert.NoError(t, err)
	})
}

func TestGetPartnerNCServiceKey(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Doit.Set(config.NS, doctl.ArgPartnerNCType, "partner")

		iaID := "e819b321-a9a1-4078-b437-8e6b8bf13530"
		tm.partnerNetworkConnects.EXPECT().GetServiceKey(iaID).Return(&testServiceKey, nil)

		config.Args = append(config.Args, iaID)

		err := RunGetPartnerNCServiceKey(config)
		assert.NoError(t, err)
	})
}

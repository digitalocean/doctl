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
	testPartnerAttachment = do.PartnerNetworkConnect{
		PartnerNetworkConnect: &godo.PartnerNetworkConnect{
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

	testPartnerIAList = do.PartnerNetworkConnects{
		testPartnerAttachment,
	}

	testPartnerAttachmentRoute = do.PartnerAttachmentRoute{
		RemoteRoute: &godo.RemoteRoute{
			ID:   "test-route-id",
			Cidr: "10.10.0.0/24",
		},
	}

	testPartnerIARouteList = do.PartnerAttachmentRoutes{
		testPartnerAttachmentRoute,
	}

	testRegenerateServiceKey = do.PartnerAttachmentRegenerateServiceKey{
		RegenerateServiceKey: &godo.RegenerateServiceKey{},
	}

	testBGPAuthKey = do.PartnerAttachmentBGPAuthKey{
		BgpAuthKey: &godo.BgpAuthKey{
			Value: "test-bgp-auth-key",
		},
	}

	testServiceKey = do.PartnerAttachmentServiceKey{
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

func TestPartnerAttachmentCreate(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Doit.Set(config.NS, doctl.ArgPartnerAttachmentType, "partner")
		config.Doit.Set(config.NS, doctl.ArgPartnerAttachmentName, "doctl-pia")
		config.Doit.Set(config.NS, doctl.ArgPartnerAttachmentConnectionBandwidthInMbps, 50)
		config.Doit.Set(config.NS, doctl.ArgPartnerAttachmentRegion, "stage2")
		config.Doit.Set(config.NS, doctl.ArgPartnerAttachmentNaaSProvider, "MEGAPORT")
		config.Doit.Set(config.NS, doctl.ArgPartnerAttachmentVPCIDs, []string{"d35e5cb7-7957-4643-8e3a-1ab4eb3a494c"})
		config.Doit.Set(config.NS, doctl.ArgPartnerAttachmentBGPLocalASN, 65001)
		config.Doit.Set(config.NS, doctl.ArgPartnerAttachmentBGPLocalRouterIP, "192.168.1.1")
		config.Doit.Set(config.NS, doctl.ArgPartnerAttachmentBGPPeerASN, 65002)
		config.Doit.Set(config.NS, doctl.ArgPartnerAttachmentBGPPeerRouterIP, "192.168.1.2")

		expectedRequest := &godo.PartnerNetworkConnectCreateRequest{
			Name:                      "doctl-pia",
			ConnectionBandwidthInMbps: 50,
			Region:                    "stage2",
			NaaSProvider:              "MEGAPORT",
			VPCIDs:                    []string{"d35e5cb7-7957-4643-8e3a-1ab4eb3a494c"},
		}

		tm.partnerNetworkConnects.EXPECT().Create(expectedRequest).Return(&testPartnerAttachment, nil)

		err := RunPartnerAttachmentCreate(config)
		assert.NoError(t, err)
	})
}

func TestPartnerAttachmentCreateUnsupportedType(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Doit.Set(config.NS, doctl.ArgPartnerAttachmentType, "unsupported")
		err := RunPartnerAttachmentCreate(config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported attachment type")
	})
}

func TestPartnerNCGet(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Doit.Set(config.NS, doctl.ArgPartnerAttachmentType, "partner")

		pncID := "e819b321-a9a1-4078-b437-8e6b8bf13530"
		tm.partnerNetworkConnects.EXPECT().GetPartnerNetworkConnect(pncID).Return(&testPartnerAttachment, nil)

		config.Args = append(config.Args, pncID)

		err := RunPartnerNCGet(config)
		assert.NoError(t, err)
	})
}

func TestPartnerNCGetNoID(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Doit.Set(config.NS, doctl.ArgPartnerAttachmentType, "partner")

		err := RunPartnerNCGet(config)
		assert.Error(t, err)
	})
}

func TestPartnerNCList(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Doit.Set(config.NS, doctl.ArgPartnerAttachmentType, "partner")

		tm.partnerNetworkConnects.EXPECT().ListPartnerNetworkConnects().Return(testPartnerIAList, nil)

		err := RunPartnerNCList(config)
		assert.NoError(t, err)
	})
}

func TestPartnerNCDelete(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Doit.Set(config.NS, doctl.ArgPartnerAttachmentType, "partner")

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
		config.Doit.Set(config.NS, doctl.ArgPartnerAttachmentType, "partner")

		iaID := "ia-uuid1"
		iaName := "ia-name"
		vpcIDs := "f81d4fae-7dec-11d0-a765-00a0c91e6bf6,3f900b61-30d7-40d8-9711-8c5d6264b268"
		r := godo.PartnerNetworkConnectUpdateRequest{Name: iaName, VPCIDs: strings.Split(vpcIDs, ",")}
		tm.partnerNetworkConnects.EXPECT().UpdatePartnerNetworkConnect(iaID, &r).Return(&testPartnerAttachment, nil)

		config.Args = append(config.Args, iaID)
		config.Doit.Set(config.NS, doctl.ArgPartnerAttachmentName, iaName)
		config.Doit.Set(config.NS, doctl.ArgPartnerAttachmentVPCIDs, vpcIDs)

		err := RunPartnerNCUpdate(config)
		assert.NoError(t, err)
	})
}

func TestRunPartnerNCUpdateNoID(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Doit.Set(config.NS, doctl.ArgPartnerAttachmentType, "partner")

		err := RunPartnerNCUpdate(config)
		assert.Error(t, err)
	})
}

func TestPartnerAttachmentRouteList(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Doit.Set(config.NS, doctl.ArgPartnerAttachmentType, "partner")

		iaID := "ia-uuid1"
		config.Args = append(config.Args, iaID)
		tm.partnerNetworkConnects.EXPECT().ListPartnerAttachmentRoutes(iaID).Return(testPartnerIARouteList, nil)

		err := RunPartnerAttachmentRouteList(config)
		assert.NoError(t, err)
	})
}

func TestPartnerNCRegenerateServiceKey(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Doit.Set(config.NS, doctl.ArgPartnerAttachmentType, "partner")

		iaID := "e819b321-a9a1-4078-b437-8e6b8bf13530"
		tm.partnerNetworkConnects.EXPECT().RegenerateServiceKey(iaID).Return(&testRegenerateServiceKey, nil)

		config.Args = append(config.Args, iaID)

		err := RunPartnerNCRegenerateServiceKey(config)
		assert.NoError(t, err)
	})
}

func TestGetPartnerNCBGPAuthKey(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Doit.Set(config.NS, doctl.ArgPartnerAttachmentType, "partner")

		iaID := "e819b321-a9a1-4078-b437-8e6b8bf13530"
		tm.partnerNetworkConnects.EXPECT().GetBGPAuthKey(iaID).Return(&testBGPAuthKey, nil)

		config.Args = append(config.Args, iaID)

		err := RunGetPartnerNCBGPAuthKey(config)
		assert.NoError(t, err)
	})
}

func TestGetPartnerAttachmentServiceKey(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Doit.Set(config.NS, doctl.ArgPartnerAttachmentType, "partner")

		iaID := "e819b321-a9a1-4078-b437-8e6b8bf13530"
		tm.partnerNetworkConnects.EXPECT().GetServiceKey(iaID).Return(&testServiceKey, nil)

		config.Args = append(config.Args, iaID)

		err := RunGetPartnerAttachmentServiceKey(config)
		assert.NoError(t, err)
	})
}

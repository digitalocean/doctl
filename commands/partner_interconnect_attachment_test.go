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
	testPartnerAttachment = do.PartnerInterconnectAttachment{
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

	testPartnerIAList = do.PartnerInterconnectAttachments{
		testPartnerAttachment,
	}

	testPartnerAttachmentRoute = do.PartnerInterconnectAttachmentRoute{
		RemoteRoute: &godo.RemoteRoute{
			ID:   "test-route-id",
			Cidr: "10.10.0.0/24",
		},
	}

	testPartnerIARouteList = do.PartnerInterconnectAttachmentRoutes{
		testPartnerAttachmentRoute,
	}

	testRegenerateServiceKey = do.PartnerInterconnectAttachmentRegenerateServiceKey{
		RegenerateServiceKey: &godo.RegenerateServiceKey{},
	}

	testBGPAuthKey = do.PartnerInterconnectAttachmentBGPAuthKey{
		BgpAuthKey: &godo.BgpAuthKey{
			Value: "test-bgp-auth-key",
		},
	}

	testServiceKey = do.PartnerInterconnectAttachmentServiceKey{
		ServiceKey: &godo.ServiceKey{
			Value:     "test-service-key",
			State:     "active",
			CreatedAt: time.Date(2025, 1, 30, 0, 0, 0, 0, time.UTC),
		},
	}
)

func TestPartnerInterconnectAttachmentsCommand(t *testing.T) {
	cmd := PartnerInterconnectAttachments()
	assert.NotNil(t, cmd)

	assertCommandNames(t, cmd, "create", "get", "list", "delete", "update", "list-routes", "regenerate-service-key", "get-bgp-auth-key", "get-service-key")
}

func TestPartnerInterconnectAttachmentCreate(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Doit.Set(config.NS, doctl.ArgInterconnectAttachmentType, "partner")
		config.Doit.Set(config.NS, doctl.ArgPartnerInterconnectAttachmentName, "doctl-pia")
		config.Doit.Set(config.NS, doctl.ArgPartnerInterconnectAttachmentConnectionBandwidthInMbps, 50)
		config.Doit.Set(config.NS, doctl.ArgPartnerInterconnectAttachmentRegion, "stage2")
		config.Doit.Set(config.NS, doctl.ArgPartnerInterconnectAttachmentNaaSProvider, "MEGAPORT")
		config.Doit.Set(config.NS, doctl.ArgPartnerInterconnectAttachmentVPCIDs, []string{"d35e5cb7-7957-4643-8e3a-1ab4eb3a494c"})
		config.Doit.Set(config.NS, doctl.ArgPartnerInterconnectAttachmentBGPLocalASN, 65001)
		config.Doit.Set(config.NS, doctl.ArgPartnerInterconnectAttachmentBGPLocalRouterIP, "192.168.1.1")
		config.Doit.Set(config.NS, doctl.ArgPartnerInterconnectAttachmentBGPPeerASN, 65002)
		config.Doit.Set(config.NS, doctl.ArgPartnerInterconnectAttachmentBGPPeerRouterIP, "192.168.1.2")

		expectedRequest := &godo.PartnerNetworkConnectCreateRequest{
			Name:                      "doctl-pia",
			ConnectionBandwidthInMbps: 50,
			Region:                    "stage2",
			NaaSProvider:              "MEGAPORT",
			VPCIDs:                    []string{"d35e5cb7-7957-4643-8e3a-1ab4eb3a494c"},
		}

		tm.partnerInterconnectAttachment.EXPECT().Create(expectedRequest).Return(&testPartnerAttachment, nil)

		err := RunPartnerInterconnectAttachmentCreate(config)
		assert.NoError(t, err)
	})
}

func TestPartnerInterconnectAttachmentCreateUnsupportedType(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Doit.Set(config.NS, doctl.ArgInterconnectAttachmentType, "unsupported")
		err := RunPartnerInterconnectAttachmentCreate(config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported attachment type")
	})
}

func TestInterconnectAttachmentsGet(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Doit.Set(config.NS, doctl.ArgInterconnectAttachmentType, "partner")

		iaID := "e819b321-a9a1-4078-b437-8e6b8bf13530"
		tm.partnerInterconnectAttachment.EXPECT().GetPartnerInterconnectAttachment(iaID).Return(&testPartnerAttachment, nil)

		config.Args = append(config.Args, iaID)

		err := RunPartnerInterconnectAttachmentGet(config)
		assert.NoError(t, err)
	})
}

func TestInterconnectAttachmentsGetNoID(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Doit.Set(config.NS, doctl.ArgInterconnectAttachmentType, "partner")

		err := RunPartnerInterconnectAttachmentGet(config)
		assert.Error(t, err)
	})
}

func TestInterconnectAttachmentsList(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Doit.Set(config.NS, doctl.ArgInterconnectAttachmentType, "partner")

		tm.partnerInterconnectAttachment.EXPECT().ListPartnerInterconnectAttachments().Return(testPartnerIAList, nil)

		err := RunPartnerInterconnectAttachmentList(config)
		assert.NoError(t, err)
	})
}

func TestInterconnectAttachmentsDelete(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Doit.Set(config.NS, doctl.ArgInterconnectAttachmentType, "partner")

		iaID := "e819b321-a9a1-4078-b437-8e6b8bf13530"
		tm.partnerInterconnectAttachment.EXPECT().DeletePartnerInterconnectAttachment(iaID).Return(nil)

		config.Args = append(config.Args, iaID)
		config.Doit.Set(config.NS, doctl.ArgForce, true)

		err := RunPartnerInterconnectAttachmentDelete(config)
		assert.NoError(t, err)
	})
}

func TestInterconnectAttachmentsUpdate(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Doit.Set(config.NS, doctl.ArgInterconnectAttachmentType, "partner")

		iaID := "ia-uuid1"
		iaName := "ia-name"
		vpcIDs := "f81d4fae-7dec-11d0-a765-00a0c91e6bf6,3f900b61-30d7-40d8-9711-8c5d6264b268"
		r := godo.PartnerNetworkConnectUpdateRequest{Name: iaName, VPCIDs: strings.Split(vpcIDs, ",")}
		tm.partnerInterconnectAttachment.EXPECT().UpdatePartnerInterconnectAttachment(iaID, &r).Return(&testPartnerAttachment, nil)

		config.Args = append(config.Args, iaID)
		config.Doit.Set(config.NS, doctl.ArgPartnerInterconnectAttachmentName, iaName)
		config.Doit.Set(config.NS, doctl.ArgPartnerInterconnectAttachmentVPCIDs, vpcIDs)

		err := RunPartnerInterconnectAttachmentUpdate(config)
		assert.NoError(t, err)
	})
}

func TestInterconnectAttachmentsUpdateNoID(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Doit.Set(config.NS, doctl.ArgInterconnectAttachmentType, "partner")

		err := RunPartnerInterconnectAttachmentUpdate(config)
		assert.Error(t, err)
	})
}

func TestInterconnectAttachmentRoutesList(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Doit.Set(config.NS, doctl.ArgInterconnectAttachmentType, "partner")

		iaID := "ia-uuid1"
		config.Args = append(config.Args, iaID)
		tm.partnerInterconnectAttachment.EXPECT().ListPartnerInterconnectAttachmentRoutes(iaID).Return(testPartnerIARouteList, nil)

		err := RunPartnerInterconnectAttachmentRouteList(config)
		assert.NoError(t, err)
	})
}

func TestInterconnectAttachmentsRegenerateServiceKey(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Doit.Set(config.NS, doctl.ArgInterconnectAttachmentType, "partner")

		iaID := "e819b321-a9a1-4078-b437-8e6b8bf13530"
		tm.partnerInterconnectAttachment.EXPECT().RegenerateServiceKey(iaID).Return(&testRegenerateServiceKey, nil)

		config.Args = append(config.Args, iaID)

		err := RunPartnerInterconnectAttachmentRegenerateServiceKey(config)
		assert.NoError(t, err)
	})
}

func TestInterconnectAttachmentsBgpAuthKey(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Doit.Set(config.NS, doctl.ArgInterconnectAttachmentType, "partner")

		iaID := "e819b321-a9a1-4078-b437-8e6b8bf13530"
		tm.partnerInterconnectAttachment.EXPECT().GetBGPAuthKey(iaID).Return(&testBGPAuthKey, nil)

		config.Args = append(config.Args, iaID)

		err := RunGetPartnerInterconnectAttachmentBGPAuthKey(config)
		assert.NoError(t, err)
	})
}

func TestInterconnectAttachmentsGetServiceKey(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Doit.Set(config.NS, doctl.ArgInterconnectAttachmentType, "partner")

		iaID := "e819b321-a9a1-4078-b437-8e6b8bf13530"
		tm.partnerInterconnectAttachment.EXPECT().GetServiceKey(iaID).Return(&testServiceKey, nil)

		config.Args = append(config.Args, iaID)

		err := RunGetPartnerInterconnectAttachmentServiceKey(config)
		assert.NoError(t, err)
	})
}

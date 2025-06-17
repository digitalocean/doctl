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
	testPartnerAttachmentWithRedundancyZone = do.PartnerAttachment{
		PartnerAttachment: &godo.PartnerAttachment{
			ID:                        "test-id",
			Name:                      "doctl-pia",
			State:                     "active",
			ConnectionBandwidthInMbps: 50,
			Region:                    "stage2",
			NaaSProvider:              "MEGAPORT",
			VPCIDs:                    []string{"d35e5cb7-7957-4643-8e3a-1ab4eb3a494c"},
			CreatedAt:                 time.Date(2025, 1, 30, 0, 0, 0, 0, time.UTC),
			RedundancyZone:            "MEGAPORT_RED",
		},
	}

	testPartnerAttachmentWithHA = do.PartnerAttachment{
		PartnerAttachment: &godo.PartnerAttachment{
			ID:                        "test-id",
			Name:                      "doctl-pia",
			State:                     "active",
			ConnectionBandwidthInMbps: 50,
			Region:                    "stage2",
			NaaSProvider:              "MEGAPORT",
			VPCIDs:                    []string{"d35e5cb7-7957-4643-8e3a-1ab4eb3a494c"},
			CreatedAt:                 time.Date(2025, 1, 30, 0, 0, 0, 0, time.UTC),
			ParentUuid:                "fd1aad75-94ff-47f9-bae1-30c5d9caa14e",
			Children:                  []string{"28cedc83-85bb-4398-a48e-d2735ca028ac"},
		},
	}

	testPartnerAttachment = do.PartnerAttachment{
		PartnerAttachment: &godo.PartnerAttachment{
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

	testPartnerAttachmentList = do.PartnerAttachments{
		testPartnerAttachment,
	}

	testPartnerAttachmentRoute = do.PartnerAttachmentRoute{
		RemoteRoute: &godo.RemoteRoute{
			ID:   "test-route-id",
			Cidr: "10.10.0.0/24",
		},
	}

	testPartnerAttachmentRouteList = do.PartnerAttachmentRoutes{
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

func TestPartnerAttachmentsCommand(t *testing.T) {
	cmd := PartnerAttachments()
	assert.NotNil(t, cmd)

	assertCommandNames(t, cmd, "create", "get", "list", "delete", "update", "list-routes", "regenerate-service-key", "get-bgp-auth-key", "get-service-key")
}

func TestPartnerAttachmentCreate(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Doit.Set(config.NS, doctl.ArgPartnerAttachmentType, "partner")
		config.Doit.Set(config.NS, doctl.ArgPartnerAttachmentName, "doctl-pia")
		config.Doit.Set(config.NS, doctl.ArgPartnerAttachmentBandwidthInMbps, 50)
		config.Doit.Set(config.NS, doctl.ArgPartnerAttachmentRegion, "stage2")
		config.Doit.Set(config.NS, doctl.ArgPartnerAttachmentNaaSProvider, "MEGAPORT")
		config.Doit.Set(config.NS, doctl.ArgPartnerAttachmentVPCIDs, []string{"d35e5cb7-7957-4643-8e3a-1ab4eb3a494c"})
		config.Doit.Set(config.NS, doctl.ArgPartnerAttachmentBGPLocalASN, 65001)
		config.Doit.Set(config.NS, doctl.ArgPartnerAttachmentBGPLocalRouterIP, "192.168.1.1")
		config.Doit.Set(config.NS, doctl.ArgPartnerAttachmentBGPPeerASN, 65002)
		config.Doit.Set(config.NS, doctl.ArgPartnerAttachmentBGPPeerRouterIP, "192.168.1.2")

		expectedRequest := &godo.PartnerAttachmentCreateRequest{
			Name:                      "doctl-pia",
			ConnectionBandwidthInMbps: 50,
			Region:                    "stage2",
			NaaSProvider:              "MEGAPORT",
			VPCIDs:                    []string{"d35e5cb7-7957-4643-8e3a-1ab4eb3a494c"},
		}

		tm.partnerAttachments.EXPECT().Create(expectedRequest).Return(&testPartnerAttachment, nil)

		err := RunPartnerAttachmentCreate(config)
		assert.NoError(t, err)
	})
}

func TestPartnerAttachmentCreateHA(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Doit.Set(config.NS, doctl.ArgPartnerAttachmentType, "partner")
		config.Doit.Set(config.NS, doctl.ArgPartnerAttachmentName, "doctl-pia")
		config.Doit.Set(config.NS, doctl.ArgPartnerAttachmentBandwidthInMbps, 50)
		config.Doit.Set(config.NS, doctl.ArgPartnerAttachmentRegion, "stage2")
		config.Doit.Set(config.NS, doctl.ArgPartnerAttachmentNaaSProvider, "MEGAPORT")
		config.Doit.Set(config.NS, doctl.ArgPartnerAttachmentVPCIDs, []string{"d35e5cb7-7957-4643-8e3a-1ab4eb3a494c"})
		config.Doit.Set(config.NS, doctl.ArgPartnerAttachmentBGPLocalASN, 65001)
		config.Doit.Set(config.NS, doctl.ArgPartnerAttachmentBGPLocalRouterIP, "192.168.1.1")
		config.Doit.Set(config.NS, doctl.ArgPartnerAttachmentBGPPeerASN, 65002)
		config.Doit.Set(config.NS, doctl.ArgPartnerAttachmentBGPPeerRouterIP, "192.168.1.2")
		config.Doit.Set(config.NS, doctl.ArgPartnerAttachmentParentUUID, "fd1aad75-94ff-47f9-bae1-30c5d9caa14e")

		expectedRequest := &godo.PartnerAttachmentCreateRequest{
			Name:                      "doctl-pia",
			ConnectionBandwidthInMbps: 50,
			Region:                    "stage2",
			NaaSProvider:              "MEGAPORT",
			VPCIDs:                    []string{"d35e5cb7-7957-4643-8e3a-1ab4eb3a494c"},
			ParentUuid:                "fd1aad75-94ff-47f9-bae1-30c5d9caa14e",
		}

		tm.partnerAttachments.EXPECT().Create(expectedRequest).Return(&testPartnerAttachmentWithHA, nil)

		err := RunPartnerAttachmentCreate(config)
		assert.NoError(t, err)
	})
}

func TestPartnerAttachmentCreateWithRedundancyZone(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Doit.Set(config.NS, doctl.ArgPartnerAttachmentType, "partner")
		config.Doit.Set(config.NS, doctl.ArgPartnerAttachmentName, "doctl-pia")
		config.Doit.Set(config.NS, doctl.ArgPartnerAttachmentBandwidthInMbps, 50)
		config.Doit.Set(config.NS, doctl.ArgPartnerAttachmentRegion, "stage2")
		config.Doit.Set(config.NS, doctl.ArgPartnerAttachmentNaaSProvider, "MEGAPORT")
		config.Doit.Set(config.NS, doctl.ArgPartnerAttachmentVPCIDs, []string{"d35e5cb7-7957-4643-8e3a-1ab4eb3a494c"})
		config.Doit.Set(config.NS, doctl.ArgPartnerAttachmentBGPLocalASN, 65001)
		config.Doit.Set(config.NS, doctl.ArgPartnerAttachmentBGPLocalRouterIP, "192.168.1.1")
		config.Doit.Set(config.NS, doctl.ArgPartnerAttachmentBGPPeerASN, 65002)
		config.Doit.Set(config.NS, doctl.ArgPartnerAttachmentBGPPeerRouterIP, "192.168.1.2")
		config.Doit.Set(config.NS, doctl.ArgPartnerAttachmentRedundancyZone, "MEGAPORT_RED")

		expectedRequest := &godo.PartnerAttachmentCreateRequest{
			Name:                      "doctl-pia",
			ConnectionBandwidthInMbps: 50,
			Region:                    "stage2",
			NaaSProvider:              "MEGAPORT",
			VPCIDs:                    []string{"d35e5cb7-7957-4643-8e3a-1ab4eb3a494c"},
			RedundancyZone:            "MEGAPORT_RED",
		}

		tm.partnerAttachments.EXPECT().Create(expectedRequest).Return(&testPartnerAttachmentWithRedundancyZone, nil)

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

func TestPartnerAttachmentGet(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Doit.Set(config.NS, doctl.ArgPartnerAttachmentType, "partner")

		paID := "e819b321-a9a1-4078-b437-8e6b8bf13530"
		tm.partnerAttachments.EXPECT().GetPartnerAttachment(paID).Return(&testPartnerAttachment, nil)

		config.Args = append(config.Args, paID)

		err := RunPartnerAttachmentGet(config)
		assert.NoError(t, err)
	})
}

func TestPartnerAttachmentGetNoID(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Doit.Set(config.NS, doctl.ArgPartnerAttachmentType, "partner")

		err := RunPartnerAttachmentGet(config)
		assert.Error(t, err)
	})
}

func TestPartnerAttachmentList(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Doit.Set(config.NS, doctl.ArgPartnerAttachmentType, "partner")

		tm.partnerAttachments.EXPECT().ListPartnerAttachments().Return(testPartnerAttachmentList, nil)

		err := RunPartnerAttachmentList(config)
		assert.NoError(t, err)
	})
}

func TestPartnerAttachmentDelete(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Doit.Set(config.NS, doctl.ArgPartnerAttachmentType, "partner")

		iaID := "e819b321-a9a1-4078-b437-8e6b8bf13530"
		tm.partnerAttachments.EXPECT().DeletePartnerAttachment(iaID).Return(nil)

		config.Args = append(config.Args, iaID)
		config.Doit.Set(config.NS, doctl.ArgForce, true)

		err := RunPartnerAttachmentDelete(config)
		assert.NoError(t, err)
	})
}

func TestRunPartnerAttachmentUpdate(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Doit.Set(config.NS, doctl.ArgPartnerAttachmentType, "partner")

		iaID := "ia-uuid1"
		iaName := "ia-name"
		vpcIDs := "f81d4fae-7dec-11d0-a765-00a0c91e6bf6,3f900b61-30d7-40d8-9711-8c5d6264b268"
		r := godo.PartnerAttachmentUpdateRequest{Name: iaName, VPCIDs: strings.Split(vpcIDs, ",")}
		tm.partnerAttachments.EXPECT().UpdatePartnerAttachment(iaID, &r).Return(&testPartnerAttachment, nil)

		config.Args = append(config.Args, iaID)
		config.Doit.Set(config.NS, doctl.ArgPartnerAttachmentName, iaName)
		config.Doit.Set(config.NS, doctl.ArgPartnerAttachmentVPCIDs, vpcIDs)

		err := RunPartnerAttachmentUpdate(config)
		assert.NoError(t, err)
	})
}

func TestRunPartnerAttachmentUpdateNoID(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Doit.Set(config.NS, doctl.ArgPartnerAttachmentType, "partner")

		err := RunPartnerAttachmentUpdate(config)
		assert.Error(t, err)
	})
}

func TestPartnerAttachmentRouteList(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Doit.Set(config.NS, doctl.ArgPartnerAttachmentType, "partner")

		iaID := "ia-uuid1"
		config.Args = append(config.Args, iaID)
		tm.partnerAttachments.EXPECT().ListPartnerAttachmentRoutes(iaID).Return(testPartnerAttachmentRouteList, nil)

		err := RunPartnerAttachmentRouteList(config)
		assert.NoError(t, err)
	})
}

func TestPartnerAttachmentRegenerateServiceKey(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Doit.Set(config.NS, doctl.ArgPartnerAttachmentType, "partner")

		iaID := "e819b321-a9a1-4078-b437-8e6b8bf13530"
		tm.partnerAttachments.EXPECT().RegenerateServiceKey(iaID).Return(&testRegenerateServiceKey, nil)

		config.Args = append(config.Args, iaID)

		err := RunPartnerAttachmentRegenerateServiceKey(config)
		assert.NoError(t, err)
	})
}

func TestGetPartnerAttachmentBGPAuthKey(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Doit.Set(config.NS, doctl.ArgPartnerAttachmentType, "partner")

		iaID := "e819b321-a9a1-4078-b437-8e6b8bf13530"
		tm.partnerAttachments.EXPECT().GetBGPAuthKey(iaID).Return(&testBGPAuthKey, nil)

		config.Args = append(config.Args, iaID)

		err := RunGetPartnerAttachmentBGPAuthKey(config)
		assert.NoError(t, err)
	})
}

func TestGetPartnerAttachmentServiceKey(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Doit.Set(config.NS, doctl.ArgPartnerAttachmentType, "partner")

		iaID := "e819b321-a9a1-4078-b437-8e6b8bf13530"
		tm.partnerAttachments.EXPECT().GetServiceKey(iaID).Return(&testServiceKey, nil)

		config.Args = append(config.Args, iaID)

		err := RunGetPartnerAttachmentServiceKey(config)
		assert.NoError(t, err)
	})
}

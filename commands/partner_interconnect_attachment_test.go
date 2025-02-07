package commands

import (
	"testing"
	"time"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"
)

var (
	testPartnerAttachment = do.PartnerInterconnectAttachment{
		PartnerInterconnectAttachment: &godo.PartnerInterconnectAttachment{
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
)

func TestPartnerInterconnectAttachmentsCommand(t *testing.T) {
	cmd := PartnerInterconnectAttachments()
	assert.NotNil(t, cmd)

	assertCommandNames(t, cmd, "create")
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

		expectedRequest := &godo.PartnerInterconnectAttachmentCreateRequest{
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

package commands

import (
	"testing"

	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/do"
)

var (
	testIA = do.PartnerInterconnectAttachment{
		PartnerInterconnectAttachment: &godo.PartnerInterconnectAttachment{
			Name:   "ia-name",
			VPCIDs: []string{"f81d4fae-7dec-11d0-a765-00a0c91e6bf6", "3f900b61-30d7-40d8-9711-8c5d6264b268"},
		},
	}

	testIAList = do.PartnerInterconnectAttachments{
		testIA,
	}
)

func TestInterconnectAttachmentsCommand(t *testing.T) {
	cmd := PartnerInterconnectAttachments()
	assert.NotNil(t, cmd)
	assertCommandNames(t, cmd, "get", "list")
}

func TestInterconnectAttachmentsGet(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		iaID := "e819b321-a9a1-4078-b437-8e6b8bf13530"
		tm.vpcs.EXPECT().GetPartnerInterconnectAttachment(iaID).Return(&testIA, nil)

		config.Doit.Set("network", doctl.ArgInterconnectAttachmentType, "partner")

		config.Args = append(config.Args, iaID)

		err := RunPartnerInterconnectAttachmentGet(config)
		assert.NoError(t, err)
	})
}

func TestInterconnectAttachmentsGetNoID(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		err := RunPartnerInterconnectAttachmentGet(config)
		assert.Error(t, err)
	})
}

func TestInterconnectAttachmentsList(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.vpcs.EXPECT().ListPartnerInterconnectAttachments().Return(testIAList, nil)
		config.Doit.Set("network", doctl.ArgInterconnectAttachmentType, "partner")

		err := RunPartnerInterconnectAttachmentList(config)
		assert.NoError(t, err)
	})
}

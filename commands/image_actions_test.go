package commands

import (
	"testing"

	"github.com/bryanl/doit"
	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"
)

func TestImageActionCommand(t *testing.T) {
	cmd := ImageAction()
	assert.NotNil(t, cmd)
	assertCommandNames(t, cmd, "get", "transfer")
}

func TestImageActionsGet(t *testing.T) {
	client := &godo.Client{
		ImageActions: &doit.ImageActionsServiceMock{
			GetFn: func(imageID, actionID int) (*godo.Action, *godo.Response, error) {
				assert.Equal(t, imageID, 1)
				assert.Equal(t, actionID, 2)
				return &testAction, nil, nil
			},
		},
	}

	withTestClient(client, func(config *cmdConfig) {
		config.args = append(config.args, "1")

		config.doitConfig.Set(config.ns, doit.ArgActionID, 2)

		RunImageActionsGet(config)
	})

}

func TestImageActionsTransfer(t *testing.T) {
	client := &godo.Client{
		ImageActions: &doit.ImageActionsServiceMock{
			TransferFn: func(imageID int, req *godo.ActionRequest) (*godo.Action, *godo.Response, error) {
				assert.Equal(t, imageID, 1)

				region := (*req)["region"]
				assert.Equal(t, region, "dev0")

				return &testAction, nil, nil
			},
		},
	}

	withTestClient(client, func(config *cmdConfig) {
		config.args = append(config.args, "1")

		config.doitConfig.Set(config.ns, doit.ArgRegionSlug, "dev0")

		RunImageActionsTransfer(config)
	})
}

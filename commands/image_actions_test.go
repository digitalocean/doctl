package commands

import (
	"testing"

	"github.com/bryanl/doit"
	domocks "github.com/bryanl/doit/do/mocks"
	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"
)

func TestImageActionCommand(t *testing.T) {
	cmd := ImageAction()
	assert.NotNil(t, cmd)
	assertCommandNames(t, cmd, "get", "transfer")
}

func TestImageActionsGet(t *testing.T) {
	withTestClient(func(config *cmdConfig) {
		ias := &domocks.ImageActionsService{}
		config.ias = ias

		ias.On("Get", 1, 2).Return(&testAction, nil)

		config.args = append(config.args, "1")

		config.doitConfig.Set(config.ns, doit.ArgActionID, 2)

		RunImageActionsGet(config)
	})

}

func TestImageActionsTransfer(t *testing.T) {
	withTestClient(func(config *cmdConfig) {
		ias := &domocks.ImageActionsService{}
		config.ias = ias

		ar := &godo.ActionRequest{"region": "dev0"}
		ias.On("Transfer", 1, ar).Return(&testAction, nil)

		config.args = append(config.args, "1")

		config.doitConfig.Set(config.ns, doit.ArgRegionSlug, "dev0")

		RunImageActionsTransfer(config)
	})
}

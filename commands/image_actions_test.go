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
	withTestClient(t, func(config *cmdConfig, tm *tcMocks) {
		tm.imageActions.On("Get", 1, 2).Return(&testAction, nil)

		config.args = append(config.args, "1")

		config.doitConfig.Set(config.ns, doit.ArgActionID, 2)

		err := RunImageActionsGet(config)
		assert.NoError(t, err)
	})

}

func TestImageActionsTransfer(t *testing.T) {
	withTestClient(t, func(config *cmdConfig, tm *tcMocks) {
		ar := &godo.ActionRequest{"region": "dev0"}
		tm.imageActions.On("Transfer", 1, ar).Return(&testAction, nil)

		config.args = append(config.args, "1")

		config.doitConfig.Set(config.ns, doit.ArgRegionSlug, "dev0")

		err := RunImageActionsTransfer(config)
		assert.NoError(t, err)
	})
}

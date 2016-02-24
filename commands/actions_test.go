package commands

import (
	"testing"

	"github.com/bryanl/doit/do"
	domocks "github.com/bryanl/doit/do/mocks"
	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"
)

var (
	testAction     = do.Action{Action: &godo.Action{ID: 1, Region: &godo.Region{Slug: "dev0"}}}
	testActionList = do.Actions{
		testAction,
	}
)

func TestActionsCommand(t *testing.T) {
	cmd := Actions()
	assert.NotNil(t, cmd)
	assertCommandNames(t, cmd, "get", "list", "wait")
}

func TestActionList(t *testing.T) {
	withTestClient(func(config *cmdConfig) {
		as := &domocks.ActionsService{}
		config.acts = as
		as.On("List").Return(testActionList, nil)

		err := RunCmdActionList(config)
		assert.NoError(t, err)
	})
}

func TestActionGet(t *testing.T) {
	withTestClient(func(config *cmdConfig) {
		as := &domocks.ActionsService{}
		config.acts = as
		as.On("Get", 1).Return(&testAction, nil)

		config.args = append(config.args, "1")

		err := RunCmdActionGet(config)
		assert.NoError(t, err)
	})
}

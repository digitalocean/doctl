package commands

import (
	"testing"

	domocks "github.com/bryanl/doit/do/mocks"
	"github.com/stretchr/testify/assert"
)

func TestFloatingIPActionCommand(t *testing.T) {
	cmd := FloatingIPAction()
	assert.NotNil(t, cmd)
	assertCommandNames(t, cmd, "assign", "get", "unassign")
}

func TestFloatingIPActionsGet(t *testing.T) {
	withTestClient(func(config *cmdConfig) {
		fias := &domocks.FloatingIPActionsService{}
		config.fias = fias

		fias.On("Get", "127.0.0.1", 2).Return(&testAction, nil)

		config.args = append(config.args, "127.0.0.1", "2")

		RunFloatingIPActionsGet(config)
	})

}

func TestFloatingIPActionsAssign(t *testing.T) {
	withTestClient(func(config *cmdConfig) {
		fias := &domocks.FloatingIPActionsService{}
		config.fias = fias

		fias.On("Assign", "127.0.0.1", 2).Return(&testAction, nil)

		config.args = append(config.args, "127.0.0.1", "2")

		RunFloatingIPActionsAssign(config)
	})
}

func TestFloatingIPActionsUnassign(t *testing.T) {
	withTestClient(func(config *cmdConfig) {
		fias := &domocks.FloatingIPActionsService{}
		config.fias = fias

		fias.On("Unassign", "127.0.0.1").Return(&testAction, nil)

		config.args = append(config.args, "127.0.0.1")

		RunFloatingIPActionsUnassign(config)
	})
}

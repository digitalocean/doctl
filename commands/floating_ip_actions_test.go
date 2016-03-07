package commands

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFloatingIPActionCommand(t *testing.T) {
	cmd := FloatingIPAction()
	assert.NotNil(t, cmd)
	assertCommandNames(t, cmd, "assign", "get", "unassign")
}

func TestFloatingIPActionsGet(t *testing.T) {
	withTestClient(t, func(config *cmdConfig, tm *tcMocks) {
		tm.floatingIPActions.On("Get", "127.0.0.1", 2).Return(&testAction, nil)

		config.args = append(config.args, "127.0.0.1", "2")

		err := RunFloatingIPActionsGet(config)
		assert.NoError(t, err)
	})

}

func TestFloatingIPActionsAssign(t *testing.T) {
	withTestClient(t, func(config *cmdConfig, tm *tcMocks) {
		tm.floatingIPActions.On("Assign", "127.0.0.1", 2).Return(&testAction, nil)

		config.args = append(config.args, "127.0.0.1", "2")

		err := RunFloatingIPActionsAssign(config)
		assert.NoError(t, err)
	})
}

func TestFloatingIPActionsUnassign(t *testing.T) {
	withTestClient(t, func(config *cmdConfig, tm *tcMocks) {
		tm.floatingIPActions.On("Unassign", "127.0.0.1").Return(&testAction, nil)

		config.args = append(config.args, "127.0.0.1")

		err := RunFloatingIPActionsUnassign(config)
		assert.NoError(t, err)
	})
}

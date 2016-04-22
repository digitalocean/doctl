package commands

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDriveActionCommand(t *testing.T) {
	cmd := DriveAction()
	assert.NotNil(t, cmd)
	assertCommandNames(t, cmd, "attach", "detach")
}

func TestDriveActionsAttach(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.driveActions.On("Attach", testDrive.ID, testDroplet.ID).Return(&testAction, nil)
		config.Args = append(config.Args, testDrive.ID)
		config.Args = append(config.Args, fmt.Sprintf("%d", testDroplet.ID))

		err := RunDriveAttach(config)
		assert.NoError(t, err)
	})
}

func TestDriveActionsDetach(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.driveActions.On("Detach", testDrive.ID).Return(&testAction, nil)
		config.Args = append(config.Args, testDrive.ID)

		err := RunDriveDetach(config)
		assert.NoError(t, err)
	})
}

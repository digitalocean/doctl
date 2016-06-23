package commands

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVolumeActionCommand(t *testing.T) {
	cmd := VolumeAction()
	assert.NotNil(t, cmd)
	assertCommandNames(t, cmd, "attach", "detach")
}

func TestVolumeActionsAttach(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.volumeActions.On("Attach", testVolume.ID, testDroplet.ID).Return(&testAction, nil)
		config.Args = append(config.Args, testVolume.ID)
		config.Args = append(config.Args, fmt.Sprintf("%d", testDroplet.ID))

		err := RunVolumeAttach(config)
		assert.NoError(t, err)
	})
}

func TestVolumeActionsDetach(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.volumeActions.On("Detach", testVolume.ID).Return(&testAction, nil)
		config.Args = append(config.Args, testVolume.ID)

		err := RunVolumeDetach(config)
		assert.NoError(t, err)
	})
}

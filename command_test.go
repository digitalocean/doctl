// +build !windows

/*
Copyright 2018 The Doctl Authors All rights reserved.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package doctl

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLiveCommand_Run(t *testing.T) {
	lc := NewLiveCommand("/bin/ls")
	out, err := lc.Run("-d", "/tmp")
	assert.NoError(t, err)
	assert.True(t, len(string(out)) > 0)
}

func TestLiveCommand_Start(t *testing.T) {
	lc := NewLiveCommand("/bin/ls")
	err := lc.Start("/tmp")
	assert.NoError(t, err)

	assert.Equal(t, []string{"/bin/ls", "/tmp"}, lc.cmd.Args)

	err = lc.Stop()
	assert.NoError(t, err)
}

func TestMockCommand_Run(t *testing.T) {
	mc := NewMockCommand("/bin/ls")
	assert.Equal(t, "/bin/ls", mc.path)

	runErr := errors.New("an error")
	mc.runFn = func() error {
		return runErr
	}

	_, err := mc.Run()
	assert.Error(t, err)
}

func TestMockCommand_Start(t *testing.T) {
	mc := NewMockCommand("/bin/ls")

	startErr := errors.New("start error")
	stopErr := errors.New("top error")
	mc.startFn = func() error {
		return startErr
	}
	mc.stopFn = func() error {
		return stopErr
	}

	err := mc.Start()
	assert.Error(t, err)
	assert.True(t, mc.running)

	err = mc.Stop()
	assert.Error(t, err)
	assert.False(t, mc.running)

}

/*
Copyright 2026 The Doctl Authors All rights reserved.
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

package commands

import (
	"testing"

	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func action(id int, status string) *do.Action {
	return &do.Action{Action: &godo.Action{ID: id, Status: status}}
}

func TestWaitForAction(t *testing.T) {
	t.Run("polls while the action is in progress", func(t *testing.T) {
		withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
			gomock := tm.actions.EXPECT()
			gomock.Get(1).Return(action(1, "in-progress"), nil)
			gomock.Get(1).Return(action(1, "completed"), nil)

			a, err := waitForAction(newTestWaiter(), config.Actions(), 1, 0)

			require.NoError(t, err)
			assert.Equal(t, "completed", a.Status)
		})
	})

	// An errored action used to end the wait successfully, which left --wait
	// reporting that a reboot or a resize had finished when the API had
	// already given up on it.
	t.Run("reports an errored action as a failure", func(t *testing.T) {
		withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
			tm.actions.EXPECT().Get(1).Return(action(1, "errored"), nil)

			a, err := waitForAction(newTestWaiter(), config.Actions(), 1, 0)

			assert.Nil(t, a)
			assert.EqualError(t, err, "action (1) failed")
		})
	})
}

func TestWaitForActiveDroplets(t *testing.T) {
	droplet := func(id int, name, status string) *do.Droplet {
		return &do.Droplet{Droplet: &godo.Droplet{ID: id, Name: name, Status: status}}
	}

	t.Run("waits for every Droplet in the batch", func(t *testing.T) {
		withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
			expect := tm.droplets.EXPECT()
			expect.Get(1).Return(droplet(1, "one", "new"), nil)
			expect.Get(2).Return(droplet(2, "two", "active"), nil)
			expect.Get(1).Return(droplet(1, "one", "active"), nil)

			created := do.Droplets{*droplet(1, "one", "new"), *droplet(2, "two", "new")}
			active, err := waitForActiveDroplets(newTestWaiter(), config.Droplets(), created)

			require.NoError(t, err)
			// Each Droplet is returned as re-read, so that addresses assigned
			// during boot appear in the output.
			require.Len(t, active, 2)
			assert.Equal(t, "active", active[0].Status)
			assert.Equal(t, "active", active[1].Status)
		})
	})

	t.Run("an empty batch does not poll", func(t *testing.T) {
		withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
			active, err := waitForActiveDroplets(newTestWaiter(), config.Droplets(), nil)

			require.NoError(t, err)
			assert.Empty(t, active)
		})
	})
}

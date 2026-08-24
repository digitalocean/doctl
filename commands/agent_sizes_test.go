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

	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAgentSizesCommand(t *testing.T) {
	cmd := AgentSizes()
	assert.NotNil(t, cmd)
	assert.Equal(t, "sizes", cmd.Name())
	assert.Contains(t, cmd.Aliases, "size")
	assertCommandNames(t, cmd, "list")
}

func TestAgentSizesList(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.hostedAgents.EXPECT().
			ListSandboxSizes().
			Return([]godo.HostedAgentSandboxSize{
				{Slug: "mv-1vcpu-2gb", VCPUs: 1, MemoryMB: 2048},
				{Slug: "mv-2vcpu-4gb", VCPUs: 2, MemoryMB: 4096},
			}, nil)

		require.NoError(t, RunAgentsSizesList(config))
	})
}

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
	"errors"
	"testing"

	"github.com/digitalocean/doctl/commands/displayers"
	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAgentRegionsCommand(t *testing.T) {
	cmd := AgentRegions()
	assert.NotNil(t, cmd)
	assert.Equal(t, "regions", cmd.Name())
	assert.Contains(t, cmd.Aliases, "region")
	assertCommandNames(t, cmd, "list")
}

func TestAgentRegionsList(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.hostedAgents.EXPECT().
			ListRegions().
			Return([]godo.HostedAgentRegion{
				{Slug: "mem1", IsDefault: false},
				{Slug: "ric1", IsDefault: true},
			}, nil)

		require.NoError(t, RunAgentsRegionsList(config))
	})
}

// A team with no grants still sees the default region, so the command never
// prints an empty catalog.
func TestAgentRegionsList_DefaultOnly(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.hostedAgents.EXPECT().
			ListRegions().
			Return([]godo.HostedAgentRegion{{Slug: "ric1", IsDefault: true}}, nil)

		require.NoError(t, RunAgentsRegionsList(config))
	})
}

func TestAgentRegionsList_Error(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.hostedAgents.EXPECT().
			ListRegions().
			Return(nil, errors.New("unable to authenticate you"))

		err := RunAgentsRegionsList(config)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unable to authenticate you")
	})
}

// The default column is a marker rather than a bare bool: it answers "which
// one do I get if I omit --region".
func TestHostedAgentRegionDisplayer(t *testing.T) {
	d := &displayers.HostedAgentRegion{Regions: []godo.HostedAgentRegion{
		{Slug: "mem1"},
		{Slug: "ric1", IsDefault: true},
	}}

	assert.Equal(t, []string{"Slug", "Default"}, d.Cols())

	kv := d.KV()
	require.Len(t, kv, 2)
	assert.Equal(t, "mem1", kv[0]["Slug"])
	assert.Equal(t, "", kv[0]["Default"])
	assert.Equal(t, "ric1", kv[1]["Slug"])
	assert.Equal(t, "*", kv[1]["Default"])
}

func TestHostedAgentRegionDisplayer_Empty(t *testing.T) {
	var d *displayers.HostedAgentRegion
	assert.Empty(t, d.KV())
}

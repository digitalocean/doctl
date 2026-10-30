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

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/godo"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setAgentRegion overrides the group-level --region key for one test and
// restores what was there before. viper is process-global, so these tests
// cannot run in parallel.
func setAgentRegion(t *testing.T, v string) {
	t.Helper()
	prev := viper.GetString(agentRegionFlagKey)
	viper.Set(agentRegionFlagKey, v)
	t.Cleanup(func() { viper.Set(agentRegionFlagKey, prev) })
}

func TestRegionAPIURL(t *testing.T) {
	tests := []struct {
		name   string
		region string
		want   string
	}{
		{
			// No region means the region-less host, which the default region
			// serves. Returning "" leaves the client's host untouched, so an
			// unflagged command is byte-for-byte what it is today.
			name:   "unset leaves the host alone",
			region: "",
			want:   "",
		},
		{
			// An explicit region always names its own host, the default
			// region included, so --region never silently means "whatever
			// host I would have called anyway".
			name:   "default region named explicitly still gets its host",
			region: defaultAgentRegion,
			want:   "https://api-ric1.digitalocean.com/",
		},
		{
			name:   "non-default region gets its own host",
			region: "mem1",
			want:   "https://api-mem1.digitalocean.com/",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, regionAPIURL(tt.region))
		})
	}
}

func TestAgentRegionFlagRejectsBadSlugs(t *testing.T) {
	// doctl keeps no list of known regions — harness-api owns that — so the
	// only check is the shape of what gets interpolated into a hostname.
	for _, region := range []string{
		"MEM1",
		"mem 1",
		"mem1.evil.test",
		"mem1/v2",
		"../ric1",
		"mem_1",
	} {
		t.Run(region, func(t *testing.T) {
			setAgentRegion(t, region)
			_, err := agentRegionFlag()
			require.Error(t, err)
			assert.Contains(t, err.Error(), "invalid region")
		})
	}
}

func TestAgentRegionFlagAcceptsSlug(t *testing.T) {
	setAgentRegion(t, "mem1")
	got, err := agentRegionFlag()
	require.NoError(t, err)
	assert.Equal(t, "mem1", got)
}

func TestResolvedAgentRegionFallsBackToDefault(t *testing.T) {
	setAgentRegion(t, "")
	assert.Equal(t, defaultAgentRegion, resolvedAgentRegion())

	setAgentRegion(t, "mem1")
	assert.Equal(t, "mem1", resolvedAgentRegion())
}

func TestAgentRegionClientOpts(t *testing.T) {
	t.Run("unset sends no option", func(t *testing.T) {
		setAgentRegion(t, "")
		opts, err := agentRegionClientOpts()
		require.NoError(t, err)
		assert.Nil(t, opts)
	})

	t.Run("default region named explicitly repoints the client", func(t *testing.T) {
		setAgentRegion(t, defaultAgentRegion)
		opts, err := agentRegionClientOpts()
		require.NoError(t, err)
		require.Len(t, opts, 1)

		client, err := godo.New(nil, opts...)
		require.NoError(t, err)
		assert.Equal(t, "https://api-ric1.digitalocean.com/", client.BaseURL.String())
	})

	t.Run("non-default region repoints the client", func(t *testing.T) {
		setAgentRegion(t, "mem1")
		opts, err := agentRegionClientOpts()
		require.NoError(t, err)
		require.Len(t, opts, 1)

		client, err := godo.New(nil, opts...)
		require.NoError(t, err)
		assert.Equal(t, "https://api-mem1.digitalocean.com/", client.BaseURL.String())
	})

	t.Run("an invalid region fails the command", func(t *testing.T) {
		setAgentRegion(t, "not a region")
		_, err := agentRegionClientOpts()
		require.Error(t, err)
	})
}

func TestAgentRegionBaseURL(t *testing.T) {
	t.Run("unset region uses the default host", func(t *testing.T) {
		setAPIURL(t, "")
		setAgentRegion(t, "")
		got, err := agentRegionBaseURL()
		require.NoError(t, err)
		assert.Equal(t, defaultAPIURL, got)
	})

	t.Run("region selects its host", func(t *testing.T) {
		setAPIURL(t, "")
		setAgentRegion(t, "mem1")
		got, err := agentRegionBaseURL()
		require.NoError(t, err)
		assert.Equal(t, "https://api-mem1.digitalocean.com/", got)
	})

	t.Run("an explicit --api-url wins over --region", func(t *testing.T) {
		// doctl never rewrites a host the user chose; the region then travels
		// in the request body alone, and a disagreement is the server's
		// region_mismatch to report.
		setAPIURL(t, "https://api.preview.example.com/")
		setAgentRegion(t, "mem1")
		got, err := agentRegionBaseURL()
		require.NoError(t, err)
		assert.Equal(t, "https://api.preview.example.com/", got)
	})
}

func TestHostedAgentsWSURLHonorsRegion(t *testing.T) {
	// The tunnel builds its own URL instead of going through the godo client,
	// so it has to make the same host choice or it dials the wrong region and
	// 404s on a session that exists.
	t.Run("region selects the tunnel host", func(t *testing.T) {
		setAPIURL(t, "")
		setAgentRegion(t, "mem1")
		got, err := hostedAgentsWSURL("sess-1", 9119)
		require.NoError(t, err)
		assert.Equal(t, "wss://api-mem1.digitalocean.com/v2/agents/sessions/sess-1/port-forward/9119", got)
	})

	t.Run("no region keeps the region-less host", func(t *testing.T) {
		setAPIURL(t, "")
		setAgentRegion(t, "")
		got, err := hostedAgentsWSURL("sess-1", 9119)
		require.NoError(t, err)
		assert.Equal(t, "wss://api.digitalocean.com/v2/agents/sessions/sess-1/port-forward/9119", got)
	})

	t.Run("an explicit --api-url wins", func(t *testing.T) {
		setAPIURL(t, "https://api.preview.example.com/")
		setAgentRegion(t, "mem1")
		got, err := hostedAgentsWSURL("sess-1", 9119)
		require.NoError(t, err)
		assert.Equal(t, "wss://api.preview.example.com/v2/agents/sessions/sess-1/port-forward/9119", got)
	})

	t.Run("an invalid region fails before dialing", func(t *testing.T) {
		setAPIURL(t, "")
		setAgentRegion(t, "nope!")
		_, err := hostedAgentsWSURL("sess-1", 9119)
		require.Error(t, err)
	})
}

func TestLocationBaseURL(t *testing.T) {
	tests := []struct {
		name     string
		location string
		want     string
	}{
		{
			// The canonical URL of the created session, which is what the
			// server sends; only its host matters for follow-up calls.
			name:     "resource url reduces to its host",
			location: "https://api-mem1.digitalocean.com/v2/agents/sessions/sess_abc123",
			want:     "https://api-mem1.digitalocean.com/",
		},
		{
			name:     "no header leaves the host alone",
			location: "",
			want:     "",
		},
		{
			// A header doctl cannot read must not fail a session that was
			// created successfully.
			name:     "unreadable header leaves the host alone",
			location: "://nope",
			want:     "",
		},
		{
			name:     "path-only header leaves the host alone",
			location: "/v2/agents/sessions/sess_abc123",
			want:     "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, locationBaseURL(tt.location))
		})
	}
}

func TestAgentsRegionFlagIsPersistent(t *testing.T) {
	cmd := Agents()

	flag := cmd.PersistentFlags().Lookup(doctl.ArgRegionSlug)
	require.NotNil(t, flag, "--region must live on the group so it is accepted before the subcommand")
	assert.Equal(t, "", flag.DefValue, "an unset --region must mean the region-less host")

	// Cobra merges a parent's persistent flags into every child's flag set,
	// which is what makes `... create --region mem1` work as well as
	// `... --region mem1 create`.
	for _, child := range cmd.Commands() {
		assert.NotNilf(t, child.InheritedFlags().Lookup(doctl.ArgRegionSlug),
			"%s must accept --region after the subcommand name", child.Name())
	}
}

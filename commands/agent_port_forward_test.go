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

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseForwardPair(t *testing.T) {
	tests := []struct {
		name string
		arg  string
		want forwardPair
	}{
		{"bare port forwards both ends", "8080", forwardPair{local: 8080, remote: 8080}},
		{"explicit pair", "12222:2222", forwardPair{local: 12222, remote: 2222}},
		{"zero local lets the OS pick", "0:9119", forwardPair{local: 0, remote: 9119}},
		{"lowest allowed remote", "1024", forwardPair{local: 1024, remote: 1024}},
		{"highest allowed remote", "65535", forwardPair{local: 65535, remote: 65535}},
		// Empty local is treated as "same port", not as an OS-assigned one --
		// only an explicit 0 does that. kubectl reads ":<remote>" as random.
		{"empty local mirrors the remote", ":8080", forwardPair{local: 8080, remote: 8080}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseForwardPair(tt.arg)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestParseForwardPairErrors(t *testing.T) {
	tests := []struct {
		name string
		arg  string
	}{
		{"empty", ""},
		{"not a number", "http"},
		{"remote below the privileged-port floor", "80"},
		{"remote just below the floor", "1023"},
		{"remote above the range", "65536"},
		{"remote negative", "-1"},
		{"missing remote", "8080:"},
		{"local not a number", "abc:8080"},
		{"local above the range", "70000:8080"},
		{"local negative", "-1:8080"},
		{"too many segments", "1:2:3"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseForwardPair(tt.arg)
			assert.Error(t, err)
		})
	}
}

// setAPIURL overrides the global --api-url viper key for one test and restores
// whatever was there before. hostedAgentsWSURL reads it directly, and viper is
// process-global, so these tests cannot run in parallel.
func setAPIURL(t *testing.T, v string) {
	t.Helper()
	prev := viper.GetString("api-url")
	viper.Set("api-url", v)
	t.Cleanup(func() { viper.Set("api-url", prev) })
}

func TestHostedAgentsWSURL(t *testing.T) {
	tests := []struct {
		name   string
		apiURL string
		want   string
	}{
		{
			// Unset --api-url falls back to doctl.HostedAgentsAPIURL, whose
			// trailing slash must not survive into the joined path.
			name:   "default hosted-agents host",
			apiURL: "",
			want:   "wss://ohr-agent.do-ai.run/v2/agents/sessions/sess-1/port-forward/9119",
		},
		{
			name:   "preview host via --api-url",
			apiURL: "https://ohr-agent.do-ai-test.run",
			want:   "wss://ohr-agent.do-ai-test.run/v2/agents/sessions/sess-1/port-forward/9119",
		},
		{
			name:   "https upgrades to wss",
			apiURL: "https://api.example.com/",
			want:   "wss://api.example.com/v2/agents/sessions/sess-1/port-forward/9119",
		},
		{
			name:   "http downgrades to ws for a local dev stack",
			apiURL: "http://127.0.0.1:8080",
			want:   "ws://127.0.0.1:8080/v2/agents/sessions/sess-1/port-forward/9119",
		},
		{
			name:   "base path prefix is preserved",
			apiURL: "https://api.example.com/edge/",
			want:   "wss://api.example.com/edge/v2/agents/sessions/sess-1/port-forward/9119",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setAPIURL(t, tt.apiURL)
			got, err := hostedAgentsWSURL("sess-1", 9119)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestHostedAgentsWSURLErrors(t *testing.T) {
	tests := []struct {
		name   string
		apiURL string
	}{
		{"unparseable url", "://nope"},
		{"unsupported scheme", "ftp://api.example.com"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setAPIURL(t, tt.apiURL)
			_, err := hostedAgentsWSURL("sess-1", 9119)
			assert.Error(t, err)
		})
	}
}

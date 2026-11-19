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
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/digitalocean/doctl"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	yaml "gopkg.in/yaml.v2"
)

func TestPreferHarnessKeyInManifest(t *testing.T) {
	t.Run("renames agent to harness", func(t *testing.T) {
		out, err := preferHarnessKeyInManifest([]byte("name: demo\nagent: opencode\n"))
		require.NoError(t, err)

		var doc map[string]any
		require.NoError(t, yaml.Unmarshal(out, &doc))
		assert.Equal(t, "opencode", doc["harness"])
		assert.NotContains(t, doc, "agent")
		assert.Equal(t, "demo", doc["name"])
	})

	t.Run("leaves harness alone", func(t *testing.T) {
		in := []byte("harness: claude-code\nname: demo\n")
		out, err := preferHarnessKeyInManifest(in)
		require.NoError(t, err)
		assert.Contains(t, string(out), "harness: claude-code")
		assert.NotContains(t, string(out), "agent:")
	})

	t.Run("drops redundant agent when harness matches", func(t *testing.T) {
		out, err := preferHarnessKeyInManifest([]byte("harness: opencode\nagent: opencode\n"))
		require.NoError(t, err)

		var doc map[string]any
		require.NoError(t, yaml.Unmarshal(out, &doc))
		assert.Equal(t, "opencode", doc["harness"])
		assert.NotContains(t, doc, "agent")
	})

	t.Run("leaves conflicting agent and harness for validation", func(t *testing.T) {
		out, err := preferHarnessKeyInManifest([]byte("harness: opencode\nagent: codex\n"))
		require.NoError(t, err)
		assert.Contains(t, string(out), "harness: opencode")
		assert.Contains(t, string(out), "agent: codex")
	})

	t.Run("legacy envelope unchanged", func(t *testing.T) {
		in := []byte(sampleManifest)
		out, err := preferHarnessKeyInManifest(in)
		require.NoError(t, err)
		assert.Contains(t, string(out), "apiVersion:")
		assert.NotContains(t, string(out), "harness:")
	})
}

func TestRunAgentsCreate_DryRunRewritesAgentKeyToHarness(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "agents.yaml")
	require.NoError(t, os.WriteFile(path, []byte("name: demo\nagent: opencode\n"), 0o644))

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		var buf bytes.Buffer
		config.Out = &buf
		config.Args = []string{path}
		config.Doit.Set(config.NS, doctl.ArgAgentDryRun, true)

		require.NoError(t, RunAgentsCreate(config))
		out := buf.String()
		assert.Contains(t, out, "harness: opencode")
		assert.NotContains(t, out, "agent:")
		assert.Contains(t, out, "name: demo")
	})
}

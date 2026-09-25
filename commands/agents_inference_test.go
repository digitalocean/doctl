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

package commands

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveDOInferenceSkipsWhenKeyIsSupplied(t *testing.T) {
	t.Setenv(harnessInferenceAPIKeyEnv, "")

	t.Run("adapter with no key slot never engages", func(t *testing.T) {
		t.Setenv(openAIAPIKeyEnv, "")
		got, err := resolveDOInference(nil, "opencode", nil)
		require.NoError(t, err)
		assert.Nil(t, got, "opencode resolves credentials itself; doctl cannot tell a key is missing")
	})

	t.Run("native key in the environment", func(t *testing.T) {
		t.Setenv(openAIAPIKeyEnv, "sk-from-env")
		got, err := resolveDOInference(nil, codexAgentName, nil)
		require.NoError(t, err)
		assert.Nil(t, got)
	})

	t.Run("native key passed as --secret", func(t *testing.T) {
		t.Setenv(openAIAPIKeyEnv, "")
		got, err := resolveDOInference(nil, codexAgentName,
			map[string]string{openAIAPIKeyEnv: "sk-from-flag"})
		require.NoError(t, err)
		assert.Nil(t, got)
	})

	t.Run("blank native key is not a key", func(t *testing.T) {
		t.Setenv(openAIAPIKeyEnv, "")
		// The fallback engages, and with no TTY to paste into it fails with
		// instructions — the point being that a blank --secret did not pass
		// for a supplied credential.
		_, err := resolveDOInference(&CmdConfig{Out: io.Discard}, codexAgentName,
			map[string]string{openAIAPIKeyEnv: "  "})
		require.Error(t, err)
		assert.Contains(t, err.Error(), modelAccessKeyDocsURL)
	})
}

func TestResolveDOInferenceUsesEnvKey(t *testing.T) {
	t.Setenv(openAIAPIKeyEnv, "")
	t.Setenv(harnessInferenceAPIKeyEnv, "do-model-access-key")
	t.Setenv(harnessInferenceModelEnv, "")

	got, err := resolveDOInference(nil, codexAgentName, nil)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "do-model-access-key", got.apiKey)
	assert.Equal(t, defaultDOInferenceModel, got.model)
}

// The key the user pastes is used for this invocation only. Writing it to disk
// would put a live credential in a second, unaudited place.
func TestResolveDOInferenceDoesNotPersistTheKey(t *testing.T) {
	home := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", home)
	t.Setenv("HOME", home)
	t.Setenv(openAIAPIKeyEnv, "")
	t.Setenv(harnessInferenceAPIKeyEnv, "")

	_, err := resolveDOInference(&CmdConfig{Out: io.Discard}, codexAgentName, nil)
	require.Error(t, err, "no TTY, so the prompt path fails rather than caching anything")

	var found []string
	_ = filepath.Walk(home, func(path string, info os.FileInfo, err error) error {
		if err == nil && info != nil && !info.IsDir() {
			found = append(found, path)
		}
		return nil
	})
	assert.Empty(t, found, "doctl must not write a credential store")
}

// Without a TTY there is nobody to paste a key, so the fallback has to fail
// with instructions rather than hang or create a session on a blank credential.
// This is also the shape CI hits.
func TestResolveDOInferenceNonInteractiveExplainsHowToGetAKey(t *testing.T) {
	t.Setenv(openAIAPIKeyEnv, "")
	t.Setenv(harnessInferenceAPIKeyEnv, "")

	_, err := resolveDOInference(&CmdConfig{Out: io.Discard}, codexAgentName, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), modelAccessKeyDocsURL)
	assert.Contains(t, err.Error(), harnessInferenceAPIKeyEnv)
	assert.Contains(t, err.Error(), openAIAPIKeyEnv)
	assert.NotContains(t, err.Error(), "is not set",
		"running on DigitalOcean inference is a supported path, not a broken environment")
}

// The banner is what the user actually reads, so its wording is worth pinning.
func TestModelAccessKeyBannerLeadsWithInference(t *testing.T) {
	var buf bytes.Buffer
	printModelAccessKeyBanner(&buf, "kimi-k3", openAIAPIKeyEnv)
	out := buf.String()

	first := strings.SplitN(out, "\n", 2)[0]
	assert.Contains(t, first, "DigitalOcean inference")
	assert.NotContains(t, first, openAIAPIKeyEnv,
		"the first line must not open on a variable the user did not set")

	assert.Contains(t, out, "kimi-k3")
	assert.Contains(t, out, modelAccessKeyConsolePath)
	assert.Contains(t, out, modelAccessKeyDocsURL)
	assert.Contains(t, out, "cancel and set "+openAIAPIKeyEnv,
		"their own provider is still offered, as an alternative rather than a fault")
}

func TestBuildHarnessManifestDOInference(t *testing.T) {
	raw, err := buildHarnessManifest(harnessManifestOpts{
		harness:    "codex",
		permission: "allow",
		inference:  &doInference{apiKey: "do-key", model: "kimi-k3"},
	})
	require.NoError(t, err)

	out := string(raw)
	assert.Contains(t, out, harnessInferenceModelEnv+": kimi-k3")
	assert.Contains(t, out, harnessInferenceBaseURLEnv+": "+defaultDOInferenceBaseURL)
	assert.Contains(t, out, harnessInferenceAPIKeyEnv+": do-key")
	assert.NotContains(t, out, openAIAPIKeyEnv,
		"writing an empty native key slot would strand the session on a blank credential")
}

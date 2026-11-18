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
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHasInferenceProvider(t *testing.T) {
	cases := []struct {
		name     string
		manifest string
		want     bool
	}{
		{name: "model env", manifest: "harness: opencode\nenv:\n  HARNESS_INFERENCE_MODEL: kimi-k3\n", want: true},
		{name: "openai secret", manifest: "harness: codex\nsecrets:\n  OPENAI_API_KEY: sk-openai\n", want: true},
		{name: "anthropic secret", manifest: "harness: claude-code\nsecrets:\n  ANTHROPIC_API_KEY: sk-ant\n", want: true},
		{name: "harness key", manifest: "harness: opencode\nsecrets:\n  HARNESS_INFERENCE_API_KEY: sk-do\n", want: true},
		{name: "placeholder secret is not a provider", manifest: "harness: claude-code\nsecrets:\n  ANTHROPIC_API_KEY: ${ANTHROPIC_API_KEY}\n", want: false},
		{name: "empty", manifest: "harness: opencode\n", want: false},
		{name: "provided minimax key counts as a provider", manifest: "harness: minimax-code\nsecrets:\n  MINIMAX_API_KEY: mm-key\n", want: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, hasInferenceProvider([]byte(tc.manifest)))
		})
	}
}

func TestDefaultModelForHarness(t *testing.T) {
	assert.Equal(t, defaultDOSIModel, defaultModelForHarness("opencode"))
	assert.Equal(t, defaultOpenHandsModel, defaultModelForHarness("openhands"))
	assert.Equal(t, defaultOpenHandsModel, defaultModelForHarness("open-hands"))
	assert.Equal(t, defaultHermesModel, defaultModelForHarness("hermes"))
	assert.Equal(t, defaultDOSIModel, defaultModelForHarness("claude-code"))
	assert.Equal(t, defaultDOSIModel, defaultModelForHarness(""))
}

func TestInjectDefaultDOSIModel(t *testing.T) {
	t.Run("injects shared DO SI default for opencode", func(t *testing.T) {
		out, err := injectDefaultDOSIModel([]byte("harness: opencode\n"))
		require.NoError(t, err)
		assert.Contains(t, string(out), harnessInferenceModelEnv+": "+defaultDOSIModel)
	})

	t.Run("openhands file gets the openhands catalogue default", func(t *testing.T) {
		out, err := injectDefaultDOSIModel([]byte("harness: openhands\ntemplate: openhands\nkeep_warm: false\n"))
		require.NoError(t, err)
		assert.Contains(t, string(out), harnessInferenceModelEnv+": "+defaultOpenHandsModel)
		assert.NotContains(t, string(out), defaultDOSIModel)
	})

	t.Run("hermes file gets the hermes catalogue default", func(t *testing.T) {
		out, err := injectDefaultDOSIModel([]byte("harness: hermes\n"))
		require.NoError(t, err)
		assert.Contains(t, string(out), harnessInferenceModelEnv+": "+defaultHermesModel)
	})

	t.Run("does not overwrite existing model", func(t *testing.T) {
		in := []byte("harness: opencode\nenv:\n  HARNESS_INFERENCE_MODEL: already-set\n")
		out, err := injectDefaultDOSIModel(in)
		require.NoError(t, err)
		assert.Contains(t, string(out), "HARNESS_INFERENCE_MODEL: already-set")
		assert.NotContains(t, string(out), defaultDOSIModel)
	})
}

func TestResolveInferenceProvider_ProviderlessInject(t *testing.T) {
	out, err := resolveInferenceProvider([]byte("harness: opencode\n"), false)
	require.NoError(t, err)
	assert.True(t, needsManagedModelAccessKey(out), "injection must precede the mint predicate")
	assert.Contains(t, string(out), harnessInferenceModelEnv+": "+defaultDOSIModel)
}

func TestResolveInferenceProvider_OpenHandsFileUsesOpenHandsDefault(t *testing.T) {
	out, err := resolveInferenceProvider([]byte("harness: openhands\ntemplate: openhands\nkeep_warm: false\n"), false)
	require.NoError(t, err)
	assert.Contains(t, string(out), harnessInferenceModelEnv+": "+defaultOpenHandsModel)
	assert.True(t, needsManagedModelAccessKey(out))
}

func TestResolveInferenceProvider_HermesFileUsesHermesDefault(t *testing.T) {
	out, err := resolveInferenceProvider([]byte("harness: hermes\n"), false)
	require.NoError(t, err)
	assert.Contains(t, string(out), harnessInferenceModelEnv+": "+defaultHermesModel)
	assert.True(t, needsManagedModelAccessKey(out))
}

func TestResolveInferenceProvider_BareSandboxSkipped(t *testing.T) {
	in := []byte("harness: none\n")
	out, err := resolveInferenceProvider(in, false)
	require.NoError(t, err)
	assert.NotContains(t, string(out), harnessInferenceModelEnv)
	assert.False(t, needsManagedModelAccessKey(out))
}

func TestResolveInferenceProvider_BlankVendorToDOSI(t *testing.T) {
	_ = os.Unsetenv(anthropicAPIKeyEnv)
	raw, err := buildHarnessManifest(harnessManifestOpts{harness: "claude-code"})
	require.NoError(t, err)

	out, err := resolveInferenceProvider(raw, false)
	require.NoError(t, err)

	assert.NotContains(t, string(out), anthropicAPIKeyEnv)
	assert.Contains(t, string(out), harnessInferenceModelEnv+": "+defaultDOSIModel)
	assert.True(t, needsManagedModelAccessKey(out))
}

func TestResolveInferenceProvider_KeepsProvidedVendorKey(t *testing.T) {
	t.Setenv(anthropicAPIKeyEnv, "sk-ant-real")
	raw, err := buildHarnessManifest(harnessManifestOpts{harness: "claude-code"})
	require.NoError(t, err)

	out, err := resolveInferenceProvider(raw, false)
	require.NoError(t, err)

	assert.Contains(t, string(out), "ANTHROPIC_API_KEY: ${ANTHROPIC_API_KEY}")
	assert.NotContains(t, string(out), harnessInferenceModelEnv)
	assert.False(t, needsManagedModelAccessKey(out))
}

func TestResolveInferenceProvider_OpenAISandboxKeepsRequiredKey(t *testing.T) {
	_ = os.Unsetenv(openAIAPIKeyEnv)
	raw, err := buildHarnessManifest(harnessManifestOpts{harness: "codex-agentapi", prompt: "hi"})
	require.NoError(t, err)

	out, err := resolveInferenceProvider(raw, false)
	require.NoError(t, err)

	// Must not strip CODEX_API_KEY / OPENAI path into DO SI for agentapi, and
	// must not catch-all inject HARNESS_INFERENCE_MODEL.
	assert.Contains(t, string(out), "CODEX_API_KEY")
	assert.Contains(t, string(out), "${OPENAI_API_KEY}")
	assert.NotContains(t, string(out), harnessInferenceModelEnv)
}

func TestResolveInferenceProvider_MinimaxStaysRequired(t *testing.T) {
	_ = os.Unsetenv(minimaxAPIKeyEnv)
	raw, err := buildHarnessManifest(harnessManifestOpts{harness: "minimax-code"})
	require.NoError(t, err)

	out, err := resolveInferenceProvider(raw, false)
	require.NoError(t, err)

	// MiniMax is out of blank→DO SI; placeholder remains for ensureEnvVar to require.
	assert.Contains(t, string(out), "MINIMAX_API_KEY: ${MINIMAX_API_KEY}")
}

func TestResolveInferenceProvider_PromptBlankFallsBackToDOSI(t *testing.T) {
	_ = os.Unsetenv(openAIAPIKeyEnv)
	prev := promptOptionalVendorKey
	t.Cleanup(func() { promptOptionalVendorKey = prev })
	promptOptionalVendorKey = func(name string) (string, error) {
		assert.Equal(t, openAIAPIKeyEnv, name)
		return "", nil
	}

	raw, err := buildHarnessManifest(harnessManifestOpts{harness: "codex"})
	require.NoError(t, err)

	out, err := resolveInferenceProvider(raw, true)
	require.NoError(t, err)
	assert.True(t, needsManagedModelAccessKey(out))
	assert.Contains(t, string(out), harnessInferenceModelEnv+": "+defaultDOSIModel)
}

func TestBuildHarnessManifest_OpencodeDefaultsToDOSI(t *testing.T) {
	raw, err := buildHarnessManifest(harnessManifestOpts{harness: "opencode"})
	require.NoError(t, err)
	assert.Contains(t, string(raw), harnessInferenceModelEnv+": "+defaultDOSIModel)
	assert.True(t, needsManagedModelAccessKey(raw))
}

func TestResolveInferenceProvider_MinimaxPlaceholderDoesNotCatchAllInject(t *testing.T) {
	_ = os.Unsetenv(minimaxAPIKeyEnv)
	raw, err := buildHarnessManifest(harnessManifestOpts{harness: "minimax-code"})
	require.NoError(t, err)

	out, err := resolveInferenceProvider(raw, false)
	require.NoError(t, err)
	assert.Contains(t, string(out), "MINIMAX_API_KEY: ${MINIMAX_API_KEY}")
	assert.NotContains(t, string(out), harnessInferenceModelEnv)
}

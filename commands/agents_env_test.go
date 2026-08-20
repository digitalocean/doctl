/*
Copyright 2026 The Doctl Authors All rights reserved.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
    15|See the License for the specific language governing permissions and
limitations under the License.
*/

package commands

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsSecretEnvVarName(t *testing.T) {
	assert.True(t, isSecretEnvVarName("OPENAI_API_KEY"))
	assert.True(t, isSecretEnvVarName("GITHUB_TOKEN"))
	assert.True(t, isSecretEnvVarName("db_password"))
	assert.False(t, isSecretEnvVarName("CODEX_ENVIRONMENT_ID"))
	assert.False(t, isSecretEnvVarName("HOME"))
}

func TestEnsureEnvVar_UsesExisting(t *testing.T) {
	t.Setenv("DOCTL_TEST_ENSURE_KEY", "already-set")
	prevPrompt := promptEnvVarValue
	t.Cleanup(func() { promptEnvVarValue = prevPrompt })
	promptEnvVarValue = func(string) (string, error) {
		t.Fatal("should not prompt when env is set")
		return "", nil
	}

	v, err := ensureEnvVar("DOCTL_TEST_ENSURE_KEY")
	require.NoError(t, err)
	assert.Equal(t, "already-set", v)
}

func TestEnsureEnvVar_PromptsAndExports(t *testing.T) {
	_ = os.Unsetenv("DOCTL_TEST_ENSURE_PROMPT")
	prevPrompt := promptEnvVarValue
	t.Cleanup(func() {
		promptEnvVarValue = prevPrompt
		_ = os.Unsetenv("DOCTL_TEST_ENSURE_PROMPT")
	})
	promptEnvVarValue = func(name string) (string, error) {
		assert.Equal(t, "DOCTL_TEST_ENSURE_PROMPT", name)
		return "from-prompt", nil
	}

	v, err := ensureEnvVar("DOCTL_TEST_ENSURE_PROMPT")
	require.NoError(t, err)
	assert.Equal(t, "from-prompt", v)
	assert.Equal(t, "from-prompt", os.Getenv("DOCTL_TEST_ENSURE_PROMPT"))
}

func TestExpandManifestEnvCollect_NonInteractiveListsMissing(t *testing.T) {
	_ = os.Unsetenv("DOCTL_TEST_COLLECT_A")
	prevInteractive := Interactive
	t.Cleanup(func() {
		Interactive = prevInteractive
		_ = os.Unsetenv("DOCTL_TEST_COLLECT_A")
	})

	Interactive = false
	_, err := expandManifestEnvCollect([]byte("a: ${DOCTL_TEST_COLLECT_A}\n"), os.LookupEnv)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "DOCTL_TEST_COLLECT_A")
	assert.Contains(t, err.Error(), "interactive terminal")
}

func TestExpandManifestEnvCollect_AfterEnsureEnvVar(t *testing.T) {
	_ = os.Unsetenv("DOCTL_TEST_COLLECT_B")
	prevPrompt := promptEnvVarValue
	t.Cleanup(func() {
		promptEnvVarValue = prevPrompt
		_ = os.Unsetenv("DOCTL_TEST_COLLECT_B")
	})
	promptEnvVarValue = func(name string) (string, error) {
		return "collected", nil
	}
	_, err := ensureEnvVar("DOCTL_TEST_COLLECT_B")
	require.NoError(t, err)
	out, err := expandManifestEnvCollect([]byte("a: ${DOCTL_TEST_COLLECT_B}\n"), os.LookupEnv)
	require.NoError(t, err)
	assert.Equal(t, "a: collected\n", string(out))
}

func TestEnsureManifestEnvVars_SkipsENV_ID(t *testing.T) {
	prevInteractive := Interactive
	prevPrompt := promptEnvVarValue
	t.Cleanup(func() {
		Interactive = prevInteractive
		promptEnvVarValue = prevPrompt
	})
	Interactive = true
	promptEnvVarValue = func(string) (string, error) {
		t.Fatal("ENV_ID must not be prompted")
		return "", nil
	}

	err := ensureManifestEnvVars([]byte("id: ${ENV_ID}\n"), os.LookupEnv)
	require.NoError(t, err)

	_, err = expandManifestEnvLookup([]byte("id: ${ENV_ID}\n"), os.LookupEnv)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ENV_ID")
}

func TestPrepareOpenAISandboxStart_PromptsForAPIKey(t *testing.T) {
	_ = os.Unsetenv(openAIAPIKeyEnv)
	prevPrompt := promptEnvVarValue
	origCreate := createOpenAIAgentsSession
	t.Cleanup(func() {
		promptEnvVarValue = prevPrompt
		createOpenAIAgentsSession = origCreate
		_ = os.Unsetenv(openAIAPIKeyEnv)
	})

	promptEnvVarValue = func(name string) (string, error) {
		assert.Equal(t, openAIAPIKeyEnv, name)
		return "sk-prompted", nil
	}
	createOpenAIAgentsSession = func(ctx context.Context, apiKey string, body json.RawMessage) (*openAIAgentsSession, error) {
		assert.Equal(t, "sk-prompted", apiKey)
		return &openAIAgentsSession{ID: "oa_1", EnvironmentID: "env_1"}, nil
	}

	const manifest = `agent: codex-agentapi
env:
  CODEX_ENVIRONMENT_ID: ${ENV_ID}
  CODEX_API_KEY: ${OPENAI_API_KEY}
config:
  agent:
    model: gpt-5.6-sol
`
	id, overlay, err := prepareOpenAISandboxStart(context.Background(), []byte(manifest))
	require.NoError(t, err)
	assert.Equal(t, "oa_1", id)
	assert.Equal(t, "env_1", overlay["ENV_ID"])
	assert.Equal(t, "sk-prompted", os.Getenv(openAIAPIKeyEnv))
}

func TestFindMissingManifestEnvRefs(t *testing.T) {
	t.Setenv("DOCTL_TEST_SET", "1")
	missing := findMissingManifestEnvRefs(
		[]byte("a: ${DOCTL_TEST_SET}\nb: ${DOCTL_TEST_MISS}\nc: $${DOCTL_TEST_MISS}\nd: ${DOCTL_TEST_MISS}\n"),
		os.LookupEnv,
	)
	assert.Equal(t, []string{"DOCTL_TEST_MISS"}, missing)
}

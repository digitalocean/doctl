package commands

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/yaml"
)

func TestManifestSecretsFlatAndLegacy(t *testing.T) {
	var flat map[string]any
	require.NoError(t, yaml.Unmarshal([]byte(`agent: opencode
secrets:
  OPENAI_API_KEY: {source: tenantSecret, provider: openai}
  GITHUB_OAUTH: oauth/github
  INLINE: sk-typed
`), &flat))
	assert.Equal(t, []manifestSlot{
		{Name: "GITHUB_OAUTH", Source: "oauth", Provider: "github"},
		{Name: "INLINE", HasValue: true},
		{Name: "OPENAI_API_KEY", Source: "tenantSecret", Provider: "openai"},
	}, manifestSecrets(flat))

	setManifestSecretValue(flat, "OPENAI_API_KEY", "sk-new")
	assert.Equal(t, "sk-new", flat["secrets"].(map[string]any)["OPENAI_API_KEY"].(map[string]any)["value"])

	var legacy map[string]any
	require.NoError(t, yaml.Unmarshal([]byte(`apiVersion: agents.digitalocean.com/v1alpha1
kind: Agent
spec:
  secrets:
    - {name: OPENAI_API_KEY, source: tenantSecret, provider: openai}
`), &legacy))
	require.Len(t, manifestSecrets(legacy), 1)
	setManifestSecretValue(legacy, "OPENAI_API_KEY", "sk-new")
	assert.True(t, manifestSecrets(legacy)[0].HasValue)
}

func TestReusedSlotWithValue(t *testing.T) {
	spec := []byte("agent: opencode\nsecrets:\n  A: {source: tenantSecret, value: x}\n  B: {source: tenantSecret}\n")
	assert.Equal(t, "A", reusedSlotWithValue(spec, []string{"A"}), "listed and valued is a local error")
	assert.Equal(t, "", reusedSlotWithValue(spec, []string{"B"}))
}

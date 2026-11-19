package commands

import (
	"strings"
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

// The "Changed" row of the confirm summary. A legacy envelope keeps the model
// and egress under spec, so an edit there must show up. A secret value typed
// after the edit must not, since "New values" lists it.
func TestChangedManifestKeys(t *testing.T) {
	parse := func(s string) map[string]any {
		var doc map[string]any
		require.NoError(t, yaml.Unmarshal([]byte(s), &doc))
		return doc
	}
	legacy := `apiVersion: agents.digitalocean.com/v1alpha1
kind: Agent
metadata: {name: reviewer}
spec:
  agent: opencode
  model: gpt-5
  egress: {allow: [api.openai.com]}
  secrets:
    - {name: OPENAI_API_KEY, source: tenantSecret}
`
	source, err := manifestYAML(parse(legacy))
	require.NoError(t, err)

	unedited := parse(legacy)
	setManifestSecretValue(unedited, "OPENAI_API_KEY", "sk-new")
	assert.Empty(t, changedManifestKeys(source, unedited))

	edited := parse(strings.NewReplacer("gpt-5", "gpt-5-mini", "[api.openai.com]", "[api.openai.com, github.com]").Replace(legacy))
	setManifestSecretValue(edited, "OPENAI_API_KEY", "sk-new")
	assert.Equal(t, []string{"spec.egress", "spec.model"}, changedManifestKeys(source, edited))

	flat := "agent: opencode\nmodel: gpt-5\nsecrets:\n  OPENAI_API_KEY: {source: tenantSecret}\n"
	source, err = manifestYAML(parse(flat))
	require.NoError(t, err)
	edited = parse(strings.Replace(flat, "gpt-5", "gpt-5-mini", 1))
	setManifestSecretValue(edited, "OPENAI_API_KEY", "sk-new")
	assert.Equal(t, []string{"model"}, changedManifestKeys(source, edited))
}

func TestReusedSlotWithValue(t *testing.T) {
	spec := []byte("agent: opencode\nsecrets:\n  A: {source: tenantSecret, value: x}\n  B: {source: tenantSecret}\n")
	assert.Equal(t, "A", reusedSlotWithValue(spec, []string{"A"}), "listed and valued is a local error")
	assert.Equal(t, "", reusedSlotWithValue(spec, []string{"B"}))
}

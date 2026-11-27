package commands

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"
	"testing"

	"github.com/digitalocean/doctl/commands/charm/confirm"
	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
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

// fakeCreateFromPrompts answers create-from's questions: the prefilled name,
// no to the editor, choices[slot] for keep or new, a typed value for each new
// secret, and confirmCreate at the final confirm.
func fakeCreateFromPrompts(choices map[string]string, confirmCreate confirm.Choice) createFromPrompts {
	return createFromPrompts{
		name: func(initial string, validate func(string) error) (string, error) {
			return initial, validate(initial)
		},
		confirm: func(text string, _ confirm.Choice) (confirm.Choice, error) {
			if strings.HasPrefix(text, "Create ") {
				return confirmCreate, nil
			}
			return confirm.No, nil
		},
		choose: func(label string, _ []string) (string, error) {
			return choices[strings.TrimSuffix(strings.Fields(label)[0], ":")], nil
		},
		secret: func(label string) (string, error) {
			return "typed-" + strings.TrimSuffix(strings.Fields(label)[3], ":"), nil
		},
	}
}

func createFromSource() *godo.HostedAgentConfig {
	on := true
	return &godo.HostedAgentConfig{
		ID:       "cfg_src",
		Name:     "reviewer",
		Insights: &godo.HostedAgentInsightsOptIn{Metrics: &on},
		Manifest: []byte(`{"agent":"opencode","secrets":{` +
			`"OPENAI_API_KEY":{"source":"tenantSecret","provider":"openai"},` +
			`"ANTHROPIC_API_KEY":{"source":"tenantSecret"}}}`),
		Credentials: []godo.HostedAgentConfigCredentialSlot{
			{Name: "OPENAI_API_KEY", Source: "tenantSecret", Provider: "openai"},
			{Name: "ANTHROPIC_API_KEY", Source: "tenantSecret"},
		},
	}
}

// Keep sends the slot in reuse_secrets with no value in the manifest; New puts
// the typed value in the manifest. source_config_id goes with reuse_secrets
// only, and Insights are copied either way.
func TestCreateAgentConfigFrom_KeepOrNewPerSecret(t *testing.T) {
	for _, tc := range []struct {
		name       string
		choices    map[string]string
		wantReuse  []string
		wantSource string
		wantTyped  map[string]bool
	}{
		{
			name:       "keep one, new the other",
			choices:    map[string]string{"OPENAI_API_KEY": secretChoiceKeep, "ANTHROPIC_API_KEY": secretChoiceNew},
			wantReuse:  []string{"OPENAI_API_KEY"},
			wantSource: "cfg_src",
			wantTyped:  map[string]bool{"ANTHROPIC_API_KEY": true},
		},
		{
			name:      "new for every secret",
			choices:   map[string]string{"OPENAI_API_KEY": secretChoiceNew, "ANTHROPIC_API_KEY": secretChoiceNew},
			wantTyped: map[string]bool{"ANTHROPIC_API_KEY": true, "OPENAI_API_KEY": true},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
				config.Out = io.Discard
				src := createFromSource()
				tm.hostedAgents.EXPECT().GetAgentConfig("cfg_src").Return(src, nil)
				tm.hostedAgents.EXPECT().CreateAgentConfig(gomock.Any(), gomock.Any()).
					DoAndReturn(func(req *godo.HostedAgentConfigCreateRequest, _ bool) (*godo.HostedAgentConfig, error) {
						assert.Equal(t, "reviewer-2", req.Name)
						assert.Equal(t, src.Insights, req.Insights)
						assert.Equal(t, tc.wantSource, req.SourceConfigID)
						assert.Equal(t, tc.wantReuse, req.ReuseSecrets)
						var doc map[string]any
						require.NoError(t, yaml.Unmarshal([]byte(req.ManifestYAML), &doc))
						for _, slot := range manifestSecrets(doc) {
							assert.Equal(t, tc.wantTyped[slot.Name], slot.HasValue, slot.Name)
						}
						return &godo.HostedAgentConfig{ID: "cfg_new", Name: req.Name}, nil
					})
				require.NoError(t, createAgentConfigFrom(config, fakeCreateFromPrompts(tc.choices, confirm.Yes), "cfg_src", ""))
			})
		})
	}
}

// Declining the final confirm creates nothing.
func TestCreateAgentConfigFrom_DeclineCreatesNothing(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Out = io.Discard
		tm.hostedAgents.EXPECT().GetAgentConfig("cfg_src").Return(createFromSource(), nil)
		choices := map[string]string{"OPENAI_API_KEY": secretChoiceKeep, "ANTHROPIC_API_KEY": secretChoiceKeep}
		err := createAgentConfigFrom(config, fakeCreateFromPrompts(choices, confirm.No), "cfg_src", "")
		assert.True(t, isPromptCanceled(err), "got %v", err)
	})
}

// With -o json, stdout is the created config as one JSON document: the source
// card, manifest and plan go to stderr.
func TestCreateAgentConfigFrom_JSONOutputIsOnlyJSON(t *testing.T) {
	prev := Output
	Output = "json"
	t.Cleanup(func() { Output = prev })

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		var out bytes.Buffer
		config.Out = &out
		tm.hostedAgents.EXPECT().GetAgentConfig("cfg_src").Return(createFromSource(), nil)
		tm.hostedAgents.EXPECT().CreateAgentConfig(gomock.Any(), gomock.Any()).
			Return(&godo.HostedAgentConfig{ID: "cfg_new", Name: "reviewer-2"}, nil)
		choices := map[string]string{"OPENAI_API_KEY": secretChoiceKeep, "ANTHROPIC_API_KEY": secretChoiceKeep}
		require.NoError(t, createAgentConfigFrom(config, fakeCreateFromPrompts(choices, confirm.Yes), "cfg_src", ""))

		assert.True(t, json.Valid(out.Bytes()), "stdout is not JSON:\n%s", out.String())
		assert.Contains(t, out.String(), "cfg_new")
	})
}

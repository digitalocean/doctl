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
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// generateFor runs the flag-only path — the same one --no-interactive and any
// agent driving doctl take — and returns the encoded manifest.
func generateFor(t *testing.T, a *generateAnswers) string {
	t.Helper()
	require.NoError(t, applyGenerateDerivations(a))
	m, err := buildGeneratedManifest(a)
	require.NoError(t, err)
	out, err := encodeGeneratedManifest(m, generateFormatYAML)
	require.NoError(t, err)
	return string(out)
}

func TestGenerate_FlagOnlyGolden(t *testing.T) {
	a := &generateAnswers{
		Harness:           "claude-code",
		Name:              "payments-reviewer",
		Repos:             []string{"acme/payments-service"},
		GitHubAccess:      true,
		ToolsPreset:       generateToolsPresetDefault,
		PermissionDefault: generateDefaultPermission,
	}

	const want = `name: payments-reviewer
agent: claude-code
repos:
- acme/payments-service
secrets:
  ANTHROPIC_API_KEY: ${ANTHROPIC_API_KEY}
  GITHUB_TOKEN: oauth/github
egress:
- github.com
- api.github.com
tools:
- do.actions:
  - web_search
  - execute_code
  - toolbelt:read-only
permissions:
  default: ask
`
	assert.Equal(t, want, generateFor(t, a))
}

// The generator's whole job is emitting a manifest the platform accepts, so
// every harness must round-trip through the same validator `validate` uses
// without a single warning. Warnings here are how a stale env mapping shows up.
func TestGenerate_RoundTripsValidatorCleanly(t *testing.T) {
	providers := []struct {
		provider string
		url      string
		model    string
	}{
		{provider: inferenceProviderNative},
		{provider: inferenceProviderNative, model: "some-model"},
		{provider: inferenceProviderDO, model: "do-catalog-slug"},
		{provider: inferenceProviderCustom, url: "https://llm.example.com/v1", model: "served-model"},
	}

	for _, h := range generateHarnessCatalog {
		for _, p := range providers {
			name := h.ID + "/" + p.provider
			if p.model != "" && p.provider == inferenceProviderNative {
				name += "-with-model"
			}
			t.Run(name, func(t *testing.T) {
				a := &generateAnswers{
					Harness:           h.ID,
					Name:              "rt",
					InferenceProvider: p.provider,
					InferenceURL:      p.url,
					Model:             p.model,
					ToolsPreset:       generateToolsPresetDefault,
					PermissionDefault: generateDefaultPermission,
				}
				if h.ID == "custom" {
					a.Image = "registry.example.com/agent@sha256:" + strings.Repeat("a", 64)
					a.Entrypoint = []string{"/bin/agent"}
				}
				if h.Inference == nil && p.provider != inferenceProviderNative {
					// Harnesses that never call a model endpoint reject the
					// choice outright; that path is asserted separately.
					require.Error(t, applyGenerateDerivations(a))
					return
				}
				if h.Inference != nil && h.Inference.NativeOnly && p.provider != inferenceProviderNative {
					require.Error(t, applyGenerateDerivations(a))
					return
				}

				v := validateAgentManifest([]byte(generateFor(t, a)))
				assert.True(t, v.ok(), "errors=%v", v.Errors)
				assert.Empty(t, v.Warnings)
			})
		}
	}
}

func TestGenerate_InferenceWiring(t *testing.T) {
	tests := []struct {
		name     string
		answers  generateAnswers
		wantEnv  map[string]string
		wantKey  string
		wantHost string
	}{
		{
			// Native with no model is the Enter-through default: one vendor key
			// and nothing else, so the agent uses the model it ships with.
			name:    "native declares only the vendor key",
			answers: generateAnswers{Harness: "claude-code", InferenceProvider: inferenceProviderNative},
			wantEnv: nil,
			wantKey: "ANTHROPIC_API_KEY",
		},
		{
			// claude-code has no model env of its own, so a pinned model has to
			// travel as HARNESS_INFERENCE_MODEL; ANTHROPIC_MODEL is what the
			// CLI itself reads.
			name:    "native claude-code pins a model through both vars",
			answers: generateAnswers{Harness: "claude-code", InferenceProvider: inferenceProviderNative, Model: "claude-sonnet-4-5"},
			wantEnv: map[string]string{
				"ANTHROPIC_MODEL":        "claude-sonnet-4-5",
				harnessInferenceModelEnv: "claude-sonnet-4-5",
			},
			wantKey: "ANTHROPIC_API_KEY",
		},
		{
			// codex reads MODEL, so the native path uses it rather than the
			// harness var.
			name:    "native codex uses its own model var",
			answers: generateAnswers{Harness: "codex", InferenceProvider: inferenceProviderNative, Model: "gpt-5-codex"},
			wantEnv: map[string]string{"MODEL": "gpt-5-codex"},
			wantKey: "OPENAI_API_KEY",
		},
		{
			// Endpoint routing must not declare the vendor key: the runtime
			// prefers it and would ignore the endpoint entirely.
			name:    "do inference swaps the key and sets the endpoint",
			answers: generateAnswers{Harness: "opencode", InferenceProvider: inferenceProviderDO, Model: "deepseek-v4-pro"},
			wantEnv: map[string]string{
				harnessInferenceBaseURLEnv: doInferenceBaseURL,
				harnessInferenceModelEnv:   "deepseek-v4-pro",
			},
			wantKey: harnessInferenceAPIKeyEnv,
		},
		{
			name:    "custom endpoint also opens egress for its host",
			answers: generateAnswers{Harness: "codex", InferenceProvider: inferenceProviderCustom, InferenceURL: "https://llm.internal.example.com/v1", Model: "served"},
			wantEnv: map[string]string{
				harnessInferenceBaseURLEnv: "https://llm.internal.example.com/v1",
				harnessInferenceModelEnv:   "served",
			},
			wantKey:  harnessInferenceAPIKeyEnv,
			wantHost: "llm.internal.example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := tt.answers
			require.NoError(t, applyGenerateDerivations(&a))

			assert.Equal(t, tt.wantEnv, a.Env)
			assert.Equal(t, []string{tt.wantKey}, a.secretNames())
			if tt.wantHost != "" {
				assert.Contains(t, a.Egress, tt.wantHost)
			} else {
				assert.Empty(t, a.Egress)
			}
			// The vendor key and the endpoint key are never both declared.
			if tt.wantKey == harnessInferenceAPIKeyEnv {
				harness, _ := lookupGenerateHarness(a.Harness)
				assert.False(t, a.hasSecret(harness.Inference.NativeKeyEnv))
			}
		})
	}
}

// DigitalOcean's endpoint publishes no default model, and an arbitrary endpoint
// tells doctl nothing, so both must refuse rather than emit a manifest whose
// first turn fails.
func TestGenerate_InferenceRejectsUnusableCombinations(t *testing.T) {
	tests := []struct {
		name    string
		answers generateAnswers
		wantErr string
	}{
		{
			name:    "do inference without a model",
			answers: generateAnswers{Harness: "opencode", InferenceProvider: inferenceProviderDO},
			wantErr: "needs --model",
		},
		{
			name:    "custom endpoint without a url",
			answers: generateAnswers{Harness: "opencode", InferenceProvider: inferenceProviderCustom, Model: "m"},
			wantErr: "needs --inference-url",
		},
		{
			name:    "custom endpoint without a model",
			answers: generateAnswers{Harness: "opencode", InferenceProvider: inferenceProviderCustom, InferenceURL: "https://llm.example.com/v1"},
			wantErr: "needs --model",
		},
		{
			name:    "plaintext endpoint",
			answers: generateAnswers{Harness: "opencode", InferenceProvider: inferenceProviderCustom, InferenceURL: "http://llm.example.com/v1", Model: "m"},
			wantErr: "must be an https:// URL",
		},
		{
			name:    "unknown provider",
			answers: generateAnswers{Harness: "opencode", InferenceProvider: "bogus"},
			wantErr: "must be native, do, or custom",
		},
		{
			// Cursor's agent talks to Cursor, so an endpoint override would be
			// silently ignored rather than honored.
			name:    "endpoint override on a fixed-backend harness",
			answers: generateAnswers{Harness: "cursor", InferenceProvider: inferenceProviderDO, Model: "m"},
			wantErr: "only supports native inference",
		},
		{
			// The loop runs at OpenAI, so the sandbox never calls a model.
			name:    "endpoint override on an externally-hosted loop",
			answers: generateAnswers{Harness: "codex-agentapi", InferenceProvider: inferenceProviderDO, Model: "m"},
			wantErr: "does not apply",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := tt.answers
			err := applyGenerateDerivations(&a)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
			// doctl's error beautifier rewrites any message naming the vendor
			// into an unrelated "check your API key" card, so these must not.
			assert.NotContains(t, strings.ToLower(err.Error()), "openai")
		})
	}
}

// --no-interactive exists to be the wizard's Enter key, so its defaults have to
// come from the same place rather than a second, barer set.
func TestGenerate_EnterDefaultsMatchWizardDefaults(t *testing.T) {
	restore := creationClock
	creationClock = func() time.Time { return time.Date(2026, 9, 2, 11, 48, 0, 0, time.UTC) }
	defer func() { creationClock = restore }()

	a := &generateAnswers{Harness: "claude-code"}
	applyGenerateEnterDefaults(a)

	assert.Equal(t, "claude-code-0902-1148", a.Name)
	assert.Equal(t, inferenceProviderNative, a.InferenceProvider)
	assert.Equal(t, generateToolsPresetDefault, a.ToolsPreset)
	assert.Equal(t, generateDefaultPermission, a.PermissionDefault)
	// Questions whose default is "nothing" stay unset, and nothing is inferred
	// from the working directory.
	assert.Empty(t, a.Repos)
	assert.Empty(t, a.Model)
	assert.Empty(t, a.MCPServers)
	assert.Empty(t, a.Skills)
}

// Running derivations twice is what lets the wizard show a truthful review card
// and then write the file, so it must not double up slots or hosts.
func TestGenerate_DerivationsAreIdempotent(t *testing.T) {
	a := &generateAnswers{
		Harness:           "codex",
		Name:              "twice",
		Repos:             []string{"acme/api"},
		GitHubAccess:      true,
		InferenceProvider: inferenceProviderCustom,
		InferenceURL:      "https://llm.example.com/v1",
		Model:             "m",
	}
	require.NoError(t, applyGenerateDerivations(a))
	first := generateFor(t, a)
	assert.Equal(t, first, generateFor(t, a))
}

// Highlighting must never reach a pipe: `generate > agents.yaml` and
// `generate | config create --spec -` have to receive the bytes that were
// validated, not a decorated copy of them.
func TestHighlightManifest_PassesBytesThroughUnstyled(t *testing.T) {
	restore := stylingEnabled
	stylingEnabled = false
	defer func() { stylingEnabled = restore }()

	const manifest = `name: reviewer
agent: claude-code
secrets:
  ANTHROPIC_API_KEY: ${ANTHROPIC_API_KEY}
`
	assert.Equal(t, manifest, highlightManifest([]byte(manifest)))
}

// With styling on, the added bytes must be ANSI and nothing else: the document
// has to survive being copied out of the terminal, which keeps plain text only.
func TestHighlightManifest_AddsOnlyANSI(t *testing.T) {
	restore := stylingEnabled
	stylingEnabled = true
	// lipgloss decides separately whether to emit escapes, and under `go test`
	// it sees no terminal (and honors NO_COLOR), so the profile is forced or
	// this asserts nothing.
	restoreProfile := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	defer func() {
		stylingEnabled = restore
		lipgloss.SetColorProfile(restoreProfile)
	}()

	const manifest = `name: reviewer
agent: claude-code
repos:
- acme/api
env:
  HARNESS_INFERENCE_BASE_URL: https://inference.do-ai.run/v1
secrets:
  ANTHROPIC_API_KEY: ${ANTHROPIC_API_KEY}
tools:
- name: linear
  url: https://mcp.linear.app/sse
`
	styled := highlightManifest([]byte(manifest))
	require.NotEqual(t, manifest, styled, "expected styling to be applied")

	ansi := regexp.MustCompile("\x1b\\[[0-9;]*m")
	assert.Equal(t, manifest, ansi.ReplaceAllString(styled, ""))
}

func TestSplitManifestKey(t *testing.T) {
	tests := []struct {
		node  string
		key   string
		value string
		ok    bool
	}{
		{node: "name: reviewer", key: "name", value: "reviewer", ok: true},
		{node: "secrets:", key: "secrets", ok: true},
		// The colon in a URL must not be mistaken for the key separator, or
		// half the scheme would be colored as a key.
		{node: "url: https://mcp.linear.app/sse", key: "url", value: "https://mcp.linear.app/sse", ok: true},
		{node: `"name": "as-json",`, key: `"name"`, value: `"as-json",`, ok: true},
		// A colon inside a quoted scalar is not a separator either.
		{node: `"toolbelt:read-only",`, ok: false},
		{node: "web_search", ok: false},
	}
	for _, tt := range tests {
		t.Run(tt.node, func(t *testing.T) {
			key, _, value, ok := splitManifestKey(tt.node)
			assert.Equal(t, tt.ok, ok)
			assert.Equal(t, tt.key, key)
			assert.Equal(t, tt.value, value)
		})
	}
}

func TestGenerate_JSONFormat(t *testing.T) {
	a := &generateAnswers{Harness: "opencode", Name: "as-json"}
	require.NoError(t, applyGenerateDerivations(a))
	m, err := buildGeneratedManifest(a)
	require.NoError(t, err)

	out, err := encodeGeneratedManifest(m, generateFormatJSON)
	require.NoError(t, err)
	assert.Equal(t, `{
  "name": "as-json",
  "agent": "opencode",
  "secrets": {
    "OPENAI_API_KEY": "${OPENAI_API_KEY}"
  }
}
`, string(out))

	_, err = encodeGeneratedManifest(m, "toml")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must be yaml or json")
}

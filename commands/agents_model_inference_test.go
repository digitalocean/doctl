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
	"testing"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/doctl/internal/ui"
	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestBuildHarnessManifest_ModelUsesDOInference(t *testing.T) {
	cases := []struct {
		harness string
		keyEnv  string
	}{
		{"claude-code", "ANTHROPIC_API_KEY"},
		{"codex", "OPENAI_API_KEY"},
	}
	for _, tc := range cases {
		t.Run(tc.harness+" model without vendor secret drops the slot", func(t *testing.T) {
			raw, err := buildHarnessManifest(harnessManifestOpts{harness: tc.harness, model: "kimi-k3"})
			require.NoError(t, err)
			assert.NotContains(t, string(raw), tc.keyEnv)
			assert.NotContains(t, string(raw), "secrets")
			assert.Contains(t, string(raw), "HARNESS_INFERENCE_MODEL: kimi-k3")
			assert.True(t, needsManagedModelAccessKey(raw))
		})
		t.Run(tc.harness+" model with vendor secret keeps the slot", func(t *testing.T) {
			raw, err := buildHarnessManifest(harnessManifestOpts{
				harness: tc.harness, model: "kimi-k3",
				secrets: map[string]string{tc.keyEnv: "x"},
			})
			require.NoError(t, err)
			assert.Contains(t, string(raw), tc.keyEnv+": ${"+tc.keyEnv+"}")
		})
		t.Run(tc.harness+" model with an unrelated secret still drops the slot", func(t *testing.T) {
			raw, err := buildHarnessManifest(harnessManifestOpts{
				harness: tc.harness, model: "kimi-k3",
				secrets: map[string]string{"OTHER": "x"},
			})
			require.NoError(t, err)
			assert.NotContains(t, string(raw), tc.keyEnv)
		})
		t.Run(tc.harness+" without model is unchanged", func(t *testing.T) {
			raw, err := buildHarnessManifest(harnessManifestOpts{harness: tc.harness})
			require.NoError(t, err)
			assert.Contains(t, string(raw), tc.keyEnv+": ${"+tc.keyEnv+"}")
			assert.NotContains(t, string(raw), "HARNESS_INFERENCE_MODEL")
		})
	}

	t.Run("codex-agentapi keeps its key with a model", func(t *testing.T) {
		raw, err := buildHarnessManifest(harnessManifestOpts{harness: "codex-agentapi", model: "kimi-k3"})
		require.NoError(t, err)
		assert.Contains(t, string(raw), "CODEX_API_KEY: ${OPENAI_API_KEY}")
	})
}

// stdout and stderr captured separately, so a test can assert the note never
// reaches machine-readable output.
func captureUIEnv(t *testing.T) (stdout, stderr *bytes.Buffer) {
	t.Helper()
	stdout, stderr = &bytes.Buffer{}, &bytes.Buffer{}
	withUIEnv(t, ui.Detect(stdout, stderr))
	return stdout, stderr
}

func TestRunAgentsCreate_ModelUsesManagedKey(t *testing.T) {
	cases := []struct {
		harness string
		keyEnv  string
		vendor  string
	}{
		{"claude-code", "ANTHROPIC_API_KEY", "Anthropic"},
		{"codex", "OPENAI_API_KEY", "OpenAI"},
	}
	for _, tc := range cases {
		for _, exported := range []bool{false, true} {
			name := tc.harness + " key unset"
			if exported {
				name = tc.harness + " key exported"
			}
			t.Run(name, func(t *testing.T) {
				t.Setenv(tc.keyEnv, "sk-from-shell")
				if !exported {
					require.NoError(t, os.Unsetenv(tc.keyEnv))
				}
				prevOutput := Output
				Output = "json"
				t.Cleanup(func() { Output = prevOutput })
				stdout, stderr := captureUIEnv(t)

				withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
					tm.hostedAgents.EXPECT().
						CreateSessionFromManifestWithManagedKey(gomock.Any(), nil).
						DoAndReturn(func(manifest []byte, _ *godo.HostedAgentManifestCreateOptions) (*do.HostedAgentSession, error) {
							assert.NotContains(t, string(manifest), tc.keyEnv)
							assert.NotContains(t, string(manifest), "sk-from-shell")
							assert.Contains(t, string(manifest), "HARNESS_INFERENCE_MODEL: kimi-k3")
							return nil, assertCalledErr
						})

					var out bytes.Buffer
					config.Out = &out
					config.Doit.Set(config.NS, doctl.ArgAgentHarness, tc.harness)
					config.Doit.Set(config.NS, doctl.ArgInferenceModel, "kimi-k3")

					require.ErrorIs(t, RunAgentsCreate(config), assertCalledErr)
					assert.NotContains(t, out.String(), "DigitalOcean Inference")
				})
				assert.NotContains(t, stdout.String(), "DigitalOcean Inference")

				note := "Using DigitalOcean Inference for kimi-k3. " + tc.keyEnv + " in your environment is not used; to use your own " + tc.vendor + " key, pass --secret " + tc.keyEnv + "=... (or =@file)."
				if exported {
					assert.Equal(t, 1, bytes.Count(stderr.Bytes(), []byte(note)), "stderr=%q", stderr.String())
				} else {
					assert.NotContains(t, stderr.String(), "Using DigitalOcean Inference")
				}
			})
		}
	}
}

func TestRunAgentsCreate_ModelWithVendorSecretSkipsManagedKey(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "sk-from-shell")
	_, stderr := captureUIEnv(t)
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.hostedAgents.EXPECT().
			CreateSessionFromManifest(gomock.Any(), nil).
			DoAndReturn(func(manifest []byte, _ *godo.HostedAgentManifestCreateOptions) (*do.HostedAgentSession, error) {
				assert.Contains(t, string(manifest), "ANTHROPIC_API_KEY: sk-own")
				assert.NotContains(t, string(manifest), "sk-from-shell")
				return nil, assertCalledErr
			})
		config.Doit.Set(config.NS, doctl.ArgAgentHarness, "claude-code")
		config.Doit.Set(config.NS, doctl.ArgInferenceModel, "kimi-k3")
		config.Doit.Set(config.NS, doctl.ArgAgentSecret, []string{"ANTHROPIC_API_KEY=sk-own"})
		require.ErrorIs(t, RunAgentsCreate(config), assertCalledErr)
	})
	assert.NotContains(t, stderr.String(), "Using DigitalOcean Inference")
}

// The other vendor's key is not this harness's key: the harness keeps its own
// slot, no key is issued, and no note claims DigitalOcean Inference.
func TestRunAgentsCreate_ModelWithOtherVendorSecretKeepsOwnSlot(t *testing.T) {
	cases := []struct{ harness, own, other string }{
		{"claude-code", "ANTHROPIC_API_KEY", "OPENAI_API_KEY"},
		{"codex", "OPENAI_API_KEY", "ANTHROPIC_API_KEY"},
	}
	for _, tc := range cases {
		t.Run(tc.harness, func(t *testing.T) {
			t.Setenv(tc.own, "sk-from-shell")
			_, stderr := captureUIEnv(t)
			withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
				tm.hostedAgents.EXPECT().
					CreateSessionFromManifest(gomock.Any(), nil).
					DoAndReturn(func(manifest []byte, _ *godo.HostedAgentManifestCreateOptions) (*do.HostedAgentSession, error) {
						assert.Contains(t, string(manifest), tc.own+": sk-from-shell")
						assert.Contains(t, string(manifest), tc.other+": sk-other")
						return nil, assertCalledErr
					})
				config.Doit.Set(config.NS, doctl.ArgAgentHarness, tc.harness)
				config.Doit.Set(config.NS, doctl.ArgInferenceModel, "kimi-k3")
				config.Doit.Set(config.NS, doctl.ArgAgentSecret, []string{tc.other + "=sk-other"})
				require.ErrorIs(t, RunAgentsCreate(config), assertCalledErr)
			})
			assert.NotContains(t, stderr.String(), "Using DigitalOcean Inference")
		})
	}
}

func TestRunAgentsCreate_ModelWithVendorSecretSkipsManagedKeyCodex(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "sk-from-shell")
	_, stderr := captureUIEnv(t)
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.hostedAgents.EXPECT().
			CreateSessionFromManifest(gomock.Any(), nil).
			DoAndReturn(func(manifest []byte, _ *godo.HostedAgentManifestCreateOptions) (*do.HostedAgentSession, error) {
				assert.Contains(t, string(manifest), "OPENAI_API_KEY: sk-own")
				assert.NotContains(t, string(manifest), "sk-from-shell")
				return nil, assertCalledErr
			})
		config.Doit.Set(config.NS, doctl.ArgAgentHarness, "codex")
		config.Doit.Set(config.NS, doctl.ArgInferenceModel, "kimi-k3")
		config.Doit.Set(config.NS, doctl.ArgAgentSecret, []string{"OPENAI_API_KEY=sk-own"})
		require.ErrorIs(t, RunAgentsCreate(config), assertCalledErr)
	})
	assert.NotContains(t, stderr.String(), "Using DigitalOcean Inference")
}

// The user's own DigitalOcean key: no vendor slot and nothing issued.
func TestRunAgentsCreate_ModelWithHarnessKeySecret(t *testing.T) {
	for _, harness := range []string{"claude-code", "codex"} {
		t.Run(harness, func(t *testing.T) {
			t.Setenv("ANTHROPIC_API_KEY", "")
			os.Unsetenv("ANTHROPIC_API_KEY")
			os.Unsetenv("OPENAI_API_KEY")
			captureUIEnv(t)
			withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
				tm.hostedAgents.EXPECT().
					CreateSessionFromManifest(gomock.Any(), nil).
					DoAndReturn(func(manifest []byte, _ *godo.HostedAgentManifestCreateOptions) (*do.HostedAgentSession, error) {
						assert.NotContains(t, string(manifest), "ANTHROPIC_API_KEY")
						assert.NotContains(t, string(manifest), "OPENAI_API_KEY")
						assert.Contains(t, string(manifest), "HARNESS_INFERENCE_API_KEY: sk-do")
						return nil, assertCalledErr
					})
				config.Doit.Set(config.NS, doctl.ArgAgentHarness, harness)
				config.Doit.Set(config.NS, doctl.ArgInferenceModel, "kimi-k3")
				config.Doit.Set(config.NS, doctl.ArgAgentSecret, []string{"HARNESS_INFERENCE_API_KEY=sk-do"})
				require.ErrorIs(t, RunAgentsCreate(config), assertCalledErr)
			})
		})
	}
}

// launch shares resolveAgentCreationSource with create, so resolving is enough.
func TestResolveAgentCreationSource_ModelDropsVendorSlot(t *testing.T) {
	os.Unsetenv("ANTHROPIC_API_KEY")
	captureUIEnv(t)
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Doit.Set(config.NS, doctl.ArgAgentHarness, "claude-code")
		config.Doit.Set(config.NS, doctl.ArgInferenceModel, "kimi-k3")
		src, err := resolveAgentCreationSource(config)
		require.NoError(t, err)
		assert.NotContains(t, string(src.manifest), "ANTHROPIC_API_KEY")
		assert.True(t, needsManagedModelAccessKey(src.manifest))
	})
}

func TestRunAgentsCreate_DryRunModelNotesManagedKey(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "")
	require.NoError(t, os.Unsetenv("ANTHROPIC_API_KEY"))
	_, stderr := captureUIEnv(t)
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		var out bytes.Buffer
		config.Out = &out
		config.Doit.Set(config.NS, doctl.ArgAgentHarness, "claude-code")
		config.Doit.Set(config.NS, doctl.ArgInferenceModel, "kimi-k3")
		config.Doit.Set(config.NS, doctl.ArgAgentDryRun, true)
		require.NoError(t, RunAgentsCreate(config))
		assert.NotContains(t, out.String(), "ANTHROPIC_API_KEY")
		assert.Contains(t, out.String(), "HARNESS_INFERENCE_MODEL: kimi-k3")
	})
	assert.Equal(t, 1, bytes.Count(stderr.Bytes(), []byte("A DigitalOcean Inference key will be requested")))
}

func TestValidateAgentManifest_ClaudeCodeManagedKeyManifestDoesNotWarn(t *testing.T) {
	raw, err := buildHarnessManifest(harnessManifestOpts{harness: "claude-code", model: "kimi-k3"})
	require.NoError(t, err)
	v := validateAgentManifest(raw)
	assert.True(t, v.ok(), "errors=%v", v.Errors)
	assert.NotContains(t, joinStrings(v.Warnings), "HARNESS_INFERENCE_MODEL")

	const withNative = `agent: claude-code
env:
  HARNESS_INFERENCE_MODEL: kimi-k3
secrets:
  ANTHROPIC_API_KEY: sk-ant-key
`
	v = validateAgentManifest([]byte(withNative))
	assert.Contains(t, joinStrings(v.Warnings), "HARNESS_INFERENCE_MODEL")
}

func TestValidateAgentManifest_ClaudeCodeOpenAIKeyOnlyStillWarns(t *testing.T) {
	const manifest = `agent: claude-code
env:
  HARNESS_INFERENCE_MODEL: kimi-k3
secrets:
  OPENAI_API_KEY: sk-openai
`
	v := validateAgentManifest([]byte(manifest))
	assert.Contains(t, joinStrings(v.Warnings), "HARNESS_INFERENCE_MODEL")
}

func TestPrintResolvedManifest_NoKeyNoteForSpecWithPlaceholder(t *testing.T) {
	os.Unsetenv("ANTHROPIC_API_KEY")
	_, stderr := captureUIEnv(t)
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		var out bytes.Buffer
		config.Out = &out
		src := &agentCreationSource{manifest: []byte("harness: claude-code\nenv:\n  HARNESS_INFERENCE_MODEL: kimi-k3\nsecrets:\n  ANTHROPIC_API_KEY: ${ANTHROPIC_API_KEY}\n")}
		require.NoError(t, printResolvedManifest(config, src))
	})
	assert.NotContains(t, stderr.String(), "will be requested")
}

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
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/digitalocean/doctl"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateAgentManifest_FlatOK(t *testing.T) {
	v := validateAgentManifest([]byte("agent: opencode\n"))
	assert.True(t, v.ok())
	assert.Empty(t, v.Errors)
}

func TestValidateAgentManifest_EnvelopeOK(t *testing.T) {
	v := validateAgentManifest([]byte(sampleManifest))
	assert.True(t, v.ok(), "errors=%v", v.Errors)
}

func TestValidateAgentManifest_FlatRequiresAgent(t *testing.T) {
	v := validateAgentManifest([]byte("env:\n  FOO: bar\n"))
	require.False(t, v.ok())
	assert.Contains(t, v.Errors[0], `"agent"`)
}

func TestValidateAgentManifest_EnvelopeRequiresRuntimeAdapter(t *testing.T) {
	const bad = `apiVersion: agents.digitalocean.com/v1alpha1
kind: Agent
spec:
  adapter: opencode
`
	v := validateAgentManifest([]byte(bad))
	require.False(t, v.ok())
	joined := bytes.Join(stringSliceToBytes(v.Errors), []byte("\n"))
	assert.Contains(t, string(joined), "spec.runtime.adapter")
}

func TestValidateAgentManifest_UnknownAdapter(t *testing.T) {
	v := validateAgentManifest([]byte("agent: not-a-real-adapter\n"))
	require.False(t, v.ok())
	assert.Contains(t, v.Errors[0], "not a known adapter")
}

func TestValidateAgentManifest_RemovedAdaptersRejected(t *testing.T) {
	for _, adapter := range []string{"cursor", "hermes", "cursor-cli"} {
		t.Run(adapter, func(t *testing.T) {
			v := validateAgentManifest([]byte("agent: " + adapter + "\n"))
			require.False(t, v.ok())
			assert.Contains(t, v.Errors[0], "not a known adapter")
			assert.Contains(t, v.Errors[0], adapter)
		})
	}
}

func TestValidateAgentManifest_CredentialPlaceholderNoWarn(t *testing.T) {
	const manifest = `agent: opencode
env:
  OPENAI_API_KEY: ${OPENAI_API_KEY}
`
	v := validateAgentManifest([]byte(manifest))
	assert.True(t, v.ok(), "errors=%v", v.Errors)
	assert.Empty(t, v.Warnings)
}

func TestValidateAgentManifest_CredentialInEnvWarns(t *testing.T) {
	const manifest = `agent: opencode
env:
  OPENAI_API_KEY: sk-proj-literal
`
	v := validateAgentManifest([]byte(manifest))
	assert.True(t, v.ok(), "errors=%v", v.Errors)
	require.NotEmpty(t, v.Warnings)
	assert.Contains(t, v.Warnings[0], "OPENAI_API_KEY")
	assert.Contains(t, v.Warnings[0], "secrets")
}

func TestValidateAgentManifest_SecretSlotCollisionWarns(t *testing.T) {
	const manifest = `agent: opencode
env:
  OPENAI_API_KEY: sk-proj-literal
secrets:
  - name: OPENAI_API_KEY
    source: tenantSecret
    value: sk-proj-literal
`
	v := validateAgentManifest([]byte(manifest))
	assert.True(t, v.ok(), "errors=%v", v.Errors)
	require.NotEmpty(t, v.Warnings)
	assert.Contains(t, v.Warnings[0], "OPENAI_API_KEY")
	assert.Contains(t, v.Warnings[0], "secrets")
}

func TestValidateAgentManifest_FlatSecretsMapCollisionWarns(t *testing.T) {
	const manifest = `agent: opencode
env:
  OPENAI_API_KEY: placeholder
secrets:
  OPENAI_API_KEY: ${OPENAI_API_KEY}
`
	v := validateAgentManifest([]byte(manifest))
	assert.True(t, v.ok(), "errors=%v", v.Errors)
	require.NotEmpty(t, v.Warnings)
	assert.Contains(t, v.Warnings[0], "OPENAI_API_KEY")
}

func TestReportDurableAgentManifestValidation_PromotesCredentialWarnings(t *testing.T) {
	const manifest = `agent: opencode
env:
  OPENAI_API_KEY: sk-proj-literal
`
	err := reportDurableAgentManifestValidation(validateAgentManifest([]byte(manifest)))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "OPENAI_API_KEY")
}

func TestValidateAgentManifest_ReservedEnvKey(t *testing.T) {
	const manifest = `agent: opencode
env:
  SESSION_ID: custom
`
	v := validateAgentManifest([]byte(manifest))
	require.False(t, v.ok())
	assert.Contains(t, v.Errors[0], "SESSION_ID")
	assert.Contains(t, v.Errors[0], "reserved")
}

func TestValidateAgentManifest_ModelVsHarnessInferenceWarns(t *testing.T) {
	const manifest = `agent: opencode
env:
  HARNESS_INFERENCE_BASE_URL: https://inference.do-ai.run/v1
  HARNESS_INFERENCE_API_KEY: sk-test
  MODEL: gpt-5.5
`
	v := validateAgentManifest([]byte(manifest))
	assert.True(t, v.ok(), "errors=%v", v.Errors)
	require.NotEmpty(t, v.Warnings)
	joined := joinStrings(v.Warnings)
	assert.Contains(t, joined, "HARNESS_INFERENCE_MODEL")
}

func TestValidateAgentManifest_ClaudeCodeModelKeyWarns(t *testing.T) {
	const manifest = `apiVersion: agents.digitalocean.com/v1alpha1
kind: Agent
metadata:
  name: notwork
spec:
  runtime:
    adapter: claude-code
  env:
    MODEL: claude-opus-4-5
    ANTHROPIC_API_KEY: sk-ant-XXX
`
	v := validateAgentManifest([]byte(manifest))
	assert.True(t, v.ok(), "errors=%v", v.Errors)
	joined := joinStrings(v.Warnings)
	assert.Contains(t, joined, "ANTHROPIC_MODEL")
	assert.Contains(t, joined, "MODEL")
}

func TestValidateAgentManifest_InvalidPermissionsDefault(t *testing.T) {
	const manifest = `name: validate-bug-test
agent: codex
secrets:
  OPENAI_API_KEY: "sk-placeholder"
permissions:
  default: banana
`
	v := validateAgentManifest([]byte(manifest))
	require.False(t, v.ok())
	assert.Contains(t, v.Errors[0], "permissions.default")
	assert.Contains(t, v.Errors[0], "banana")
	assert.Contains(t, v.Errors[0], "allow|ask|deny")
}

func TestValidateAgentManifest_PermissionsDefaultCaseSensitive(t *testing.T) {
	for _, def := range []string{"ALLOW", "Allow", "ASK", "Deny"} {
		t.Run(def, func(t *testing.T) {
			manifest := fmt.Sprintf("agent: opencode\npermissions:\n  default: %s\n", def)
			v := validateAgentManifest([]byte(manifest))
			require.False(t, v.ok())
			assert.Contains(t, v.Errors[0], "permissions.default")
			assert.Contains(t, v.Errors[0], "allow|ask|deny")
		})
	}
}

func TestValidateAgentManifest_ValidPermissionsDefault(t *testing.T) {
	for _, def := range []string{"allow", "ask", "deny"} {
		t.Run(def, func(t *testing.T) {
			manifest := fmt.Sprintf("agent: opencode\npermissions:\n  default: %s\n", def)
			v := validateAgentManifest([]byte(manifest))
			assert.True(t, v.ok(), "errors=%v", v.Errors)
		})
	}
}

func TestValidateAgentManifest_EnvelopeInvalidPermissionsDefault(t *testing.T) {
	const manifest = `apiVersion: agents.digitalocean.com/v1alpha1
kind: Agent
spec:
  runtime:
    adapter: opencode
  permissions:
    default: banana
`
	v := validateAgentManifest([]byte(manifest))
	require.False(t, v.ok())
	assert.Contains(t, v.Errors[0], "spec.permissions.default")
	assert.Contains(t, v.Errors[0], "banana")
}

func TestValidateAgentManifest_InvalidSkillName(t *testing.T) {
	const manifest = `name: validate-bug-test2
agent: codex
secrets:
  OPENAI_API_KEY: "sk-placeholder"
skills:
  - name: Release-Checklist
    description: Uppercase name is invalid.
    instructions: "Test"
`
	v := validateAgentManifest([]byte(manifest))
	require.False(t, v.ok())
	assert.Contains(t, v.Errors[0], "skills[0].name")
	assert.Contains(t, v.Errors[0], "Release-Checklist")
	assert.Contains(t, v.Errors[0], `^[a-z0-9]+(-[a-z0-9]+)*$`)
}

func TestValidateAgentManifest_InvalidSkillNameVariants(t *testing.T) {
	cases := []string{
		"foo--bar",
		"-leading",
		"trailing-",
		"Has Spaces",
		"under_score",
	}
	for _, name := range cases {
		t.Run(name, func(t *testing.T) {
			manifest := fmt.Sprintf("agent: opencode\nskills:\n  - name: %q\n    instructions: test\n", name)
			v := validateAgentManifest([]byte(manifest))
			require.False(t, v.ok(), "name %q should be rejected", name)
			assert.Contains(t, v.Errors[0], "skills[0].name")
		})
	}
}

func TestValidateAgentManifest_ValidSkillName(t *testing.T) {
	const manifest = `agent: opencode
skills:
  - name: release-checklist
    description: ok
    instructions: "Test"
`
	v := validateAgentManifest([]byte(manifest))
	assert.True(t, v.ok(), "errors=%v", v.Errors)
}

func TestValidateAgentManifest_EnvelopeInvalidSkillName(t *testing.T) {
	const manifest = `apiVersion: agents.digitalocean.com/v1alpha1
kind: Agent
spec:
  runtime:
    adapter: opencode
  skills:
    - name: Bad_Name
      instructions: test
`
	v := validateAgentManifest([]byte(manifest))
	require.False(t, v.ok())
	assert.Contains(t, v.Errors[0], "spec.skills[0].name")
	assert.Contains(t, v.Errors[0], "Bad_Name")
}

func TestRunAgentsValidate_OK(t *testing.T) {
	path := filepath.Join(t.TempDir(), "agent.yaml")
	require.NoError(t, os.WriteFile(path, []byte("agent: opencode\n"), 0o600))

	withTestClient(t, func(config *CmdConfig, _ *tcMocks) {
		config.Doit.Set(config.NS, doctl.ArgAgentSpec, path)
		var buf bytes.Buffer
		config.Out = &buf
		require.NoError(t, RunAgentsValidate(config))
		assert.Contains(t, buf.String(), "Manifest looks valid")
	})
}

func TestRunAgentsValidate_Invalid(t *testing.T) {
	path := filepath.Join(t.TempDir(), "agent.yaml")
	require.NoError(t, os.WriteFile(path, []byte("env:\n  FOO: bar\n"), 0o600))

	withTestClient(t, func(config *CmdConfig, _ *tcMocks) {
		config.Doit.Set(config.NS, doctl.ArgAgentSpec, path)
		var buf bytes.Buffer
		config.Out = &buf
		err := RunAgentsValidate(config)
		require.ErrorIs(t, err, ErrExitSilently)
		out := buf.String()
		assert.Contains(t, out, "Manifest is invalid")
		assert.Contains(t, out, "Error")
		assert.Contains(t, out, `"agent"`)
	})
}

func TestRunAgentsValidate_WarningsStyled(t *testing.T) {
	path := filepath.Join(t.TempDir(), "agent.yaml")
	require.NoError(t, os.WriteFile(path, []byte("agent: opencode\nenv:\n  OPENAI_API_KEY: sk-proj-literal\n"), 0o600))

	withTestClient(t, func(config *CmdConfig, _ *tcMocks) {
		config.Doit.Set(config.NS, doctl.ArgAgentSpec, path)
		var buf bytes.Buffer
		config.Out = &buf
		require.NoError(t, RunAgentsValidate(config))
		out := buf.String()
		assert.Contains(t, out, "Manifest looks valid")
		assert.Contains(t, out, "Warning")
		assert.Contains(t, out, "OPENAI_API_KEY")
		assert.Contains(t, out, "Review before create")
	})
}

func stringSliceToBytes(ss []string) [][]byte {
	out := make([][]byte, len(ss))
	for i, s := range ss {
		out[i] = []byte(s)
	}
	return out
}

func joinStrings(ss []string) string {
	return string(bytes.Join(stringSliceToBytes(ss), []byte("\n")))
}

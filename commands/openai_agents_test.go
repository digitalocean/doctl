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
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	yaml "gopkg.in/yaml.v2"
)

const sampleOpenAIManifest = `apiVersion: agents.digitalocean.com/v1alpha1
kind: Agent
metadata:
  name: codex-agentapi-session
spec:
  runtime:
    adapter: codex-agentapi
    config:
      agent:
        model: gpt-5.6-sol
        instructions: "Answer clearly."
      environment:
        type: self_hosted
        workspace_directory: /workspace
      input:
        - role: user
          content:
            - type: input_text
              text: "hello"
  sandbox:
    template: codex-agentapi
  env:
    CODEX_ENVIRONMENT_ID: ${ENV_ID}
    CODEX_API_KEY: ${OPENAI_API_KEY}
  secrets:
    - name: CODEX_API_KEY
      source: tenantSecret
`

// sampleOpenAIManifestLegacy uses the pre-agentspec location for the OpenAI
// create body. doctl still extracts it and strips it before DO create.
const sampleOpenAIManifestLegacy = `apiVersion: agents.digitalocean.com/v1alpha1
kind: Agent
metadata:
  name: codex-agentapi-session
spec:
  runtime:
    adapter: codex-agentapi
  sandbox:
    template: codex-agentapi
  env:
    CODEX_ENVIRONMENT_ID: ${ENV_ID}
    CODEX_API_KEY: ${OPENAI_API_KEY}
  openai:
    agent:
      model: gpt-5.6-sol
      instructions: "Answer clearly."
    environment:
      type: self_hosted
      workspace_directory: /workspace
`

// sampleFlatOpenAIManifest is the flat-format equivalent of
// sampleOpenAIManifest: top-level agent + config, no envelope.
const sampleFlatOpenAIManifest = `name: codex-agentapi-session
agent: codex-agentapi
config:
  agent:
    model: gpt-5.6-sol
    instructions: "Answer clearly."
  environment:
    type: self_hosted
    workspace_directory: /workspace
  input:
    - role: user
      content:
        - type: input_text
          text: "hello"
env:
  CODEX_ENVIRONMENT_ID: ${ENV_ID}
  CODEX_API_KEY: ${OPENAI_API_KEY}
`

func TestParseOpenAIAgentsSession(t *testing.T) {
	sess, err := parseOpenAIAgentsSession([]byte(`{
		"id": "sess_a91f3",
		"environment": {"environment_id": "env_abc123"}
	}`))
	require.NoError(t, err)
	assert.Equal(t, "sess_a91f3", sess.ID)
	assert.Equal(t, "env_abc123", sess.EnvironmentID)

	sess, err = parseOpenAIAgentsSession([]byte(`{
		"session_id": "sess_alt",
		"environment": {"id": "env_alt"}
	}`))
	require.NoError(t, err)
	assert.Equal(t, "sess_alt", sess.ID)
	assert.Equal(t, "env_alt", sess.EnvironmentID)
}

func TestExtractOpenAICreateBody(t *testing.T) {
	body, err := extractOpenAICreateBody([]byte(sampleOpenAIManifest))
	require.NoError(t, err)
	var payload map[string]any
	require.NoError(t, json.Unmarshal(body, &payload))
	agent, ok := payload["agent"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "gpt-5.6-sol", agent["model"])
	env, ok := payload["environment"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "self_hosted", env["type"])
}

func TestExtractOpenAICreateBody_LegacyOpenAI(t *testing.T) {
	body, err := extractOpenAICreateBody([]byte(sampleOpenAIManifestLegacy))
	require.NoError(t, err)
	var payload map[string]any
	require.NoError(t, json.Unmarshal(body, &payload))
	agent, ok := payload["agent"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "gpt-5.6-sol", agent["model"])
}

func TestExtractOpenAICreateBody_PrefersRuntimeConfig(t *testing.T) {
	manifest := `apiVersion: agents.digitalocean.com/v1alpha1
kind: Agent
spec:
  runtime:
    adapter: codex-agentapi
    config:
      agent:
        model: from-config
  openai:
    agent:
      model: from-openai
`
	body, err := extractOpenAICreateBody([]byte(manifest))
	require.NoError(t, err)
	var payload map[string]any
	require.NoError(t, json.Unmarshal(body, &payload))
	agent, ok := payload["agent"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "from-config", agent["model"])
}

func TestExtractOpenAICreateBody_Flat(t *testing.T) {
	body, err := extractOpenAICreateBody([]byte(sampleFlatOpenAIManifest))
	require.NoError(t, err)
	var payload map[string]any
	require.NoError(t, json.Unmarshal(body, &payload))
	agent, ok := payload["agent"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "gpt-5.6-sol", agent["model"])
	env, ok := payload["environment"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "self_hosted", env["type"])
}

func TestPrepareOpenAISandboxStart_NonOpenAI(t *testing.T) {
	id, overlay, err := prepareOpenAISandboxStart(context.Background(), []byte(sampleManifest))
	require.NoError(t, err)
	assert.Empty(t, id)
	assert.Nil(t, overlay)
}

func TestPrepareOpenAISandboxStart_FlatNonOpenAI(t *testing.T) {
	id, overlay, err := prepareOpenAISandboxStart(context.Background(), []byte("agent: opencode\n"))
	require.NoError(t, err)
	assert.Empty(t, id)
	assert.Nil(t, overlay)
}

func TestPrepareOpenAISandboxStart_FlatConfigRequiresCodexAgent(t *testing.T) {
	const manifest = `agent: opencode
config:
  agent:
    model: gpt-5.6-sol
`
	_, _, err := prepareOpenAISandboxStart(context.Background(), []byte(manifest))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "codex-agentapi")
	assert.Contains(t, err.Error(), `"opencode"`)
}

func TestPrepareOpenAISandboxStart_FlatCreatesSession(t *testing.T) {
	t.Setenv(openAIAPIKeyEnv, "sk-test")
	orig := createOpenAIAgentsSession
	t.Cleanup(func() { createOpenAIAgentsSession = orig })
	createOpenAIAgentsSession = func(ctx context.Context, apiKey string, body json.RawMessage) (*openAIAgentsSession, error) {
		assert.Equal(t, "sk-test", apiKey)
		var payload map[string]any
		require.NoError(t, json.Unmarshal(body, &payload))
		assert.Contains(t, payload, "agent")
		return &openAIAgentsSession{ID: "sess_flat1", EnvironmentID: "env_flat1"}, nil
	}

	id, overlay, err := prepareOpenAISandboxStart(context.Background(), []byte(sampleFlatOpenAIManifest))
	require.NoError(t, err)
	assert.Equal(t, "sess_flat1", id)
	assert.Equal(t, map[string]string{"ENV_ID": "env_flat1"}, overlay)
}

func TestPrepareOpenAISandboxStart_CreatesSession(t *testing.T) {
	t.Setenv(openAIAPIKeyEnv, "sk-test")
	orig := createOpenAIAgentsSession
	t.Cleanup(func() { createOpenAIAgentsSession = orig })
	createOpenAIAgentsSession = func(ctx context.Context, apiKey string, body json.RawMessage) (*openAIAgentsSession, error) {
		assert.Equal(t, "sk-test", apiKey)
		var payload map[string]any
		require.NoError(t, json.Unmarshal(body, &payload))
		assert.Contains(t, payload, "agent")
		return &openAIAgentsSession{ID: "sess_a91f3", EnvironmentID: "env_abc123"}, nil
	}

	id, overlay, err := prepareOpenAISandboxStart(context.Background(), []byte(sampleOpenAIManifest))
	require.NoError(t, err)
	assert.Equal(t, "sess_a91f3", id)
	assert.Equal(t, map[string]string{"ENV_ID": "env_abc123"}, overlay)
}

func TestHTTPOpenAIAgentsClient_SendInput(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/sessions/sess_x/events", r.URL.Path)
		var body map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		events, ok := body["events"].([]any)
		require.True(t, ok)
		require.Len(t, events, 1)
		evt := events[0].(map[string]any)
		assert.Equal(t, "session.input.message", evt["type"])
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(srv.Close)

	client := &httpOpenAIAgentsClient{httpClient: srv.Client(), baseURL: srv.URL}
	require.NoError(t, client.SendInput(context.Background(), "sk-test", "sess_x", "hello"))
}

func TestResolveOpenAIAgentsBaseURL(t *testing.T) {
	t.Setenv(agentAPIBaseURLEnv, "")
	t.Setenv(openAIAPIBaseURLEnv, "")
	assert.Equal(t, defaultOpenAIAgentsBase, resolveOpenAIAgentsBaseURL())

	t.Setenv(openAIAPIBaseURLEnv, "https://api.openai.com/v1")
	assert.Equal(t, "https://api.openai.com/v1/agents", resolveOpenAIAgentsBaseURL())

	t.Setenv(openAIAPIBaseURLEnv, "https://api.openai.com")
	assert.Equal(t, "https://api.openai.com/v1/agents", resolveOpenAIAgentsBaseURL())

	t.Setenv(agentAPIBaseURLEnv, "https://example.test/v1/agents")
	assert.Equal(t, "https://example.test/v1/agents", resolveOpenAIAgentsBaseURL())
}

func TestHTTPOpenAIAgentsClient_CreateSession(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/sessions", r.URL.Path)
		assert.Equal(t, "Bearer sk-test", r.Header.Get("Authorization"))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"sess_http","environment":{"environment_id":"env_http"}}`))
	}))
	t.Cleanup(srv.Close)

	client := &httpOpenAIAgentsClient{httpClient: srv.Client(), baseURL: srv.URL}
	sess, err := client.CreateSession(context.Background(), "sk-test", json.RawMessage(`{"agent":{"model":"gpt-5.6-sol"}}`))
	require.NoError(t, err)
	assert.Equal(t, "sess_http", sess.ID)
	assert.Equal(t, "env_http", sess.EnvironmentID)
}

func TestRunAgentsStart_OpenAI(t *testing.T) {
	dir := t.TempDir()
	specPath := filepath.Join(dir, "agent.yaml")
	require.NoError(t, os.WriteFile(specPath, []byte(sampleOpenAIManifest), 0o644))
	t.Setenv(openAIAPIKeyEnv, "sk-test-key")

	orig := createOpenAIAgentsSession
	t.Cleanup(func() { createOpenAIAgentsSession = orig })
	createOpenAIAgentsSession = func(ctx context.Context, apiKey string, body json.RawMessage) (*openAIAgentsSession, error) {
		return &openAIAgentsSession{ID: "sess_a91f3", EnvironmentID: "env_abc123"}, nil
	}

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.hostedAgents.EXPECT().
			CreateSessionFromManifest(gomock.Any(), gomock.Any()).
			DoAndReturn(func(manifest []byte, opt *godo.HostedAgentManifestCreateOptions) (*do.HostedAgentSession, error) {
				require.NotNil(t, opt)
				assert.Equal(t, "sess_a91f3", opt.OpenAISessionID)
				assert.Contains(t, string(manifest), "CODEX_ENVIRONMENT_ID: env_abc123")
				assert.Contains(t, string(manifest), "CODEX_API_KEY: sk-test-key")
				assert.NotContains(t, string(manifest), "${")
				assert.NotContains(t, string(manifest), "openai:")
				// Create body lives under runtime.config and is sent to DO.
				assert.Contains(t, string(manifest), "gpt-5.6-sol")
				assert.Contains(t, string(manifest), "config:")
				return &do.HostedAgentSession{
					HostedAgentSession: &godo.HostedAgentSession{
						SessionID:           "sess_do_1",
						Name:                "codex-agentapi-session",
						AgentKind:           godo.HostedAgentKindOpenAICodex,
						Status:              godo.HostedAgentSessionStatusReady,
						OpenAISessionID:     "sess_a91f3",
						OpenAIEnvironmentID: "env_abc123",
					},
				}, nil
			})
		tm.hostedAgents.EXPECT().
			GetSession("sess_do_1").
			Return(&do.HostedAgentSession{
				HostedAgentSession: &godo.HostedAgentSession{
					SessionID:           "sess_do_1",
					Name:                "codex-agentapi-session",
					AgentKind:           godo.HostedAgentKindOpenAICodex,
					Status:              godo.HostedAgentSessionStatusReady,
					OpenAISessionID:     "sess_a91f3",
					OpenAIEnvironmentID: "env_abc123",
				},
			}, nil)

		prev := sessionReadyPollInterval
		sessionReadyPollInterval = time.Millisecond
		defer func() { sessionReadyPollInterval = prev }()

		config.Doit.Set(config.NS, doctl.ArgAgentSpec, specPath)
		assert.NoError(t, RunAgentsStart(config))
	})
}

func TestRunAgentsStart_OpenAIFlat(t *testing.T) {
	dir := t.TempDir()
	specPath := filepath.Join(dir, "agent.yaml")
	require.NoError(t, os.WriteFile(specPath, []byte(sampleFlatOpenAIManifest), 0o644))
	t.Setenv(openAIAPIKeyEnv, "sk-test-key")

	orig := createOpenAIAgentsSession
	t.Cleanup(func() { createOpenAIAgentsSession = orig })
	createOpenAIAgentsSession = func(ctx context.Context, apiKey string, body json.RawMessage) (*openAIAgentsSession, error) {
		return &openAIAgentsSession{ID: "sess_flat1", EnvironmentID: "env_flat1"}, nil
	}

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.hostedAgents.EXPECT().
			CreateSessionFromManifest(gomock.Any(), gomock.Any()).
			DoAndReturn(func(manifest []byte, opt *godo.HostedAgentManifestCreateOptions) (*do.HostedAgentSession, error) {
				require.NotNil(t, opt)
				assert.Equal(t, "sess_flat1", opt.OpenAISessionID)
				assert.Contains(t, string(manifest), "CODEX_ENVIRONMENT_ID: env_flat1")
				assert.Contains(t, string(manifest), "CODEX_API_KEY: sk-test-key")
				assert.NotContains(t, string(manifest), "${")
				// The flat manifest carries no envelope and keeps its config
				// block for DO.
				assert.NotContains(t, string(manifest), "apiVersion")
				assert.Contains(t, string(manifest), "gpt-5.6-sol")
				assert.Contains(t, string(manifest), "config:")
				return &do.HostedAgentSession{
					HostedAgentSession: &godo.HostedAgentSession{
						SessionID:           "sess_do_2",
						Name:                "codex-agentapi-session",
						AgentKind:           godo.HostedAgentKindOpenAICodex,
						Status:              godo.HostedAgentSessionStatusReady,
						OpenAISessionID:     "sess_flat1",
						OpenAIEnvironmentID: "env_flat1",
					},
				}, nil
			})
		tm.hostedAgents.EXPECT().
			GetSession("sess_do_2").
			Return(&do.HostedAgentSession{
				HostedAgentSession: &godo.HostedAgentSession{
					SessionID:           "sess_do_2",
					Name:                "codex-agentapi-session",
					AgentKind:           godo.HostedAgentKindOpenAICodex,
					Status:              godo.HostedAgentSessionStatusReady,
					OpenAISessionID:     "sess_flat1",
					OpenAIEnvironmentID: "env_flat1",
				},
			}, nil)

		prev := sessionReadyPollInterval
		sessionReadyPollInterval = time.Millisecond
		defer func() { sessionReadyPollInterval = prev }()

		config.Doit.Set(config.NS, doctl.ArgAgentSpec, specPath)
		assert.NoError(t, RunAgentsStart(config))
	})
}

func TestStripSpecOpenAI(t *testing.T) {
	out, err := stripSpecOpenAI([]byte(sampleOpenAIManifestLegacy))
	require.NoError(t, err)
	assert.NotContains(t, string(out), "openai:")
	assert.Contains(t, string(out), "codex-agentapi")
	assert.Contains(t, string(out), "CODEX_ENVIRONMENT_ID")

	// Preferred manifests keep runtime.config; strip is a no-op for openai.
	out, err = stripSpecOpenAI([]byte(sampleOpenAIManifest))
	require.NoError(t, err)
	assert.Contains(t, string(out), "gpt-5.6-sol")
	assert.Contains(t, string(out), "config:")
}

func TestStripSpecOpenAI_PreservesMultiLineSkillsInstructions(t *testing.T) {
	const withSkills = `apiVersion: agents.digitalocean.com/v1alpha1
kind: Agent
spec:
  runtime:
    adapter: codex-agentapi
  skills:
    - name: example-skill
      description: A test skill with multi-line instructions
      instructions: |
        Step one: do the first thing.
        Step two: do the second thing.

        Step three: finish up.
  openai:
    agent:
      model: gpt-5.6-sol
`
	const wantInstructions = "Step one: do the first thing.\nStep two: do the second thing.\n\nStep three: finish up.\n"

	out, err := stripSpecOpenAI([]byte(withSkills))
	require.NoError(t, err)
	assert.NotContains(t, string(out), "openai:")

	var doc map[string]any
	require.NoError(t, yaml.Unmarshal(out, &doc))
	spec := doc["spec"].(map[any]any)
	skills, ok := spec["skills"].([]any)
	require.True(t, ok)
	require.Len(t, skills, 1)
	skill := skills[0].(map[any]any)
	assert.Equal(t, "example-skill", skill["name"])
	assert.Equal(t, wantInstructions, skill["instructions"])
}

func TestIsOpenAISandboxSession(t *testing.T) {
	assert.False(t, isOpenAISandboxSession(&do.HostedAgentSession{
		HostedAgentSession: &godo.HostedAgentSession{AgentKind: godo.HostedAgentKindOpenCode},
	}))
	assert.True(t, isOpenAISandboxSession(&do.HostedAgentSession{
		HostedAgentSession: &godo.HostedAgentSession{
			AgentKind:       godo.HostedAgentKindOpenAICodex,
			OpenAISessionID: "sess_x",
		},
	}))
}

func TestOpenAIAttachRenderer_PolishedOutput(t *testing.T) {
	var buf strings.Builder
	r := &openAIAttachRenderer{out: &buf}

	r.handle(map[string]any{"type": "session.turn.created"})
	r.handle(map[string]any{
		"type": "session.turn.item.added",
		"item": map[string]any{"type": "command_execution", "command": "ls -la"},
	})
	r.handle(map[string]any{
		"type": "session.turn.item.done",
		"item": map[string]any{
			"type":   "command_execution",
			"status": "completed",
			"output": "-rw-r--r-- 1 root root 10 program.py\n",
		},
	})
	r.handle(map[string]any{"type": "session.turn.output_text.delta", "delta": "Created `program.py`."})
	r.handle(map[string]any{
		"type":  "session.turn.completed",
		"usage": map[string]any{"input_tokens": float64(10), "output_tokens": float64(5)},
	})

	out := buf.String()
	assert.Contains(t, out, "ls -la")
	assert.Contains(t, out, "program.py")
	assert.Contains(t, out, "Created")
	assert.Contains(t, out, "run complete")
	assert.NotContains(t, out, `"event_id"`)
}

func TestOpenAIAttachRenderer_ReasoningStreamedViaDelta(t *testing.T) {
	var buf strings.Builder
	r := &openAIAttachRenderer{out: &buf}

	r.handle(map[string]any{"type": "session.turn.created"})
	r.handle(map[string]any{"type": "session.turn.reasoning_summary_text.delta", "delta": "Let's check "})
	r.handle(map[string]any{"type": "session.turn.reasoning_summary_text.delta", "delta": "the file first."})
	r.handle(map[string]any{"type": "session.turn.reasoning_summary_text.done", "text": "Let's check the file first."})
	r.handle(map[string]any{"type": "session.turn.output_text.delta", "delta": "Done."})
	r.handle(map[string]any{"type": "session.turn.completed"})

	out := buf.String()
	assert.Contains(t, out, "Let's check the file first.")
	assert.Contains(t, out, "Done.")
	// The .done fallback must not duplicate text already streamed via deltas.
	assert.Equal(t, 1, strings.Count(out, "Let's check the file first."))
}

func TestOpenAIAttachRenderer_ReasoningItemDoneFallback(t *testing.T) {
	var buf strings.Builder
	r := &openAIAttachRenderer{out: &buf}

	r.handle(map[string]any{"type": "session.turn.created"})
	r.handle(map[string]any{
		"type": "session.turn.item.added",
		"item": map[string]any{"type": "reasoning"},
	})
	r.handle(map[string]any{
		"type": "session.turn.item.done",
		"item": map[string]any{
			"type": "reasoning",
			"summary": []any{
				map[string]any{"type": "summary_text", "text": "Considering the best approach."},
			},
		},
	})
	r.handle(map[string]any{"type": "session.turn.output_text.delta", "delta": "Here's the plan."})
	r.handle(map[string]any{"type": "session.turn.completed"})

	out := buf.String()
	assert.Contains(t, out, "Considering the best approach.")
	assert.Contains(t, out, "Here's the plan.")
}

func TestOpenAIAttachRenderer_ReasoningDoesNotLeakIntoFinalAnswer(t *testing.T) {
	var buf strings.Builder
	r := &openAIAttachRenderer{out: &buf}

	r.handle(map[string]any{"type": "session.turn.created"})
	r.handle(map[string]any{"type": "session.turn.reasoning_summary_text.delta", "delta": "internal thoughts"})
	r.handle(map[string]any{"type": "session.turn.reasoning_summary_text.done", "text": "internal thoughts"})
	r.handle(map[string]any{"type": "session.turn.output_text.delta", "delta": "the actual answer"})
	r.handle(map[string]any{"type": "session.turn.completed"})

	out := buf.String()
	// Reasoning is streamed directly to r.out, never through r.acc, so it must
	// not appear inside the markdown-rendered final-answer block.
	completedIdx := strings.Index(out, "the actual answer")
	require.NotEqual(t, -1, completedIdx)
	assert.NotContains(t, out[completedIdx:], "internal thoughts")
}

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
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	yaml "gopkg.in/yaml.v2"
)

const (
	openAIAgentsAdapter       = "codex-agentapi"
	openAIAgentsAdapterLegacy = "openai-agent-codex"
	// openAIAgentsSessionsPath is relative to the Agents API base
	// (https://api.openai.com/v1/agents), matching the Python preview SDK.
	openAIAgentsSessionsPath = "/sessions"
	defaultOpenAIAgentsBase  = "https://api.openai.com/v1/agents"
	openAIAPIKeyEnv          = "OPENAI_API_KEY"
	// Prefer AGENT_API_BASE_URL (SDK). OPENAI_BASE_URL is accepted but normalized
	// so values like https://api.openai.com/v1 do not produce /v1/v1/... 404s.
	openAIAPIBaseURLEnv = "OPENAI_BASE_URL"
	agentAPIBaseURLEnv  = "AGENT_API_BASE_URL"
	envIDPlaceholder    = "ENV_ID"
)

// openAIAgentsClient talks to OpenAI's Agents API for the sandbox-provider POC.
// Create is required for `agents start`; Input/Stream power `agents attach`.
type openAIAgentsClient interface {
	CreateSession(ctx context.Context, apiKey string, body json.RawMessage) (*openAIAgentsSession, error)
	SendInput(ctx context.Context, apiKey, sessionID, text string) error
	Stream(ctx context.Context, apiKey, sessionID string, onEvent func(map[string]any)) error
}

type openAIAgentsSession struct {
	ID            string
	EnvironmentID string
}

type httpOpenAIAgentsClient struct {
	httpClient *http.Client
	baseURL    string
}

func newHTTPOpenAIAgentsClient() *httpOpenAIAgentsClient {
	return &httpOpenAIAgentsClient{
		httpClient: &http.Client{Timeout: 60 * time.Second},
		baseURL:    resolveOpenAIAgentsBaseURL(),
	}
}

// newHTTPOpenAIAgentsAttachClient uses a longer timeout: self-hosted turns often
// block on POST /events until the sandbox executor connects.
func newHTTPOpenAIAgentsAttachClient() *httpOpenAIAgentsClient {
	return &httpOpenAIAgentsClient{
		httpClient: &http.Client{Timeout: 3 * time.Minute},
		baseURL:    resolveOpenAIAgentsBaseURL(),
	}
}

// resolveOpenAIAgentsBaseURL returns the Agents API root used by the preview SDK
// (…/v1/agents). Paths like /sessions are joined onto this base.
func resolveOpenAIAgentsBaseURL() string {
	if v := strings.TrimRight(os.Getenv(agentAPIBaseURLEnv), "/"); v != "" {
		return v
	}
	if v := strings.TrimRight(os.Getenv(openAIAPIBaseURLEnv), "/"); v != "" {
		switch {
		case strings.HasSuffix(v, "/v1/agents"):
			return v
		case strings.HasSuffix(v, "/v1"):
			return v + "/agents"
		case v == "https://api.openai.com" || v == "http://api.openai.com":
			return v + "/v1/agents"
		default:
			// Assume the caller already pointed at the agents root.
			return v
		}
	}
	return defaultOpenAIAgentsBase
}

// createOpenAIAgentsSession is the production helper used by agents start.
var createOpenAIAgentsSession = func(ctx context.Context, apiKey string, body json.RawMessage) (*openAIAgentsSession, error) {
	return newHTTPOpenAIAgentsClient().CreateSession(ctx, apiKey, body)
}

// newOpenAIAgentsAttachClient is the production helper used by agents attach.
var newOpenAIAgentsAttachClient = func() openAIAgentsClient {
	return newHTTPOpenAIAgentsAttachClient()
}

func (c *httpOpenAIAgentsClient) CreateSession(ctx context.Context, apiKey string, body json.RawMessage) (*openAIAgentsSession, error) {
	if len(bytes.TrimSpace(body)) == 0 {
		return nil, fmt.Errorf("openai session body is empty")
	}
	url := c.baseURL + openAIAgentsSessionsPath
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("creating OpenAI Agents session: %w", err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading OpenAI Agents session response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		detail := truncateForErr(raw)
		if detail == "" {
			detail = "(empty body)"
		}
		err := fmt.Errorf("OpenAI Agents session create failed (HTTP %d) POST %s: %s", resp.StatusCode, url, detail)
		if resp.StatusCode == http.StatusNotFound {
			err = fmt.Errorf("%w\n\nHint: empty 404 usually means this API key/project is not enrolled in the OpenAI Agents API preview (path exists — invalid keys return 401). Confirm early-access on the same org/project as your key, or set AGENT_API_BASE_URL if using a non-default endpoint", err)
		}
		return nil, err
	}
	return parseOpenAIAgentsSession(raw)
}

func (c *httpOpenAIAgentsClient) SendInput(ctx context.Context, apiKey, sessionID, text string) error {
	// Preview SDK posts input as events on /sessions/{id}/events — there is no
	// /input route (that path 404s).
	payload, err := json.Marshal(map[string]any{
		"events": []map[string]any{
			{
				"type": "session.input.message",
				"input": []map[string]any{
					{
						"role": "user",
						"content": []map[string]any{
							{"type": "input_text", "text": text},
						},
					},
				},
			},
		},
	})
	if err != nil {
		return err
	}
	url := fmt.Sprintf("%s%s/%s/events", c.baseURL, openAIAgentsSessionsPath, sessionID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("sending OpenAI session input: %w", err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading OpenAI session input response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		detail := truncateForErr(raw)
		if detail == "" {
			detail = "(empty body)"
		}
		return fmt.Errorf("OpenAI session input failed (HTTP %d) POST %s: %s", resp.StatusCode, url, detail)
	}
	return nil
}

func (c *httpOpenAIAgentsClient) Stream(ctx context.Context, apiKey, sessionID string, onEvent func(map[string]any)) error {
	// SDK uses GET /sessions/{id}/events?stream=true
	url := fmt.Sprintf("%s%s/%s/events?stream=true", c.baseURL, openAIAgentsSessionsPath, sessionID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Accept", "text/event-stream")

	// Streaming can be long-lived; don't inherit the create client's hard timeout.
	client := &http.Client{Timeout: 0}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("opening OpenAI session event stream: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<10))
		return fmt.Errorf("OpenAI session event stream failed (HTTP %d): %s", resp.StatusCode, truncateForErr(raw))
	}

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	var dataBuf strings.Builder
	flush := func() {
		if dataBuf.Len() == 0 {
			return
		}
		data := dataBuf.String()
		dataBuf.Reset()
		if data == "" || data == "[DONE]" {
			return
		}
		var evt map[string]any
		if err := json.Unmarshal([]byte(data), &evt); err != nil {
			return
		}
		if onEvent != nil {
			onEvent(evt)
		}
	}
	for scanner.Scan() {
		line := scanner.Text()
		switch {
		case line == "":
			flush()
		case strings.HasPrefix(line, "data:"):
			chunk := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
			if dataBuf.Len() > 0 {
				dataBuf.WriteByte('\n')
			}
			dataBuf.WriteString(chunk)
		}
	}
	flush()
	if err := scanner.Err(); err != nil && ctx.Err() == nil {
		return err
	}
	return nil
}

func parseOpenAIAgentsSession(raw []byte) (*openAIAgentsSession, error) {
	var envelope struct {
		ID          string          `json:"id"`
		SessionID   string          `json:"session_id"`
		Environment json.RawMessage `json:"environment"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return nil, fmt.Errorf("decoding OpenAI Agents session response: %w", err)
	}
	sess := &openAIAgentsSession{
		ID: firstNonEmpty(envelope.SessionID, envelope.ID),
	}
	if sess.ID == "" {
		return nil, fmt.Errorf("OpenAI Agents session response missing id")
	}
	if len(envelope.Environment) > 0 && string(envelope.Environment) != "null" {
		var envObj struct {
			ID            string `json:"id"`
			EnvironmentID string `json:"environment_id"`
		}
		if err := json.Unmarshal(envelope.Environment, &envObj); err == nil {
			sess.EnvironmentID = firstNonEmpty(envObj.EnvironmentID, envObj.ID)
		}
		if sess.EnvironmentID == "" {
			// Some previews return the environment id as a bare string.
			var envStr string
			if err := json.Unmarshal(envelope.Environment, &envStr); err == nil {
				sess.EnvironmentID = envStr
			}
		}
	}
	if sess.EnvironmentID == "" {
		return nil, fmt.Errorf("OpenAI Agents session response missing environment id")
	}
	return sess, nil
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func truncateForErr(b []byte) string {
	const max = 512
	s := strings.TrimSpace(string(b))
	if len(s) <= max {
		return s
	}
	return s[:max] + "…"
}

// agentManifestDoc is the subset of agents.yaml we need for OpenAI orchestration.
type agentManifestDoc struct {
	Spec struct {
		Runtime struct {
			Adapter string `yaml:"adapter"`
			// Config holds the OpenAI create-session body (agent/environment/input)
			// under the DO agentspec. Preferred over legacy spec.openai.
			Config any `yaml:"config"`
		} `yaml:"runtime"`
		// OpenAI is the legacy client-only location for the create body. Still
		// accepted for extraction, but stripped before the DO create call.
		OpenAI any `yaml:"openai"`
	} `yaml:"spec"`
}

func parseAgentManifest(manifest []byte) (*agentManifestDoc, error) {
	var doc agentManifestDoc
	if err := yaml.Unmarshal(manifest, &doc); err != nil {
		return nil, fmt.Errorf("parsing agent manifest: %w", err)
	}
	return &doc, nil
}

func isOpenAISandboxAdapter(adapter string) bool {
	switch strings.TrimSpace(adapter) {
	case openAIAgentsAdapter, openAIAgentsAdapterLegacy:
		return true
	default:
		return false
	}
}

func hasOpenAICreateBody(doc *agentManifestDoc) bool {
	if doc == nil {
		return false
	}
	return doc.Spec.Runtime.Config != nil || doc.Spec.OpenAI != nil
}

func extractOpenAICreateBody(manifest []byte) (json.RawMessage, error) {
	doc, err := parseAgentManifest(manifest)
	if err != nil {
		return nil, err
	}
	// Prefer spec.runtime.config (current DO agentspec). Fall back to legacy
	// spec.openai for older manifests.
	source := doc.Spec.Runtime.Config
	sourceName := "spec.runtime.config"
	if source == nil {
		source = doc.Spec.OpenAI
		sourceName = "spec.openai"
	}
	if source == nil {
		return nil, fmt.Errorf("manifest adapter %q requires a non-empty %s (or legacy spec.openai) block", openAIAgentsAdapter, "spec.runtime.config")
	}
	// yaml.v2 decodes nested maps as map[interface{}]interface{}; normalize
	// before JSON encoding so Marshal succeeds.
	normalized, err := normalizeYAMLValue(source)
	if err != nil {
		return nil, fmt.Errorf("normalizing %s: %w", sourceName, err)
	}
	body, err := json.Marshal(normalized)
	if err != nil {
		return nil, fmt.Errorf("encoding %s for OpenAI create: %w", sourceName, err)
	}
	return body, nil
}

// stripSpecOpenAI removes legacy spec.openai from a manifest so the DO
// agentspec validator does not reject it. Preferred create-body location is
// spec.runtime.config, which is left in place for DO.
func stripSpecOpenAI(manifest []byte) ([]byte, error) {
	var doc map[any]any
	if err := yaml.Unmarshal(manifest, &doc); err != nil {
		return nil, fmt.Errorf("parsing agent manifest to strip spec.openai: %w", err)
	}
	spec, _ := doc["spec"].(map[any]any)
	if spec == nil {
		return manifest, nil
	}
	if _, ok := spec["openai"]; !ok {
		return manifest, nil
	}
	delete(spec, "openai")
	out, err := yaml.Marshal(doc)
	if err != nil {
		return nil, fmt.Errorf("re-encoding agent manifest without spec.openai: %w", err)
	}
	return out, nil
}

// normalizeYAMLValue converts yaml.v2 decode shapes into JSON-safe values.
func normalizeYAMLValue(v any) (any, error) {
	switch t := v.(type) {
	case map[any]any:
		out := make(map[string]any, len(t))
		for k, val := range t {
			ks, ok := k.(string)
			if !ok {
				return nil, fmt.Errorf("non-string map key %T", k)
			}
			nv, err := normalizeYAMLValue(val)
			if err != nil {
				return nil, err
			}
			out[ks] = nv
		}
		return out, nil
	case map[string]any:
		out := make(map[string]any, len(t))
		for k, val := range t {
			nv, err := normalizeYAMLValue(val)
			if err != nil {
				return nil, err
			}
			out[k] = nv
		}
		return out, nil
	case []any:
		out := make([]any, len(t))
		for i, val := range t {
			nv, err := normalizeYAMLValue(val)
			if err != nil {
				return nil, err
			}
			out[i] = nv
		}
		return out, nil
	default:
		return v, nil
	}
}

// prepareOpenAISandboxStart creates the OpenAI Agents session (when needed) and
// returns the openai session id plus an env overlay for ${ENV_ID} expansion.
// For non-OpenAI manifests it returns empty opts and a nil overlay.
func prepareOpenAISandboxStart(ctx context.Context, manifest []byte) (openaiSessionID string, envOverlay map[string]string, err error) {
	doc, err := parseAgentManifest(manifest)
	if err != nil {
		return "", nil, err
	}
	if !isOpenAISandboxAdapter(doc.Spec.Runtime.Adapter) && !hasOpenAICreateBody(doc) {
		return "", nil, nil
	}
	if !isOpenAISandboxAdapter(doc.Spec.Runtime.Adapter) {
		return "", nil, fmt.Errorf("spec.runtime.config (or legacy spec.openai) is only valid with adapter %q (got %q)", openAIAgentsAdapter, doc.Spec.Runtime.Adapter)
	}

	apiKey := strings.TrimSpace(os.Getenv(openAIAPIKeyEnv))
	if apiKey == "" {
		return "", nil, fmt.Errorf("%s is required to start a %s session", openAIAPIKeyEnv, openAIAgentsAdapter)
	}
	body, err := extractOpenAICreateBody(manifest)
	if err != nil {
		return "", nil, err
	}
	sess, err := createOpenAIAgentsSession(ctx, apiKey, body)
	if err != nil {
		return "", nil, err
	}
	return sess.ID, map[string]string{envIDPlaceholder: sess.EnvironmentID}, nil
}

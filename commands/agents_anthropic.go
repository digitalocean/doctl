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
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	defaultAnthropicAPIBase = "https://api.anthropic.com"
	// anthropicBaseURLEnv mirrors the Anthropic SDKs' own override, so a key
	// pointed at a proxy/gateway validates against that proxy too.
	anthropicBaseURLEnv    = "ANTHROPIC_BASE_URL"
	anthropicAPIVersion    = "2023-06-01"
	anthropicModelsPath    = "/v1/models"
	anthropicKeyCheckLabel = "Validating Anthropic API key…"
)

// validateAnthropicAPIKey is a var (not a plain func) so tests can stub it
// out — mirrors createOpenAIAgentsSession. It calls GET /v1/models, a free,
// side-effect-free endpoint that still requires a valid key, so a bogus
// ANTHROPIC_API_KEY (empty check in ensureEnvVar only catches unset/blank)
// is caught locally with a clear message instead of surfacing later as an
// opaque failure deep inside the hosted claude-code sandbox.
var validateAnthropicAPIKey = func(ctx context.Context, apiKey string) error {
	return newHTTPAnthropicClient().checkAPIKey(ctx, apiKey)
}

type httpAnthropicClient struct {
	httpClient *http.Client
	baseURL    string
}

func newHTTPAnthropicClient() *httpAnthropicClient {
	return &httpAnthropicClient{
		httpClient: &http.Client{Timeout: 15 * time.Second},
		baseURL:    resolveAnthropicBaseURL(),
	}
}

func resolveAnthropicBaseURL() string {
	if v := strings.TrimRight(strings.TrimSpace(os.Getenv(anthropicBaseURLEnv)), "/"); v != "" {
		return v
	}
	return defaultAnthropicAPIBase
}

// checkAPIKey confirms apiKey is accepted by Anthropic before doctl ever
// calls CreateSessionFromManifest. Any non-2xx response (401/403 for a bad
// key, or anything else) fails the same way the OpenAI codex path already
// fails fast on a bad OPENAI_API_KEY: locally, before a hosted session is
// created that was never going to work.
func (c *httpAnthropicClient) checkAPIKey(ctx context.Context, apiKey string) error {
	url := c.baseURL + anthropicModelsPath
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", anthropicAPIVersion)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("validating %s: %w", anthropicAPIKeyEnv, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}
	raw, _ := io.ReadAll(resp.Body)
	detail := truncateForErr(raw)
	if detail == "" {
		detail = "(empty body)"
	}
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return fmt.Errorf("%s was rejected by Anthropic (HTTP %d): %s", anthropicAPIKeyEnv, resp.StatusCode, detail)
	}
	return fmt.Errorf("validating %s failed (HTTP %d) GET %s: %s", anthropicAPIKeyEnv, resp.StatusCode, url, detail)
}

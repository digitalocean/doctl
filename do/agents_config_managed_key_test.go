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

package do_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHostedAgentsService_CreateAgentConfigWithManagedKey(t *testing.T) {
	var gotMethod, gotPath, gotContentType string
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		gotContentType = r.Header.Get("Content-Type")
		raw, _ := io.ReadAll(r.Body)
		require.NoError(t, json.Unmarshal(raw, &gotBody))
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Location", "https://api-ric1.digitalocean.com/v2/harness-runtime/configs/cfg_managed")
		_, _ = w.Write([]byte(`{"config":{"id":"cfg_managed","name":"oh"}}`))
	}))
	t.Cleanup(srv.Close)
	client, err := godo.New(nil, godo.SetBaseURL(srv.URL+"/"))
	require.NoError(t, err)

	cfg, err := do.NewHostedAgentsService(client).CreateAgentConfig(&godo.HostedAgentConfigCreateRequest{
		Name:         "oh",
		ManifestYAML: "harness: openhands\nenv:\n  HARNESS_INFERENCE_MODEL: openai-gpt-5.5\n",
	}, true)
	require.NoError(t, err)

	assert.Equal(t, http.MethodPost, gotMethod)
	assert.Equal(t, "/v2/harness-runtime/configs", gotPath)
	assert.Equal(t, "application/json", gotContentType)
	assert.Equal(t, "oh", gotBody["name"])
	assert.Contains(t, gotBody["manifest_yaml"], "HARNESS_INFERENCE_MODEL")
	assert.Equal(t, true, gotBody["manage_model_access_key"])
	require.NotNil(t, cfg)
	assert.Equal(t, "cfg_managed", cfg.ID)
	assert.Equal(t, "https://api-ric1.digitalocean.com/v2/harness-runtime/configs/cfg_managed", cfg.Location)
}

func TestHostedAgentsService_CreateAgentConfigWithoutManagedKeyOmitsField(t *testing.T) {
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		require.NoError(t, json.Unmarshal(raw, &gotBody))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"config":{"id":"cfg_1","name":"x"}}`))
	}))
	t.Cleanup(srv.Close)
	client, err := godo.New(nil, godo.SetBaseURL(srv.URL+"/"))
	require.NoError(t, err)

	_, err = do.NewHostedAgentsService(client).CreateAgentConfig(&godo.HostedAgentConfigCreateRequest{
		Name:         "x",
		ManifestYAML: "harness: opencode\n",
	}, false)
	require.NoError(t, err)
	_, has := gotBody["manage_model_access_key"]
	assert.False(t, has, "false must omit the field rather than send false")
}

func TestHostedAgentsService_CreateAgentConfigRejectsNil(t *testing.T) {
	client, err := godo.New(nil)
	require.NoError(t, err)
	_, err = do.NewHostedAgentsService(client).CreateAgentConfig(nil, false)
	assert.ErrorContains(t, err, "create request is required")
}

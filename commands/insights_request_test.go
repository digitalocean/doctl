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
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadInsightsRequest(t *testing.T) {
	path := filepath.Join(t.TempDir(), "channel.yaml")
	require.NoError(t, os.WriteFile(path, []byte(`
name: Operations
email:
  to: ops@example.com
`), 0o600))

	request, err := readInsightsRequest[godo.NotificationChannelRequest](strings.NewReader(""), path)
	require.NoError(t, err)
	assert.Equal(t, &godo.NotificationChannelRequest{
		Name:  "Operations",
		Email: &godo.EmailNotificationConfig{To: "ops@example.com"},
	}, request)
}

func TestReadInsightsRequestFromStdin(t *testing.T) {
	request, err := readInsightsRequest[godo.NotificationChannelRequest](strings.NewReader(
		`{"name":"Operations","email":{"to":"ops@example.com"}}`,
	), "-")
	require.NoError(t, err)
	assert.Equal(t, "Operations", request.Name)
	assert.Equal(t, "ops@example.com", request.Email.To)
}

func TestReadInsightsRequestRejectsUnknownFields(t *testing.T) {
	_, err := readInsightsRequest[godo.NotificationChannelRequest](strings.NewReader(
		`{"name":"Operations","unknown":true}`,
	), "-")
	require.Error(t, err)
	assert.Contains(t, err.Error(), `unknown field "unknown"`)
}

func TestReadInsightsDashboardSpecPreservesJSONBytes(t *testing.T) {
	// Key order and whitespace are kept so the service stores exactly what the
	// user wrote.
	const spec = "{\"kind\": \"Dashboard\",\n  \"spec\": {\"display\": {\"name\": \"Droplet CPU\"}}}"
	path := filepath.Join(t.TempDir(), "dashboard.json")
	require.NoError(t, os.WriteFile(path, []byte(spec+"\n"), 0o600))

	got, err := readInsightsDashboardSpec(strings.NewReader(""), path)
	require.NoError(t, err)
	assert.Equal(t, spec, string(got))
}

func TestReadInsightsDashboardSpecFromStdin(t *testing.T) {
	got, err := readInsightsDashboardSpec(strings.NewReader(`{"kind":"Dashboard"}`), "-")
	require.NoError(t, err)
	assert.Equal(t, `{"kind":"Dashboard"}`, string(got))
}

func TestReadInsightsDashboardSpecConvertsYAML(t *testing.T) {
	got, err := readInsightsDashboardSpec(strings.NewReader(`
kind: Dashboard
spec:
  display:
    name: Droplet CPU
`), "-")
	require.NoError(t, err)
	assert.JSONEq(t, `{"kind":"Dashboard","spec":{"display":{"name":"Droplet CPU"}}}`, string(got))
}

func TestReadInsightsDashboardSpecErrors(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr string
	}{
		{name: "empty", input: "", wantErr: "spec file must contain a single JSON object"},
		{name: "whitespace", input: " \n\t", wantErr: "spec file must contain a single JSON object"},
		{name: "array", input: `[{"kind":"Dashboard"}]`, wantErr: "spec file must contain a single JSON object"},
		{name: "double encoded", input: `"{\"kind\":\"Dashboard\"}"`, wantErr: "spec file must contain a single JSON object"},
		{name: "two objects", input: `{"kind":"Dashboard"} {"kind":"Dashboard"}`, wantErr: "spec file must contain a single JSON object"},
		{name: "object then garbage", input: `{"kind":"Dashboard"} trailing`, wantErr: "spec file must contain a single JSON object"},
		{name: "invalid", input: `{"kind":`, wantErr: "parsing spec file"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := readInsightsDashboardSpec(strings.NewReader(tt.input), "-")
			assert.ErrorContains(t, err, tt.wantErr)
		})
	}
}

func TestReadInsightsDashboardSpecMissingFile(t *testing.T) {
	_, err := readInsightsDashboardSpec(strings.NewReader(""), filepath.Join(t.TempDir(), "missing.json"))
	assert.ErrorContains(t, err, "reading spec file")
}

func TestInsightsPageOptions(t *testing.T) {
	tests := []struct {
		name        string
		page        any
		perPage     any
		wantPage    int
		wantPerPage int
		wantErr     string
	}{
		{name: "unset leaves server defaults"},
		{name: "explicit values", page: 2, perPage: 50, wantPage: 2, wantPerPage: 50},
		{name: "max per page", perPage: 200, wantPerPage: 200},
		{name: "page zero", page: 0, wantErr: "--page must be at least 1"},
		{name: "page negative", page: -1, wantErr: "--page must be at least 1"},
		{name: "per page zero", perPage: 0, wantErr: "--per-page must be between 1 and 200"},
		{name: "per page negative", perPage: -5, wantErr: "--per-page must be between 1 and 200"},
		{name: "per page too large", perPage: 201, wantErr: "--per-page must be between 1 and 200"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withTestClient(t, func(config *CmdConfig, _ *tcMocks) {
				if tt.page != nil {
					config.Doit.Set(config.NS, doctl.ArgInsightsPage, tt.page)
				}
				if tt.perPage != nil {
					config.Doit.Set(config.NS, doctl.ArgInsightsPerPage, tt.perPage)
				}
				page, perPage, err := insightsPageOptions(config, 200)
				if tt.wantErr != "" {
					assert.ErrorContains(t, err, tt.wantErr)
					return
				}
				require.NoError(t, err)
				assert.Equal(t, tt.wantPage, page)
				assert.Equal(t, tt.wantPerPage, perPage)
			})
		})
	}
}

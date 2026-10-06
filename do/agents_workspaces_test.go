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
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// workspaceTestService serves handler on a throwaway server and returns a
// service bound to it.
func workspaceTestService(t *testing.T, handler http.HandlerFunc) do.HostedAgentsService {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	client, err := godo.New(nil, godo.SetBaseURL(srv.URL+"/"))
	require.NoError(t, err)
	return do.NewHostedAgentsService(client)
}

func TestHostedAgentsService_CreateWorkspace(t *testing.T) {
	var gotMethod, gotPath, gotKey string
	var gotBody map[string]any
	svc := workspaceTestService(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath, gotKey = r.Method, r.URL.Path, r.Header.Get("Idempotency-Key")
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"workspace":{"workspace_id":"ws_1","name":"notes","state":"AVAILABLE","size_gibibytes":10}}`))
	})

	ws, err := svc.CreateWorkspace(&godo.HostedAgentWorkspaceCreateRequest{SizeGibibytes: 10, Name: "notes", IdempotencyKey: "key-1"})
	require.NoError(t, err)
	assert.Equal(t, http.MethodPost, gotMethod)
	assert.Equal(t, "/v2/agents/workspaces", gotPath)
	assert.Equal(t, "key-1", gotKey, "the key travels as a header")
	assert.Equal(t, map[string]any{"size_gibibytes": float64(10), "name": "notes"}, gotBody, "and never in the body")
	assert.Equal(t, "ws_1", ws.WorkspaceID)
	assert.Equal(t, godo.HostedAgentWorkspaceStateAvailable, ws.State)
}

func TestHostedAgentsService_ListGetDeleteWorkspace(t *testing.T) {
	var calls []string
	svc := workspaceTestService(t, func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, r.Method+" "+r.URL.RequestURI())
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v2/agents/workspaces":
			_, _ = w.Write([]byte(`{"workspaces":[{"workspace_id":"ws_1"},{"workspace_id":"ws_2"}],"next_page_token":"tok2"}`))
		case r.Method == http.MethodGet:
			_, _ = w.Write([]byte(`{"workspace":{"workspace_id":"ws_1","attached_session_id":"sess_1"}}`))
		default:
			w.WriteHeader(http.StatusNoContent)
		}
	})

	list, next, err := svc.ListWorkspaces(&godo.HostedAgentWorkspaceListOptions{PageSize: 2, PageToken: "tok1"})
	require.NoError(t, err)
	require.Len(t, list, 2)
	assert.Equal(t, "tok2", next)

	ws, err := svc.GetWorkspace("ws_1")
	require.NoError(t, err)
	assert.Equal(t, "sess_1", ws.AttachedSessionID)

	require.NoError(t, svc.DeleteWorkspace("ws_1"))

	assert.Equal(t, []string{
		"GET /v2/agents/workspaces?page_size=2&page_token=tok1",
		"GET /v2/agents/workspaces/ws_1",
		"DELETE /v2/agents/workspaces/ws_1",
	}, calls)
}

func TestHostedAgentsService_WorkspaceErrorsKeepTheirStatus(t *testing.T) {
	for _, status := range []int{http.StatusConflict, http.StatusNotImplemented} {
		svc := workspaceTestService(t, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(status)
			_, _ = w.Write([]byte(`{"error":{"message":"server says no"}}`))
		})

		err := svc.DeleteWorkspace("ws_1")
		var er *godo.ErrorResponse
		require.ErrorAs(t, err, &er)
		assert.Equal(t, status, er.Response.StatusCode)
	}
}

// The session-create paths name the workspace per session: a query parameter
// for a manifest, a body field for a config.
func TestHostedAgentsService_SessionCreateCarriesWorkspace(t *testing.T) {
	var manifestQuery string
	var configBody map[string]any
	svc := workspaceTestService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// Both create paths POST to the same route; the manifest one is YAML.
		if r.Header.Get("Content-Type") == "application/x-yaml" {
			manifestQuery = r.URL.Query().Get("workspace_id")
		} else {
			_ = json.NewDecoder(r.Body).Decode(&configBody)
		}
		_, _ = w.Write([]byte(`{"session":{"session_id":"sess_1","workspace_id":"ws_1"}}`))
	})

	sess, err := svc.CreateSessionFromManifest([]byte("agent: opencode\n"), &godo.HostedAgentManifestCreateOptions{WorkspaceID: "ws_1"})
	require.NoError(t, err)
	assert.Equal(t, "ws_1", manifestQuery)
	assert.Equal(t, "ws_1", sess.WorkspaceID)

	_, err = svc.CreateSessionFromConfig(&godo.HostedAgentSessionFromConfigRequest{Name: "demo", ConfigID: "cfg_1", WorkspaceID: "ws_1"})
	require.NoError(t, err)
	assert.Equal(t, "ws_1", configBody["workspace_id"])
}

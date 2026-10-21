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
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The managed-key create must reach harness-api exactly as godo's own
// manifest create does (same path, YAML body, JSON back), plus the one query
// parameter godo has no field for. Both halves are asserted on the wire,
// because the server keys the agents.yaml form on the media type and refuses
// the parameter on any other.
func TestHostedAgentsService_CreateSessionFromManifestWithManagedKey(t *testing.T) {
	manifest := []byte("name: oh\nagent: openhands\n")
	cases := []struct {
		name      string
		opt       *godo.HostedAgentManifestCreateOptions
		wantQuery map[string]string
	}{
		{
			name:      "nil options",
			wantQuery: map[string]string{"manage_model_access_key": "true"},
		},
		{
			name: "godo's own options still travel",
			opt:  &godo.HostedAgentManifestCreateOptions{ResumeOnTopoff: true, OpenAISessionID: "oai_1"},
			wantQuery: map[string]string{
				"manage_model_access_key": "true",
				"resume_on_topoff":        "true",
				"openai_session_id":       "oai_1",
			},
		},
		{
			name:      "false options are omitted, as godo omits them",
			opt:       &godo.HostedAgentManifestCreateOptions{},
			wantQuery: map[string]string{"manage_model_access_key": "true"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var gotMethod, gotPath, gotContentType, gotAccept string
			var gotBody []byte
			gotQuery := map[string]string{}
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotMethod, gotPath = r.Method, r.URL.Path
				gotContentType, gotAccept = r.Header.Get("Content-Type"), r.Header.Get("Accept")
				for k, v := range r.URL.Query() {
					require.Len(t, v, 1, "each query parameter once: %s", k)
					gotQuery[k] = v[0]
				}
				gotBody, _ = io.ReadAll(r.Body)
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"session":{"session_id":"sess_oh","name":"oh","status":"SESSION_STATUS_PROVISIONING"}}`))
			}))
			t.Cleanup(srv.Close)
			client, err := godo.New(nil, godo.SetBaseURL(srv.URL+"/"))
			require.NoError(t, err)

			sess, err := do.NewHostedAgentsService(client).CreateSessionFromManifestWithManagedKey(manifest, tc.opt)
			require.NoError(t, err)

			assert.Equal(t, http.MethodPost, gotMethod)
			assert.Equal(t, "/v2/agents/sessions", gotPath)
			assert.Equal(t, "application/x-yaml", gotContentType)
			assert.Equal(t, "application/json", gotAccept)
			assert.Equal(t, tc.wantQuery, gotQuery)
			assert.Equal(t, manifest, gotBody, "the manifest goes out verbatim")
			require.NotNil(t, sess)
			assert.Equal(t, "sess_oh", sess.SessionID)
		})
	}
}

func TestHostedAgentsService_CreateSessionFromManifestWithManagedKeyRejectsAnEmptyManifest(t *testing.T) {
	client, err := godo.New(nil)
	require.NoError(t, err)
	_, err = do.NewHostedAgentsService(client).CreateSessionFromManifestWithManagedKey([]byte("  \n"), nil)
	assert.ErrorContains(t, err, "manifest is required")
}

// A server error comes back as godo's error, not as a nil session.
func TestHostedAgentsService_CreateSessionFromManifestWithManagedKeyServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"id":"bad_request","message":"manage_model_access_key must be true or false"}`))
	}))
	t.Cleanup(srv.Close)
	client, err := godo.New(nil, godo.SetBaseURL(srv.URL+"/"))
	require.NoError(t, err)
	_, err = do.NewHostedAgentsService(client).CreateSessionFromManifestWithManagedKey([]byte("agent: openhands\n"), nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "400")
}

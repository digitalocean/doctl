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

func TestHostedAgentsService_CancelTurn(t *testing.T) {
	cases := []struct {
		name     string
		runID    string
		status   int
		body     string
		wantBody map[string]string
		want     do.HostedAgentCancelTurnResult
		wantErr  string
	}{
		{
			name: "acked", runID: "run-1", status: http.StatusOK, body: `{"outcome":"acked","run_id":"run-1"}`,
			wantBody: map[string]string{"run_id": "run-1"},
			want:     do.HostedAgentCancelTurnResult{Outcome: do.HostedAgentCancelTurnAcked, RunID: "run-1"},
		},
		{
			name: "no turn", runID: "run-1", status: http.StatusOK, body: `{"outcome":"no_turn","run_id":"run-1"}`,
			wantBody: map[string]string{"run_id": "run-1"},
			want:     do.HostedAgentCancelTurnResult{Outcome: do.HostedAgentCancelTurnNoTurn, RunID: "run-1"},
		},
		{
			name: "unsupported", runID: "run-1", status: http.StatusOK, body: `{"outcome":"unsupported","run_id":"run-1"}`,
			wantBody: map[string]string{"run_id": "run-1"},
			want:     do.HostedAgentCancelTurnResult{Outcome: do.HostedAgentCancelTurnUnsupported, RunID: "run-1"},
		},
		{
			name: "older server echoes no run id", runID: "run-1", status: http.StatusOK, body: `{"outcome":"acked"}`,
			wantBody: map[string]string{"run_id": "run-1"},
			want:     do.HostedAgentCancelTurnResult{Outcome: do.HostedAgentCancelTurnAcked, RunID: "run-1"},
		},
		{
			name: "no run id cancels the turn in flight", status: http.StatusOK, body: `{"outcome":"acked","run_id":"run-live"}`,
			wantBody: map[string]string{},
			want:     do.HostedAgentCancelTurnResult{Outcome: do.HostedAgentCancelTurnAcked, RunID: "run-live"},
		},
		{
			name: "no run id and nothing running", status: http.StatusOK, body: `{"outcome":"no_turn"}`,
			wantBody: map[string]string{},
			want:     do.HostedAgentCancelTurnResult{Outcome: do.HostedAgentCancelTurnNoTurn},
		},
		{
			name: "paused session", runID: "run-1", status: http.StatusConflict, body: `{"error":{"code":409,"message":"session is paused"}}`,
			wantBody: map[string]string{"run_id": "run-1"},
			wantErr:  "409",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var gotMethod, gotPath string
			var gotBody map[string]string
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotMethod, gotPath = r.Method, r.URL.Path
				_ = json.NewDecoder(r.Body).Decode(&gotBody)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			}))
			t.Cleanup(srv.Close)
			client, err := godo.New(nil, godo.SetBaseURL(srv.URL+"/"))
			require.NoError(t, err)

			got, err := do.NewHostedAgentsService(client).CancelTurn("sess_x", tc.runID)

			assert.Equal(t, http.MethodPost, gotMethod)
			assert.Equal(t, "/v2/agents/sessions/sess_x/cancel", gotPath)
			assert.Equal(t, tc.wantBody, gotBody, "an unnamed cancel sends no run_id key at all")
			if tc.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestHostedAgentsService_RestartAgent(t *testing.T) {
	cases := []struct {
		name    string
		resume  bool
		status  int
		body    string
		wantErr string
	}{
		{name: "resume", resume: true, status: http.StatusAccepted},
		{name: "fresh", resume: false, status: http.StatusAccepted},
		{name: "not enabled", resume: true, status: http.StatusForbidden, body: `{"id":"forbidden","message":"agent restart is not enabled for this team"}`, wantErr: "403"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var gotMethod, gotPath string
			var gotBody map[string]any
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotMethod, gotPath = r.Method, r.URL.Path
				_ = json.NewDecoder(r.Body).Decode(&gotBody)
				if tc.body != "" {
					w.Header().Set("Content-Type", "application/json")
				}
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			}))
			t.Cleanup(srv.Close)
			client, err := godo.New(nil, godo.SetBaseURL(srv.URL+"/"))
			require.NoError(t, err)

			err = do.NewHostedAgentsService(client).RestartAgent("sess_x", tc.resume)

			assert.Equal(t, http.MethodPost, gotMethod)
			assert.Equal(t, "/v2/agents/sessions/sess_x/restart", gotPath)
			assert.Equal(t, map[string]any{"resume": tc.resume}, gotBody, "resume is always sent, false included")
			if tc.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.wantErr)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestHostedAgentsService_CancelTurnRequiresSessionID(t *testing.T) {
	client, err := godo.New(nil)
	require.NoError(t, err)

	_, err = do.NewHostedAgentsService(client).CancelTurn("", "run-1")
	assert.ErrorContains(t, err, "session id is required")
}

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

// Run posts to the run-now route and returns the execution the API wraps.
func TestHostedAgentTriggersRun(t *testing.T) {
	var got *http.Request
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"execution":{"execution_id":"ex_1","trigger_id":"tr_1","status":"pending"}}`))
	}))
	defer srv.Close()
	client, err := godo.New(http.DefaultClient, godo.SetBaseURL(srv.URL+"/"))
	require.NoError(t, err)

	e, err := do.NewHostedAgentTriggersService(client).Run("tr_1")
	require.NoError(t, err)
	assert.Equal(t, http.MethodPost, got.Method)
	assert.Equal(t, "/v2/harness-runtime/triggers/tr_1/executions", got.URL.Path)
	assert.Equal(t, "ex_1", e.ExecutionID)
	assert.Equal(t, godo.HostedAgentTriggerExecutionStatusPending, e.Status)
}

// A refusal keeps the API's status, so callers can tell a paused trigger
// (409) or a missing route (404) apart.
func TestHostedAgentTriggersRunKeepsTheAPIStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write([]byte(`{"id":"conflict","message":"trigger is paused"}`))
	}))
	defer srv.Close()
	client, err := godo.New(http.DefaultClient, godo.SetBaseURL(srv.URL+"/"))
	require.NoError(t, err)
	_, err = do.NewHostedAgentTriggersService(client).Run("tr_1")
	var er *godo.ErrorResponse
	require.ErrorAs(t, err, &er)
	assert.Equal(t, http.StatusConflict, er.Response.StatusCode)
	assert.Contains(t, er.Message, "paused")
}

// CreateKeepingSessionAwake posts the godo request plus keep_session_awake.
func TestHostedAgentTriggersCreateKeepingSessionAwake(t *testing.T) {
	var got map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&got)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"trigger":{"trigger_id":"tr_1","kind":"cron","session_mode":"reuse"}}`))
	}))
	defer srv.Close()
	client, err := godo.New(http.DefaultClient, godo.SetBaseURL(srv.URL+"/"))
	require.NoError(t, err)
	res, err := do.NewHostedAgentTriggersService(client).CreateKeepingSessionAwake(&godo.HostedAgentTriggerCreateRequest{
		Kind: godo.HostedAgentTriggerKindCron, Name: "n", SessionMode: godo.HostedAgentTriggerSessionModeReuse,
		PromptTemplate: "p", Output: godo.HostedAgentTriggerOutputWrite{Mode: godo.HostedAgentTriggerOutputModeNone},
		BoundSessionID: "sess_1", Cron: &godo.HostedAgentCreateCronConfig{CronExpr: "0 9 * * *", Timezone: "UTC"},
	})
	require.NoError(t, err)
	assert.Equal(t, true, got["keep_session_awake"])
	assert.Equal(t, "sess_1", got["bound_session_id"])
	assert.Equal(t, "reuse", got["session_mode"])
	assert.Equal(t, "tr_1", res.Trigger.TriggerID)
}

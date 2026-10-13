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

package displayers

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- HostedAgentTrigger JSON shape -------------------------------------------

// List verbs must always emit a JSON array regardless of row count.
// Get/mutate verbs (Single=true) must emit a bare JSON object.

func TestHostedAgentTriggerJSON_ListEmpty(t *testing.T) {
	var buf bytes.Buffer
	d := &HostedAgentTrigger{Triggers: nil}
	require.NoError(t, d.JSON(&buf))

	var out []any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &out), "list with 0 items must be a JSON array")
	assert.Len(t, out, 0)
}

func TestHostedAgentTriggerJSON_ListOneItem(t *testing.T) {
	var buf bytes.Buffer
	d := &HostedAgentTrigger{
		Triggers: []do.HostedAgentTrigger{{
			HostedAgentTrigger: &godo.HostedAgentTrigger{TriggerID: "tr_1"},
		}},
		// Single defaults to false → list semantics
	}
	require.NoError(t, d.JSON(&buf))

	var out []any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &out), "list with 1 item must still be a JSON array, not a bare object")
	assert.Len(t, out, 1)
}

func TestHostedAgentTriggerJSON_ListTwoItems(t *testing.T) {
	var buf bytes.Buffer
	d := &HostedAgentTrigger{
		Triggers: []do.HostedAgentTrigger{
			{HostedAgentTrigger: &godo.HostedAgentTrigger{TriggerID: "tr_1"}},
			{HostedAgentTrigger: &godo.HostedAgentTrigger{TriggerID: "tr_2"}},
		},
	}
	require.NoError(t, d.JSON(&buf))

	var out []any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &out), "list with 2 items must be a JSON array")
	assert.Len(t, out, 2)
}

func TestHostedAgentTriggerJSON_GetSingleItem(t *testing.T) {
	var buf bytes.Buffer
	d := &HostedAgentTrigger{
		Triggers: []do.HostedAgentTrigger{{
			HostedAgentTrigger: &godo.HostedAgentTrigger{TriggerID: "tr_1"},
		}},
		Single: true,
	}
	require.NoError(t, d.JSON(&buf))

	var out map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &out), "get (Single=true) with 1 item must be a bare JSON object, not an array")
	assert.Equal(t, "tr_1", out["trigger_id"])
}

// --- HostedAgentTriggerExecution JSON shape ----------------------------------

func TestHostedAgentTriggerExecutionJSON_ListOneItem(t *testing.T) {
	var buf bytes.Buffer
	d := &HostedAgentTriggerExecution{
		Executions: []do.HostedAgentTriggerExecution{{
			HostedAgentTriggerExecution: &godo.HostedAgentTriggerExecution{ExecutionID: "ex_1"},
		}},
		// Single=false → list semantics
	}
	require.NoError(t, d.JSON(&buf))

	var out []any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &out), "list-executions with 1 row must be a JSON array")
	assert.Len(t, out, 1)
}

func TestHostedAgentTriggerExecutionJSON_GetSingleItem(t *testing.T) {
	var buf bytes.Buffer
	d := &HostedAgentTriggerExecution{
		Executions: []do.HostedAgentTriggerExecution{{
			HostedAgentTriggerExecution: &godo.HostedAgentTriggerExecution{ExecutionID: "ex_1"},
		}},
		Single: true,
	}
	require.NoError(t, d.JSON(&buf))

	var out map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &out), "get-execution (Single=true) must be a bare JSON object")
	assert.Equal(t, "ex_1", out["execution_id"])
}

// --- HostedAgentWebhookProvider JSON shape (always list) --------------------

func TestHostedAgentWebhookProviderJSON_OneItem(t *testing.T) {
	var buf bytes.Buffer
	d := &HostedAgentWebhookProvider{
		Providers: []do.HostedAgentWebhookProvider{{
			HostedAgentWebhookProvider: &godo.HostedAgentWebhookProvider{Key: "github"},
		}},
	}
	require.NoError(t, d.JSON(&buf))

	var out []any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &out), "list-providers with 1 item must always be a JSON array")
	assert.Len(t, out, 1)
}

// --- HostedAgentReusableSession JSON shape (always list) --------------------

func TestHostedAgentReusableSessionJSON_OneItem(t *testing.T) {
	var buf bytes.Buffer
	d := &HostedAgentReusableSession{
		Sessions: []do.HostedAgentReusableSession{{
			HostedAgentReusableSession: &godo.HostedAgentReusableSession{SessionID: "sess_1"},
		}},
	}
	require.NoError(t, d.JSON(&buf))

	var out []any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &out), "list-reusable-sessions with 1 item must always be a JSON array")
	assert.Len(t, out, 1)
}

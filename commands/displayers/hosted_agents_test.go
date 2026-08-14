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

// --- HostedAgentSession JSON shape (MARSOHS-887) -----------------------------

// List verbs must always emit a JSON array regardless of row count. Get/mutate
// verbs (Single=true) must emit a bare JSON object. Same contract as
// HostedAgentTrigger (MARSOHS-869); regressed here once for `agents list`
// before being fixed alongside the triggers displayers.

func TestHostedAgentSessionJSON_ListEmpty(t *testing.T) {
	var buf bytes.Buffer
	d := &HostedAgentSession{Sessions: nil}
	require.NoError(t, d.JSON(&buf))

	var out []any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &out), "list with 0 items must be a JSON array")
	assert.Len(t, out, 0)
}

func TestHostedAgentSessionJSON_ListOneItem(t *testing.T) {
	var buf bytes.Buffer
	d := &HostedAgentSession{
		Sessions: []do.HostedAgentSession{{
			HostedAgentSession: &godo.HostedAgentSession{SessionID: "sess_1"},
		}},
		// Single defaults to false → list semantics
	}
	require.NoError(t, d.JSON(&buf))

	var out []any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &out), "list with 1 item must still be a JSON array, not a bare object")
	assert.Len(t, out, 1)
}

func TestHostedAgentSessionJSON_ListTwoItems(t *testing.T) {
	var buf bytes.Buffer
	d := &HostedAgentSession{
		Sessions: []do.HostedAgentSession{
			{HostedAgentSession: &godo.HostedAgentSession{SessionID: "sess_1"}},
			{HostedAgentSession: &godo.HostedAgentSession{SessionID: "sess_2"}},
		},
	}
	require.NoError(t, d.JSON(&buf))

	var out []any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &out), "list with 2 items must be a JSON array")
	assert.Len(t, out, 2)
}

func TestHostedAgentSessionJSON_GetSingleItem(t *testing.T) {
	var buf bytes.Buffer
	d := &HostedAgentSession{
		Sessions: []do.HostedAgentSession{{
			HostedAgentSession: &godo.HostedAgentSession{SessionID: "sess_1"},
		}},
		Single: true,
	}
	require.NoError(t, d.JSON(&buf))

	var out map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &out), "get/create (Single=true) with 1 item must be a bare JSON object, not an array")
	assert.Equal(t, "sess_1", out["session_id"])
}

// --- HostedAgentWorkspaceUpload JSON shape ------------------------------------
//
// Upload is currently single-file-only (always exactly one item), so this
// isn't user-visible today, but it still carries the same len==1 special case
// that caused MARSOHS-869/887. Normalized to the Single bool pattern so a
// future batch-upload can't silently regress into the same bug.

func TestHostedAgentWorkspaceUploadJSON_ListOneItem(t *testing.T) {
	var buf bytes.Buffer
	d := &HostedAgentWorkspaceUpload{
		Uploads: []*godo.HostedAgentWorkspaceUploadResponse{{Path: "src/main.go", BytesWritten: 42}},
	}
	require.NoError(t, d.JSON(&buf))

	var out []any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &out), "list semantics: 1 item must still be a JSON array")
	assert.Len(t, out, 1)
}

func TestHostedAgentWorkspaceUploadJSON_Single(t *testing.T) {
	var buf bytes.Buffer
	d := &HostedAgentWorkspaceUpload{
		Uploads: []*godo.HostedAgentWorkspaceUploadResponse{{Path: "src/main.go", BytesWritten: 42}},
		Single:  true,
	}
	require.NoError(t, d.JSON(&buf))

	var out map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &out), "Single=true must be a bare JSON object")
	assert.Equal(t, "src/main.go", out["path"])
}

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
	"time"

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

// MARSOHS-1438: -o json must preserve API fields that live on the godo
// HostedAgentSession wire type (sandbox_id, resume_on_topoff).
func TestHostedAgentSessionJSON_PreservesSandboxIDAndResumeOnTopoff(t *testing.T) {
	var buf bytes.Buffer
	d := &HostedAgentSession{
		Sessions: []do.HostedAgentSession{{
			HostedAgentSession: &godo.HostedAgentSession{
				SessionID:      "sess_1",
				SandboxID:      "sbx_1",
				ResumeOnTopoff: true,
			},
		}},
		Single: true,
	}
	require.NoError(t, d.JSON(&buf))

	var out map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &out))
	assert.Equal(t, "sbx_1", out["sandbox_id"])
	assert.Equal(t, true, out["resume_on_topoff"])
}

// --- HostedAgentSession pause reason -----------------------------------------

func TestHostedAgentSessionJSON_CarriesPauseReason(t *testing.T) {
	var buf bytes.Buffer
	d := &HostedAgentSession{
		Sessions: []do.HostedAgentSession{{
			HostedAgentSession: &godo.HostedAgentSession{
				SessionID:   "sess_1",
				Status:      godo.HostedAgentSessionStatusPaused,
				PauseReason: godo.HostedAgentSessionPauseReasonZeroBalance,
			},
		}},
		Single: true,
	}
	require.NoError(t, d.JSON(&buf))

	var out map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &out))
	assert.Equal(t, "sess_1", out["session_id"], "godo's fields must still be flat, not nested under the wrapper")
	assert.Equal(t, "zero_balance", out["pause_reason"])
}

func TestHostedAgentSessionJSON_OmitsEmptyPauseReason(t *testing.T) {
	var buf bytes.Buffer
	d := &HostedAgentSession{
		Sessions: []do.HostedAgentSession{{
			HostedAgentSession: &godo.HostedAgentSession{SessionID: "sess_1"},
		}},
		Single: true,
	}
	require.NoError(t, d.JSON(&buf))

	var out map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &out))
	assert.NotContains(t, out, "pause_reason", "a running session has no pause reason to report")
}

// The column is only carried when it has something to say, so an ordinary list
// does not grow a blank column.
func TestHostedAgentSessionCols_PauseReasonIsConditional(t *testing.T) {
	running := &HostedAgentSession{
		Sessions: []do.HostedAgentSession{
			{HostedAgentSession: &godo.HostedAgentSession{SessionID: "sess_1"}},
		},
	}
	assert.NotContains(t, running.Cols(), "PauseReason")

	paused := &HostedAgentSession{
		Sessions: []do.HostedAgentSession{
			{HostedAgentSession: &godo.HostedAgentSession{SessionID: "sess_1"}},
			{
				HostedAgentSession: &godo.HostedAgentSession{
					SessionID:   "sess_2",
					PauseReason: godo.HostedAgentSessionPauseReasonIdle,
				},
			},
		},
	}
	assert.Contains(t, paused.Cols(), "PauseReason")

	// With nothing to display the column still appears, so --format documents it.
	assert.Contains(t, (&HostedAgentSession{}).Cols(), "PauseReason")

	// ColMap always knows it, so --format PauseReason works either way.
	assert.Equal(t, "Pause Reason", running.ColMap()["PauseReason"])
}

func TestHostedAgentSessionKV_PassesUnknownReasonVerbatim(t *testing.T) {
	d := &HostedAgentSession{
		Sessions: []do.HostedAgentSession{{
			HostedAgentSession: &godo.HostedAgentSession{
				SessionID: "sess_1",
				// The API reserves the right to add reasons; an unrecognized
				// one must not be flattened to "unknown".
				PauseReason: "some_future_reason",
			},
		}},
	}

	kv := d.KV()
	require.Len(t, kv, 1)
	assert.Equal(t, "some_future_reason", kv[0]["PauseReason"])
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

func TestHostedAgentTemplateJSON_ListOneItem(t *testing.T) {
	var buf bytes.Buffer
	d := &HostedAgentTemplate{
		Templates: []godo.HostedAgentTemplate{{TemplateID: "tpl-1", Name: "my-image"}},
	}
	require.NoError(t, d.JSON(&buf))

	var out []any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &out), "list with 1 item must still be a JSON array")
	assert.Len(t, out, 1)
}

func TestHostedAgentTemplateJSON_GetSingleItem(t *testing.T) {
	var buf bytes.Buffer
	d := &HostedAgentTemplate{
		Templates: []godo.HostedAgentTemplate{{TemplateID: "tpl-1", Name: "my-image"}},
		Single:    true,
	}
	require.NoError(t, d.JSON(&buf))

	var out map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &out), "Single=true must be a bare JSON object")
	assert.Equal(t, "tpl-1", out["template_id"])
}

// --- HostedAgentSession workspace ----------------------------------------------

func TestHostedAgentSessionJSON_CarriesWorkspaceID(t *testing.T) {
	var buf bytes.Buffer
	d := &HostedAgentSession{
		Sessions: []do.HostedAgentSession{
			{HostedAgentSession: &godo.HostedAgentSession{SessionID: "sess_1", WorkspaceID: "ws_1"}},
		},
		Single: true,
	}
	require.NoError(t, d.JSON(&buf))
	var out map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &out))
	assert.Equal(t, "ws_1", out["workspace_id"])

	buf.Reset()
	d.Sessions[0].WorkspaceID = ""
	require.NoError(t, d.JSON(&buf))
	out = nil
	require.NoError(t, json.Unmarshal(buf.Bytes(), &out))
	assert.NotContains(t, out, "workspace_id")
}

// Like the pause reason, the column only appears when a session has one.
func TestHostedAgentSessionCols_WorkspaceIsConditional(t *testing.T) {
	plain := &HostedAgentSession{
		Sessions: []do.HostedAgentSession{{HostedAgentSession: &godo.HostedAgentSession{SessionID: "sess_1"}}},
	}
	assert.NotContains(t, plain.Cols(), "WorkspaceID")

	held := &HostedAgentSession{
		Sessions: []do.HostedAgentSession{
			{HostedAgentSession: &godo.HostedAgentSession{SessionID: "sess_1"}},
			{HostedAgentSession: &godo.HostedAgentSession{SessionID: "sess_2", WorkspaceID: "ws_1"}},
		},
	}
	assert.Contains(t, held.Cols(), "WorkspaceID")
	assert.Contains(t, (&HostedAgentSession{}).Cols(), "WorkspaceID", "still discoverable in --format help")
	assert.Equal(t, "Workspace", plain.ColMap()["WorkspaceID"])

	kv := held.KV()
	require.Len(t, kv, 2)
	assert.Equal(t, "", kv[0]["WorkspaceID"])
	assert.Equal(t, "ws_1", kv[1]["WorkspaceID"])
}

// --- HostedAgentWorkspace -----------------------------------------------------

func TestHostedAgentWorkspaceJSON(t *testing.T) {
	ws := godo.HostedAgentWorkspace{WorkspaceID: "ws_1", Name: "notes", State: godo.HostedAgentWorkspaceStateAvailable, SizeGibibytes: 10}

	var buf bytes.Buffer
	require.NoError(t, (&HostedAgentWorkspace{Workspaces: []godo.HostedAgentWorkspace{ws}}).JSON(&buf))
	var list []map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &list), "list with 1 item must still be a JSON array")
	require.Len(t, list, 1)
	assert.Equal(t, "ws_1", list[0]["workspace_id"])

	buf.Reset()
	require.NoError(t, (&HostedAgentWorkspace{Workspaces: []godo.HostedAgentWorkspace{ws}, Single: true}).JSON(&buf))
	var one map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &one), "get/create must be a bare object")
	assert.Equal(t, "AVAILABLE", one["state"])
	assert.NotContains(t, one, "attached_session_id")
	assert.NotContains(t, one, "last_saved_at")

	buf.Reset()
	require.NoError(t, (&HostedAgentWorkspace{}).JSON(&buf))
	assert.JSONEq(t, "[]", buf.String(), "an empty list is an array, not null")
}

func TestHostedAgentWorkspaceColsAndKV(t *testing.T) {
	d := &HostedAgentWorkspace{}
	assert.Equal(t,
		[]string{"ID", "Name", "State", "Size (GiB)", "Used", "Attached Session", "Last Saved", "Created"},
		func() []string {
			var out []string
			for _, c := range d.Cols() {
				out = append(out, d.ColMap()[c])
			}
			return out
		}())

	saved := godo.Timestamp{Time: time.Date(2026, 10, 5, 12, 0, 0, 0, time.UTC)}
	d.Workspaces = []godo.HostedAgentWorkspace{
		{
			WorkspaceID:       "ws_1",
			Name:              "notes",
			State:             godo.HostedAgentWorkspaceStateAttached,
			AttachedSessionID: "sess_1",
			SizeGibibytes:     10,
			BytesUsed:         3 * 1024 * 1024,
			LastSavedAt:       &saved,
			CreatedAt:         godo.Timestamp{Time: time.Date(2026, 10, 1, 8, 30, 0, 0, time.UTC)},
		},
		// Never saved, nothing attached.
		{WorkspaceID: "ws_2", State: godo.HostedAgentWorkspaceStateAvailable, SizeGibibytes: 1},
	}
	kv := d.KV()
	require.Len(t, kv, 2)
	assert.Equal(t, "ws_1", kv[0]["WorkspaceID"])
	assert.Equal(t, "ATTACHED", kv[0]["State"])
	assert.Equal(t, int32(10), kv[0]["SizeGibibytes"])
	assert.Equal(t, "3.00 MiB", kv[0]["BytesUsed"])
	assert.Equal(t, "sess_1", kv[0]["AttachedSessionID"])
	assert.Equal(t, "2026-10-05T12:00:00Z", kv[0]["LastSavedAt"])
	assert.Equal(t, "2026-10-01T08:30:00Z", kv[0]["CreatedAt"])

	assert.Equal(t, "", kv[1]["LastSavedAt"], "not saved yet is blank, not a zero time")
	assert.Equal(t, "", kv[1]["AttachedSessionID"])
	assert.Equal(t, "0 B", kv[1]["BytesUsed"])
}

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

func strPtr(s string) *string { return &s }
func i64Ptr(v int64) *int64   { return &v }

func TestSignalsDeletion_KV_NullFields(t *testing.T) {
	d := &SignalsDeletion{Deletions: do.SignalsDeletions{{
		SignalsDeletionJob: &godo.SignalsDeletionJob{
			TeamID: 42, DeletionID: "01J", Type: "inference", Status: "queued", CreatedAt: 100,
		},
	}}}

	kv := d.KV()
	require.Len(t, kv, 1)
	assert.Equal(t, "", kv[0]["StartedAt"])
	assert.Equal(t, "", kv[0]["CompletedAt"])
	assert.Equal(t, "", kv[0]["ErrorMessage"])
	assert.Equal(t, "", kv[0]["AgentID"])
}

func TestSignalsDeletion_KV_AllFields(t *testing.T) {
	d := &SignalsDeletion{Deletions: do.SignalsDeletions{{
		SignalsDeletionJob: &godo.SignalsDeletionJob{
			TeamID: 42, DeletionID: "01J", Type: "agent", AgentID: "agent-1",
			Status: "failed", ErrorMessage: strPtr("worker timeout"),
			CreatedAt: 100, StartedAt: i64Ptr(110), CompletedAt: i64Ptr(120),
		},
	}}}

	kv := d.KV()
	require.Len(t, kv, 1)
	assert.Equal(t, "01J", kv[0]["DeletionID"])
	assert.Equal(t, int64(42), kv[0]["TeamID"])
	assert.Equal(t, "agent", kv[0]["Type"])
	assert.Equal(t, "agent-1", kv[0]["AgentID"])
	assert.Equal(t, "failed", kv[0]["Status"])
	assert.Equal(t, "worker timeout", kv[0]["ErrorMessage"])
	assert.Equal(t, int64(100), kv[0]["CreatedAt"])
	assert.Equal(t, "110", kv[0]["StartedAt"])
	assert.Equal(t, "120", kv[0]["CompletedAt"])
}

func TestSignalsDeletion_ColsAreInColMap(t *testing.T) {
	d := &SignalsDeletion{}
	cm := d.ColMap()
	for _, c := range d.Cols() {
		assert.Contains(t, cm, c)
	}
	assert.Contains(t, cm, "ErrorMessage")
	assert.Contains(t, cm, "TeamID")
}

func TestSignalsDeletion_JSON(t *testing.T) {
	d := &SignalsDeletion{Deletions: do.SignalsDeletions{
		{SignalsDeletionJob: &godo.SignalsDeletionJob{DeletionID: "01J", Type: "agent", AgentID: "agent-1", Status: "queued"}},
		{SignalsDeletionJob: &godo.SignalsDeletionJob{DeletionID: "01K", Type: "inference", Status: "queued"}},
	}}

	var buf bytes.Buffer
	require.NoError(t, d.JSON(&buf))

	var got []map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &got))
	require.Len(t, got, 2)
	assert.Equal(t, "01J", got[0]["deletion_id"])
	assert.Equal(t, "agent-1", got[0]["agent_id"])
	_, has := got[1]["agent_id"]
	assert.False(t, has, "inference job must not have an agent_id key")
}

func TestSignalsDeletion_JSON_EmptyIsArray(t *testing.T) {
	d := &SignalsDeletion{Deletions: do.SignalsDeletions{}}
	var buf bytes.Buffer
	require.NoError(t, d.JSON(&buf))
	assert.JSONEq(t, "[]", buf.String())
}

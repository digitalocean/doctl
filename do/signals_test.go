package do

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const liveExportJobJSON = `{
	"export_id": "550e8400-e29b-41d4-a716-446655440000",
	"agent_id": "a1b2c3d4-e29b-41d4-a716-446655440000",
	"status": "queued",
	"filters": {"signal_type":["MisalignmentCorrection"],"session_ids":["sess-1"],"start_time":1,"end_time":2},
	"created_at": 1754049600,
	"completed_at": null,
	"expires_at": null,
	"error_message": null
}`

func newTestSignalsService(t *testing.T, h http.HandlerFunc) (SignalsService, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	client := godo.NewClient(nil)
	u, err := url.Parse(srv.URL + "/")
	require.NoError(t, err)
	client.BaseURL = u
	return NewSignalsService(client), srv
}

func TestCreateExport_SendsSignalTypeNotSignalTypes(t *testing.T) {
	svc, _ := newTestSignalsService(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/v1/signals/exports", r.URL.Path)
		var raw map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&raw))
		_, has := raw["signal_types"]
		assert.False(t, has)
		assert.Equal(t, []any{"MisalignmentCorrection"}, raw["signal_type"])
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(liveExportJobJSON))
	})

	job, err := svc.CreateExport(&godo.SignalsCreateExportRequest{
		AgentID:    "a1b2c3d4-e29b-41d4-a716-446655440000",
		SignalType: []string{"MisalignmentCorrection"},
	})
	require.NoError(t, err)
	assert.Equal(t, "550e8400-e29b-41d4-a716-446655440000", job.ExportID)
	assert.Equal(t, "queued", job.Status)
	assert.Equal(t, int64(1754049600), job.CreatedAt)
	assert.Equal(t, []string{"MisalignmentCorrection"}, job.Filters.SignalType)
	assert.Equal(t, []string{"sess-1"}, job.Filters.SessionIDs)
}

func TestListExports_ReadsEdgesNotExports(t *testing.T) {
	svc, _ := newTestSignalsService(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/signals/exports", r.URL.Path)
		assert.Equal(t, "agt", r.URL.Query().Get("agent_id"))
		assert.Equal(t, "20", r.URL.Query().Get("limit"))
		_, _ = w.Write([]byte(`{"edges":[{"cursor":"c1","node":` + liveExportJobJSON + `}],"page_info":{"has_next_page":true,"end_cursor":"c1"}}`))
	})

	jobs, err := svc.ListExports(&godo.SignalsListExportsOptions{
		SignalsCursorPageOptions: godo.SignalsCursorPageOptions{Limit: 20},
		AgentID:                  "agt",
	})
	require.NoError(t, err)
	require.Len(t, jobs, 1)
	assert.Equal(t, "550e8400-e29b-41d4-a716-446655440000", jobs[0].ExportID)
}

func TestGetExportDownloadAndOptions(t *testing.T) {
	svc, _ := newTestSignalsService(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/v1/signals/exports/exp-1/download":
			_, _ = w.Write([]byte(`{"download_url":"https://x","expires_at":9}`))
		case r.URL.Path == "/v1/signals/exports/options":
			_, _ = w.Write([]byte(`{"filters":{"signal_type":["A"]}}`))
		default:
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
	})

	dl, err := svc.GetExportDownload("exp-1")
	require.NoError(t, err)
	assert.Equal(t, "https://x", dl.DownloadURL)

	opts, err := svc.GetExportOptions()
	require.NoError(t, err)
	assert.Equal(t, []string{"A"}, opts.Filters.SignalType)
}

func TestSetConsentUnwrapsConsentEnvelope(t *testing.T) {
	svc, _ := newTestSignalsService(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPut, r.Method)
		assert.Equal(t, "/v1/consent/agt-1", r.URL.Path)
		_, _ = w.Write([]byte(`{"consent":{"id":1,"team_id":42,"agent_id":"agt-1","enabled":true,"updated_at":"2026-07-28T06:00:00Z"}}`))
	})
	c, err := svc.SetConsent("agt-1", true)
	require.NoError(t, err)
	assert.Equal(t, int64(1), c.ID)
	assert.True(t, c.Enabled)
}

func TestListAgentSessions(t *testing.T) {
	svc, _ := newTestSignalsService(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/signals/agents/agt-1/sessions", r.URL.Path)
		assert.Equal(t, "10", r.URL.Query().Get("limit"))
		_, _ = w.Write([]byte(`{"edges":[{"cursor":"c1","node":{"session_id":"sess-1","total_turns":5,"started_at":"2026-01-01T00:00:00Z","duration_seconds":300,"signal_count":2}}],"page_info":{"has_next_page":false}}`))
	})

	sessions, err := svc.ListAgentSessions("agt-1", &godo.SignalsListAgentSessionsOptions{
		SignalsCursorPageOptions: godo.SignalsCursorPageOptions{Limit: 10},
	})
	require.NoError(t, err)
	require.Len(t, sessions, 1)
	assert.Equal(t, "sess-1", sessions[0].SessionID)
	assert.Equal(t, 5, sessions[0].TotalTurns)
	assert.Equal(t, 2, sessions[0].SignalCount)
}

func TestListSessionDialogues(t *testing.T) {
	svc, _ := newTestSignalsService(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/signals/sessions/sess-1/dialogues", r.URL.Path)
		_, _ = w.Write([]byte(`{"session_id":"sess-1","edges":[{"cursor":"c1","node":{"id":1,"run_id":"run-1","created_at":"2026-01-01T00:00:00Z","sequence":1,"user_message":"hello","steps":[],"run_status":"complete","segment_id":"seg-1","segment_seq":0,"signals":[]}}],"page_info":{"has_next_page":false}}`))
	})

	dialogues, err := svc.ListSessionDialogues("sess-1", &godo.SignalsListDialoguesOptions{
		SignalsCursorPageOptions: godo.SignalsCursorPageOptions{Limit: 10},
	})
	require.NoError(t, err)
	require.Len(t, dialogues, 1)
	assert.Equal(t, int64(1), dialogues[0].ID)
	assert.Equal(t, "hello", dialogues[0].UserMessage)
}

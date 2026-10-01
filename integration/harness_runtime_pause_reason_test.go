package integration

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"os/exec"
	"testing"

	"github.com/sclevine/spec"
	"github.com/stretchr/testify/require"
)

// pause_reason has no godo field, so doctl decodes the session endpoints
// itself. These exercise that decode over real HTTP: if the local transport
// ever regresses to godo's, the field silently disappears and these fail.
var _ = suite("harness-runtime/pause-reason", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		server *httptest.Server
		// lastListQuery records what the CLI asked the API for.
		lastListQuery string
	)

	const sessionsBody = `{
		"sessions": [
			{
				"session_id": "sess_broke",
				"name": "broke",
				"agent_kind": "AGENT_KIND_OPENCODE",
				"status": "SESSION_STATUS_PAUSED",
				"pause_reason": "zero_balance",
				"created_at": "2026-09-16T08:44:38Z",
				"last_event_at": "2026-09-16T09:10:00Z"
			},
			{
				"session_id": "sess_idle",
				"name": "napping",
				"agent_kind": "AGENT_KIND_OPENCODE",
				"status": "SESSION_STATUS_PAUSED",
				"pause_reason": "idle",
				"created_at": "2026-09-16T08:44:38Z",
				"last_event_at": "2026-09-16T09:10:00Z"
			}
		],
		"next_page_token": ""
	}`

	it.Before(func() {
		expect = require.New(t)
		lastListQuery = ""

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			w.Header().Add("content-type", "application/json")

			switch req.URL.Path {
			case "/v2/agents/sessions":
				lastListQuery = req.URL.RawQuery
				w.Write([]byte(sessionsBody))
			case "/v2/agents/sessions/sess_broke":
				w.Write([]byte(`{
					"session": {
						"session_id": "sess_broke",
						"name": "broke",
						"agent_kind": "AGENT_KIND_OPENCODE",
						"status": "SESSION_STATUS_PAUSED",
						"pause_reason": "zero_balance",
						"created_at": "2026-09-16T08:44:38Z",
						"last_event_at": "2026-09-16T09:10:00Z"
					}
				}`))
			default:
				dump, err := httputil.DumpRequest(req, true)
				if err != nil {
					t.Fatal("failed to dump request")
				}

				t.Fatalf("received unknown request: %s", dump)
			}
		}))
	})

	it("carries pause_reason through show -o json", func() {
		cmd := exec.Command(builtBinaryPath,
			"-t", "some-magic-token",
			"-u", server.URL,
			"-o", "json",
			"harness-runtime", "show", "sess_broke",
		)

		output, err := cmd.CombinedOutput()
		expect.NoError(err, string(output))

		var got map[string]any
		expect.NoError(json.Unmarshal(output, &got))
		// The wrapper must stay flat: godo's fields alongside pause_reason,
		// not nested under an embedded object.
		expect.Equal("sess_broke", got["session_id"])
		expect.Equal("zero_balance", got["pause_reason"])
	})

	it("names the pause reason in the show card", func() {
		cmd := exec.Command(builtBinaryPath,
			"-t", "some-magic-token",
			"-u", server.URL,
			"harness-runtime", "show", "sess_broke",
		)

		output, err := cmd.CombinedOutput()
		expect.NoError(err, string(output))
		expect.Contains(string(output), "zero_balance")
		// A zero-balance pause cannot be resolved by launching, so the card
		// points at the balance first.
		expect.Contains(string(output), "cloud.digitalocean.com/account/billing")
	})

	it("filters list by pause reason and asks the API for paused sessions", func() {
		cmd := exec.Command(builtBinaryPath,
			"-t", "some-magic-token",
			"-u", server.URL,
			"-o", "json",
			"harness-runtime", "list", "--paused-by", "zero-balance",
		)

		output, err := cmd.CombinedOutput()
		expect.NoError(err, string(output))

		var got []map[string]any
		expect.NoError(json.Unmarshal(output, &got))
		expect.Len(got, 1, "the idle session must be filtered out client-side")
		expect.Equal("sess_broke", got[0]["session_id"])
		expect.Equal("zero_balance", got[0]["pause_reason"])

		// The flag has no server-side equivalent, so the most the request can
		// do is narrow to paused.
		expect.Contains(lastListQuery, "status=SESSION_STATUS_PAUSED")
	})

	// backward-compat: old servers still return "low_balance"; --paused-by low-balance
	// must still filter those sessions correctly.
	it("legacy --paused-by low-balance still matches low_balance sessions from old servers (backward-compat)", func() {
		legacyBody := `{
			"sessions": [
				{
					"session_id": "sess_broke_legacy",
					"name": "broke-legacy",
					"agent_kind": "AGENT_KIND_OPENCODE",
					"status": "SESSION_STATUS_PAUSED",
					"pause_reason": "low_balance",
					"created_at": "2026-09-16T08:44:38Z",
					"last_event_at": "2026-09-16T09:10:00Z"
				},
				{
					"session_id": "sess_idle_legacy",
					"name": "napping-legacy",
					"agent_kind": "AGENT_KIND_OPENCODE",
					"status": "SESSION_STATUS_PAUSED",
					"pause_reason": "idle",
					"created_at": "2026-09-16T08:44:38Z",
					"last_event_at": "2026-09-16T09:10:00Z"
				}
			],
			"next_page_token": ""
		}`
		legacySrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			w.Header().Add("content-type", "application/json")
			w.Write([]byte(legacyBody))
		}))
		defer legacySrv.Close()

		cmd := exec.Command(builtBinaryPath,
			"-t", "some-magic-token",
			"-u", legacySrv.URL,
			"-o", "json",
			"harness-runtime", "list", "--paused-by", "low-balance",
		)

		output, err := cmd.CombinedOutput()
		expect.NoError(err, string(output))

		var got []map[string]any
		expect.NoError(json.Unmarshal(output, &got))
		expect.Len(got, 1, "legacy flag must still match low_balance sessions from old servers")
		expect.Equal("sess_broke_legacy", got[0]["session_id"])
		expect.Equal("low_balance", got[0]["pause_reason"])
	})

	it("rejects a pause reason paired with a non-paused status", func() {
		cmd := exec.Command(builtBinaryPath,
			"-t", "some-magic-token",
			"-u", server.URL,
			"harness-runtime", "list",
			"--paused-by", "zero-balance",
			"--status", "SESSION_STATUS_READY",
		)

		output, err := cmd.CombinedOutput()
		expect.Error(err)
		expect.Contains(string(output), "only applies to paused sessions")
	})
})

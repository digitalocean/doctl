package integration

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"strings"
	"sync"
	"testing"

	"github.com/sclevine/spec"
	"github.com/stretchr/testify/require"
)

const (
	deletionAgentUUID = "a1b2c3d4-e29b-41d4-a716-446655440000"
	deletionID        = "01J9ZX0N5Q7M2K3R4S5T6V7W8X"

	deletionManagedJSON = `{"team_id":42,"deletion_id":"` + deletionID + `","type":"agent","agent_id":"` + deletionAgentUUID + `","status":"queued","error_message":null,"created_at":1754049600,"started_at":null,"completed_at":null}`
	deletionInferJSON   = `{"team_id":42,"deletion_id":"01K","type":"inference","status":"queued","created_at":1754049600}`
)

type recordedRequest struct {
	Method string
	Path   string
	Query  string
	Body   string
}

// signalsDeletionServer is a fake signals API that records every request.
type signalsDeletionServer struct {
	*httptest.Server
	mu       sync.Mutex
	requests []recordedRequest

	// createStatus and createBody are the response for POST /v1/signals/deletions.
	// Change them with setCreate; the handler reads them under mu.
	createStatus int
	createBody   string
}

func (s *signalsDeletionServer) setCreate(status int, body string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if status != 0 {
		s.createStatus = status
	}
	if body != "" {
		s.createBody = body
	}
}

func newSignalsDeletionServer() *signalsDeletionServer {
	s := &signalsDeletionServer{createStatus: http.StatusAccepted, createBody: deletionManagedJSON}
	s.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		s.mu.Lock()
		s.requests = append(s.requests, recordedRequest{r.Method, r.URL.Path, r.URL.RawQuery, string(body)})
		createStatus, createBody := s.createStatus, s.createBody
		s.mu.Unlock()

		w.Header().Set("content-type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v1/consent":
			_, _ = w.Write([]byte(`{"team_id":42,"consents":[]}`))
		case r.Method == http.MethodPost && r.URL.Path == "/v1/signals/deletions":
			w.WriteHeader(createStatus)
			_, _ = w.Write([]byte(createBody))
		case r.Method == http.MethodGet && r.URL.Path == "/v1/signals/deletions/"+deletionID:
			_, _ = w.Write([]byte(deletionManagedJSON))
		case r.Method == http.MethodGet && r.URL.Path == "/v1/signals/deletions":
			_, _ = w.Write([]byte(`{"edges":[{"cursor":"c1","node":` + deletionManagedJSON + `},{"cursor":"c2","node":` + deletionInferJSON + `}],"page_info":{"has_next_page":true,"end_cursor":"c2"}}`))
		default:
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"id":"not_found","message":"not found"}`))
		}
	}))
	return s
}

func (s *signalsDeletionServer) recorded() []recordedRequest {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]recordedRequest(nil), s.requests...)
}

func (s *signalsDeletionServer) posts() []recordedRequest {
	var out []recordedRequest
	for _, r := range s.recorded() {
		if r.Method == http.MethodPost {
			out = append(out, r)
		}
	}
	return out
}

// runDoctl runs the built binary and returns stdout, stderr and the exit error.
func runDoctl(args ...string) (string, string, error) {
	cmd := exec.Command(builtBinaryPath, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

var _ = suite("signals/deletion/create", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		server *signalsDeletionServer
		base   []string
	)

	it.Before(func() {
		expect = require.New(t)
		server = newSignalsDeletionServer()
		base = []string{"-t", "some-magic-token", "-u", server.URL, "--http-retry-max", "0", "signals", "deletion", "create"}
	})

	it.After(func() { server.Close() })

	when("deleting a managed agent", func() {
		it("sends the exact request body and prints the job", func() {
			stdout, _, err := runDoctl(append(base, "--type", "agent", "--agent-id", deletionAgentUUID, "--team-id", "42", "--force")...)
			expect.NoError(err)

			posts := server.posts()
			expect.Len(posts, 1)
			expect.Equal("/v1/signals/deletions", posts[0].Path)
			expect.JSONEq(`{"type":"agent","team_id":42,"agent_id":"`+deletionAgentUUID+`"}`, posts[0].Body)
			expect.Contains(stdout, deletionID)
			expect.Contains(stdout, "queued")
		})
	})

	when("deleting inference data", func() {
		it("sends no agent_id", func() {
			server.setCreate(0, deletionInferJSON)
			_, _, err := runDoctl(append(base, "--type", "inference", "--team-id", "42", "--force")...)
			expect.NoError(err)

			posts := server.posts()
			expect.Len(posts, 1)
			expect.JSONEq(`{"type":"inference","team_id":42}`, posts[0].Body)
		})
	})

	when("--team-id is not given", func() {
		it("looks up the team id first, then uses it", func() {
			_, _, err := runDoctl(append(base, "--type", "agent", "--agent-id", deletionAgentUUID, "--force")...)
			expect.NoError(err)

			reqs := server.recorded()
			expect.Len(reqs, 2)
			expect.Equal("GET", reqs[0].Method)
			expect.Equal("/v1/consent", reqs[0].Path)
			expect.Equal("POST", reqs[1].Method)
			expect.JSONEq(`{"type":"agent","team_id":42,"agent_id":"`+deletionAgentUUID+`"}`, reqs[1].Body)
		})
	})

	when("there is no --force and no terminal", func() {
		it("refuses and sends nothing", func() {
			stdout, stderr, err := runDoctl(append(base, "--type", "inference", "--team-id", "42")...)
			expect.Error(err)
			expect.Empty(server.posts())
			// AskForConfirm warns on stdout and returns ErrExitSilently, so there is
			// no trailing "Error: Operation aborted." line (that one comes from other
			// delete commands).
			expect.Equal("Warning: Requires confirmation. Use the `--force` flag to continue without confirmation.", strings.TrimSpace(stdout+stderr))
		})
	})

	when("the server reuses an active job (200)", func() {
		it("shows the job, exits 0 and keeps stdout valid JSON", func() {
			server.setCreate(http.StatusOK, "")
			stdout, stderr, err := runDoctl(append(base, "-o", "json", "--type", "agent", "--agent-id", deletionAgentUUID, "--team-id", "42", "--force")...)
			expect.NoError(err)
			expect.Contains(stderr, "already exists")

			var jobs []map[string]any
			expect.NoError(json.Unmarshal([]byte(stdout), &jobs), stdout)
			expect.Len(jobs, 1)
			expect.Equal(deletionID, jobs[0]["deletion_id"])
		})
	})

	when("the server rejects the request", func() {
		for _, status := range []int{http.StatusTooManyRequests, http.StatusForbidden, http.StatusNotFound} {
			status := status
			it("fails with the server message for status "+http.StatusText(status), func() {
				server.setCreate(status, `{"id":"err","message":"server says no"}`)
				_, stderr, err := runDoctl(append(base, "--type", "inference", "--team-id", "42", "--force")...)
				expect.Error(err)
				expect.Contains(stderr, "server says no")
			})
		}
	})

	when("input is invalid", func() {
		it("requires --type", func() {
			_, stderr, err := runDoctl(append(base, "--team-id", "42", "--force")...)
			expect.Error(err)
			expect.Contains(stderr, "missing required arguments")
			expect.Contains(stderr, "type")
			expect.Empty(server.recorded())
		})

		it("rejects inference with --agent-id without calling the server", func() {
			_, stderr, err := runDoctl(append(base, "--type", "inference", "--agent-id", deletionAgentUUID, "--team-id", "42", "--force")...)
			expect.Error(err)
			expect.Contains(stderr, "--agent-id must not be set")
			expect.Empty(server.recorded())
		})

		it("rejects a non-UUID agent id without calling the server", func() {
			_, stderr, err := runDoctl(append(base, "--type", "agent", "--agent-id", "nope", "--team-id", "42", "--force")...)
			expect.Error(err)
			expect.Contains(stderr, "valid UUID")
			expect.Empty(server.recorded())
		})
	})
})

var _ = suite("signals/deletion/get", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		server *signalsDeletionServer
		base   []string
	)

	it.Before(func() {
		expect = require.New(t)
		server = newSignalsDeletionServer()
		base = []string{"-t", "some-magic-token", "-u", server.URL, "--http-retry-max", "0", "signals", "deletion"}
	})

	it.After(func() { server.Close() })

	it("prints the job as a table", func() {
		stdout, _, err := runDoctl(append(base, "get", deletionID)...)
		expect.NoError(err)
		expect.Contains(stdout, "Deletion ID")
		expect.Contains(stdout, deletionID)
		expect.Contains(stdout, "agent")
	})

	it("prints valid JSON with -o json", func() {
		stdout, _, err := runDoctl(append(base, "-o", "json", "get", deletionID)...)
		expect.NoError(err)
		var jobs []map[string]any
		expect.NoError(json.Unmarshal([]byte(stdout), &jobs), stdout)
		expect.Equal(deletionID, jobs[0]["deletion_id"])
	})

	it("fails for an unknown id", func() {
		_, stderr, err := runDoctl(append(base, "get", "missing")...)
		expect.Error(err)
		expect.Contains(stderr, "not found")
	})

	it("fails without an id", func() {
		_, _, err := runDoctl(append(base, "get")...)
		expect.Error(err)
		expect.Empty(server.recorded())
	})
})

var _ = suite("signals/deletion/list", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		server *signalsDeletionServer
		base   []string
	)

	it.Before(func() {
		expect = require.New(t)
		server = newSignalsDeletionServer()
		base = []string{"-t", "some-magic-token", "-u", server.URL, "--http-retry-max", "0", "signals", "deletion"}
	})

	it.After(func() { server.Close() })

	it("lists jobs and prints the next page token to stderr", func() {
		stdout, stderr, err := runDoctl(append(base, "list", "--limit", "5")...)
		expect.NoError(err)
		expect.Contains(stdout, deletionID)
		expect.Contains(stdout, "inference")
		expect.NotContains(stdout, "Next page token")
		expect.Contains(stderr, "Next page token: c2")

		reqs := server.recorded()
		expect.Len(reqs, 1)
		expect.Equal("limit=5", reqs[0].Query)
	})

	it("sends the after cursor", func() {
		_, _, err := runDoctl(append(base, "list", "--limit", "5", "--after", "c1")...)
		expect.NoError(err)
		expect.Contains(server.recorded()[0].Query, "after=c1")
	})

	it("keeps stdout valid JSON with -o json", func() {
		stdout, _, err := runDoctl(append(base, "-o", "json", "list")...)
		expect.NoError(err)
		var jobs []map[string]any
		expect.NoError(json.Unmarshal([]byte(stdout), &jobs), stdout)
		expect.Len(jobs, 2)
		expect.Equal("limit=20", server.recorded()[0].Query, "default --limit is 20")
		_, hasAgent := jobs[1]["agent_id"]
		expect.False(hasAgent, "inference job has no agent_id")
	})

	it("shows the failure reason with --format", func() {
		stdout, _, err := runDoctl(append(base, "list", "--format", "DeletionID,Status,ErrorMessage")...)
		expect.NoError(err)
		expect.True(strings.Contains(stdout, "Error Message"), stdout)
	})

	it("rejects an out-of-range limit without calling the server", func() {
		_, stderr, err := runDoctl(append(base, "list", "--limit", "101")...)
		expect.Error(err)
		expect.Contains(stderr, "--limit must be between 1 and 100")
		expect.Empty(server.recorded())
	})
})

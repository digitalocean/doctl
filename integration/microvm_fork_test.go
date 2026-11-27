package integration

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"os/exec"
	"sync/atomic"
	"testing"

	"github.com/sclevine/spec"
	"github.com/stretchr/testify/require"
)

var _ = suite("compute/microvm/fork", func(t *testing.T, when spec.G, it spec.S) {
	const sourceID = "b2a2f7a4-8d34-4c1c-9c66-3f2b7f8f38f2"

	var (
		expect *require.Assertions
		server *httptest.Server
		forks  atomic.Int32
		status int
	)

	it.Before(func() {
		expect = require.New(t)
		forks.Store(0)
		status = http.StatusAccepted

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/v2/microvms/" + sourceID + "/fork":
				if req.Header.Get("Authorization") != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}
				if req.Method != http.MethodPost {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}
				forks.Add(1)
				if status != http.StatusAccepted {
					w.WriteHeader(status)
					w.Write([]byte(`{"id":"bad_gateway","message":"upstream unavailable"}`))
					return
				}

				var body map[string]any
				expect.NoError(json.NewDecoder(req.Body).Decode(&body))
				expect.Equal(map[string]any{"count": float64(2), "name": "eval"}, body)

				w.WriteHeader(http.StatusAccepted)
				w.Write([]byte(microVMForkResponse))
			default:
				dump, err := httputil.DumpRequest(req, true)
				if err != nil {
					t.Fatal("failed to dump request")
				}

				t.Fatalf("received unknown request: %s", dump)
			}
		}))
	})

	it.After(func() {
		server.Close()
	})

	it("forks the microvm and reports the fork id", func() {
		cmd := exec.Command(builtBinaryPath,
			"-t", "some-magic-token",
			"-u", server.URL,
			"compute", "microvm", "fork", sourceID,
			"--count", "2",
			"--name", "eval",
			"--format", "Name,State,ForkID,ForkedFrom",
		)

		output, err := cmd.CombinedOutput()
		expect.NoError(err, fmt.Sprintf("received error output: %s", output))
		expect.Contains(string(output), "Fork 8f7a9a3f-5555-4444-9999-0000000000f0 accepted from checkpoint 8f7a9a3f-5555-4444-9999-000000000001")
		expect.Regexp(`eval-1\s+creating\s+8f7a9a3f-5555-4444-9999-0000000000f0\s+`+sourceID, string(output))
		expect.Regexp(`eval-2\s+creating\s+8f7a9a3f-5555-4444-9999-0000000000f0\s+`+sourceID, string(output))
	})

	// The default --http-retry-max is 5; a fork that was retried could fork
	// more than once.
	it("sends the fork once when the server fails", func() {
		status = http.StatusBadGateway

		cmd := exec.Command(builtBinaryPath,
			"-t", "some-magic-token",
			"-u", server.URL,
			"compute", "microvm", "fork", sourceID,
			"--count", "2",
			"--name", "eval",
		)

		output, err := cmd.CombinedOutput()
		expect.Error(err)
		expect.Equal(int32(1), forks.Load(), "fork must not be retried: %s", output)
		expect.Contains(string(output), "doctl compute microvm list --forked-from "+sourceID)
	})
})

const microVMForkResponse = `
{
  "fork_id": "8f7a9a3f-5555-4444-9999-0000000000f0",
  "checkpoint_id": "8f7a9a3f-5555-4444-9999-000000000001",
  "microvms": [
    {
      "id": "c1000000-0000-4000-8000-000000000001",
      "name": "eval-1",
      "region": "nyc1",
      "state": "creating",
      "source": {"checkpoint_id": "8f7a9a3f-5555-4444-9999-000000000001"},
      "fork_id": "8f7a9a3f-5555-4444-9999-0000000000f0",
      "forked_from": "b2a2f7a4-8d34-4c1c-9c66-3f2b7f8f38f2",
      "created_at": "2026-07-16T10:00:00Z"
    },
    {
      "id": "c1000000-0000-4000-8000-000000000002",
      "name": "eval-2",
      "region": "nyc1",
      "state": "creating",
      "source": {"checkpoint_id": "8f7a9a3f-5555-4444-9999-000000000001"},
      "fork_id": "8f7a9a3f-5555-4444-9999-0000000000f0",
      "forked_from": "b2a2f7a4-8d34-4c1c-9c66-3f2b7f8f38f2",
      "created_at": "2026-07-16T10:00:00Z"
    }
  ]
}
`

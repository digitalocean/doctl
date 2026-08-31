package integration

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"os/exec"
	"strings"
	"testing"

	"github.com/sclevine/spec"
	"github.com/stretchr/testify/require"
)

var _ = suite("gradient/simulation-run/get", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		cmd    *exec.Cmd
		server *httptest.Server
	)

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/v2/gen-ai/simulation_runs/33333333-3333-4333-8333-333333333333":
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				if req.Method != http.MethodGet {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(simulationRunGetResponse))
			default:
				dump, err := httputil.DumpRequest(req, true)
				if err != nil {
					t.Fatal("failed to dump request")
				}

				t.Fatalf("received unknown request: %s", dump)
			}
		}))
	})

	when("a run uuid is passed", func() {
		it("retrieves the simulation run", func() {
			aliases := []string{"get", "g"}

			for _, alias := range aliases {
				cmd = exec.Command(builtBinaryPath,
					"-t", "some-magic-token",
					"-u", server.URL,
					"gradient",
					"simulation-run",
					alias,
					"33333333-3333-4333-8333-333333333333",
				)

				output, err := cmd.CombinedOutput()
				expect.NoError(err, fmt.Sprintf("received error output: %s", output))
				expect.Equal(strings.TrimSpace(simulationRunGetOutput), strings.TrimSpace(string(output)))
			}
		})
	})

	when("no run uuid is passed", func() {
		it("returns an error", func() {
			cmd = exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"gradient",
				"simulation-run",
				"get",
			)

			output, err := cmd.CombinedOutput()
			expect.Error(err)
			expect.Contains(string(output), "(simulation-run.get) command is missing required arguments")
		})
	})
})

const (
	simulationRunGetOutput = `
UUID                                    Name                  Status                             Scenario Set UUID                       Agent UUID                              Total Journeys    Journeys Finished    Created At                       Success    Failure    Inconclusive
33333333-3333-4333-8333-333333333333    nightly-regression    SIMULATION_RUN_STATUS_SUCCEEDED    00000000-0000-4000-8000-000000000000    99a1cbc7-b1b2-4a0d-9c1f-9b9d2b8f9d1e    6                 6                    2024-06-20 00:00:00 +0000 UTC    4          1          1
`
	simulationRunGetResponse = `
{
  "simulation_run": {
    "run_uuid": "33333333-3333-4333-8333-333333333333",
    "name": "nightly-regression",
    "scenario_set_uuid": "00000000-0000-4000-8000-000000000000",
    "status": "SIMULATION_RUN_STATUS_SUCCEEDED",
    "agent_config": {
      "agent_uuid": "99a1cbc7-b1b2-4a0d-9c1f-9b9d2b8f9d1e"
    },
    "created_at": "2024-06-20T00:00:00Z",
    "total_journeys": 6,
    "journeys_finished": 6,
    "result_summary": {
      "verdict_counts": {
        "success_count": 4,
        "failure_count": 1,
        "inconclusive_count": 1
      },
      "total_duration_sec": "180"
    }
  },
  "scenario_results": [
    {
      "scenario_uuid": "11111111-1111-4111-8111-111111111111",
      "total_journeys": 3,
      "journeys_finished": 3,
      "verdict_counts": {
        "success_count": 2,
        "failure_count": 1
      }
    }
  ]
}
`
)

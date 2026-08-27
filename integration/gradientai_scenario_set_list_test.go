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

var _ = suite("gradient/scenario-set/list", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		cmd    *exec.Cmd
		server *httptest.Server
	)

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/v2/gen-ai/scenario_sets":
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				if req.Method != http.MethodGet {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				if statuses := req.URL.Query()["statuses"]; len(statuses) > 0 {
					expect.Equal([]string{"SCENARIO_SET_STATUS_READY"}, statuses)
				}

				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(scenarioSetsListResponse))
			default:
				dump, err := httputil.DumpRequest(req, true)
				if err != nil {
					t.Fatal("failed to dump request")
				}

				t.Fatalf("received unknown request: %s", dump)
			}
		}))
	})

	when("no flags are passed", func() {
		it("lists all scenario sets", func() {
			aliases := []string{"list", "ls"}

			for _, alias := range aliases {
				cmd = exec.Command(builtBinaryPath,
					"-t", "some-magic-token",
					"-u", server.URL,
					"gradient",
					"scenario-set",
					alias,
				)

				output, err := cmd.CombinedOutput()
				expect.NoError(err, fmt.Sprintf("received error output: %s", output))
				expect.Equal(strings.TrimSpace(scenarioSetsListOutput), strings.TrimSpace(string(output)))
			}
		})
	})

	when("a status shorthand is passed", func() {
		it("expands the shorthand into the API enum", func() {
			cmd = exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"gradient",
				"scenario-set",
				"list",
				"--statuses", "ready",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(scenarioSetsListOutput), strings.TrimSpace(string(output)))
		})
	})
})

const (
	scenarioSetsListOutput = `
UUID                                    Name             Status                            Source Kind                                Scenario Count    Created At                       Updated At
00000000-0000-4000-8000-000000000000    support-flows    SCENARIO_SET_STATUS_READY         SCENARIO_SET_SOURCE_KIND_USER_UPLOAD       12                2024-05-01 00:00:00 +0000 UTC    2024-06-01 00:00:00 +0000 UTC
11111111-1111-4111-8111-111111111111    refund-flows     SCENARIO_SET_STATUS_GENERATING    SCENARIO_SET_SOURCE_KIND_GOAL_GENERATED    5                 2024-06-20 00:00:00 +0000 UTC    2024-06-24 00:00:00 +0000 UTC
`
	scenarioSetsListResponse = `
{
  "scenario_sets": [
    {
      "scenario_set_uuid": "00000000-0000-4000-8000-000000000000",
      "name": "support-flows",
      "status": "SCENARIO_SET_STATUS_READY",
      "source_kind": "SCENARIO_SET_SOURCE_KIND_USER_UPLOAD",
      "scenario_count": 12,
      "created_at": "2024-05-01T00:00:00Z",
      "updated_at": "2024-06-01T00:00:00Z"
    },
    {
      "scenario_set_uuid": "11111111-1111-4111-8111-111111111111",
      "name": "refund-flows",
      "status": "SCENARIO_SET_STATUS_GENERATING",
      "source_kind": "SCENARIO_SET_SOURCE_KIND_GOAL_GENERATED",
      "scenario_count": 5,
      "created_at": "2024-06-20T00:00:00Z",
      "updated_at": "2024-06-24T00:00:00Z"
    }
  ],
  "links": {},
  "meta": {
    "total": 2
  }
}
`
)

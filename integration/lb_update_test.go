package integration

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"os/exec"
	"strings"
	"testing"

	"github.com/sclevine/spec"
	"github.com/stretchr/testify/require"
)

var _ = suite("compute/load-balancer/update", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		server *httptest.Server
	)

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/v2/load_balancers/updated-lb-id":
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				if req.Method != "PUT" {
					w.WriteHeader(http.StatusTeapot)
					return
				}

				reqBody, err := ioutil.ReadAll(req.Body)
				expect.NoError(err)

				expect.JSONEq(lbUpdateRequest, string(reqBody))

				w.Write([]byte(lbUpdateResponse))
			default:
				dump, err := httputil.DumpRequest(req, true)
				if err != nil {
					t.Fatal("failed to dump request")
				}

				t.Fatalf("received unknown request: %s", dump)
			}
		}))
	})

	when("all required flags are passed", func() {
		it("updates the specified load balancer", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"compute",
				"load-balancer",
				"update",
				"updated-lb-id",
				"--droplet-ids", "1,2,3,4",
				"--name", "hello",
				"--region", "the-best-region",
				"--tag-name", "some-tag",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(lbUpdateOutput), strings.TrimSpace(string(output)))
		})
	})
})

// Formatting of responses from server looks very similar
// easier for us to reuse said resonses from get request.
// If / when they materially differ we should feel free
// to make these custom.
const lbUpdateOutput = lbGetOutput
const lbUpdateResponse = lbGetResponse
const lbUpdateRequest = `{
"name":"hello",
"algorithm":"round_robin",
"region":"the-best-region",
"health_check":{},
"sticky_sessions":{},
"droplet_ids":[1,2,3,4],
"tag":"some-tag"
}`

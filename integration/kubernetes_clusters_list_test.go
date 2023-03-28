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

var _ = suite("kubernetes/clusters/list", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		server *httptest.Server
		cmd    *exec.Cmd
	)

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/v2/kubernetes/clusters":
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				if req.Method != http.MethodGet {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				w.Write([]byte(k8sListResponse))
			default:
				dump, err := httputil.DumpRequest(req, true)
				if err != nil {
					t.Fatal("failed to dump request")
				}

				t.Fatalf("received unknown request: %s", dump)
			}
		}))
	})

	when("command is list", func() {
		it("lists the kubernetes clusters", func() {
			cmd = exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"kubernetes",
				"cluster",
				"list",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(k8sListOutput), strings.TrimSpace(string(output)))
		})
	})

	when("command is ls", func() {
		it("lists the kubernetes clusters", func() {
			cmd = exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"kubernetes",
				"cluster",
				"ls",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(k8sListOutput), strings.TrimSpace(string(output)))
		})
	})

	when("command contains the formatted flag", func() {
		it("non-default columns can be displayed", func() {
			cmd = exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"kubernetes",
				"cluster",
				"list",
				"--format",
				"Tags,Created",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(k8sListFormattedOutput), strings.TrimSpace(string(output)))
		})
	})
})

var (
	k8sListResponse = `
{
  "kubernetes_clusters": [
	{
		"id": "some-cluster-id",
		"name": "some-cluster-name",
		"region": "nyc3",
		"version": "some-kube-version",
		"tags": ["production"],
		"auto_upgrade": true,
		"node_pools": [
		{
			"name": "frontend-pool"
		}
		],
		"status": {
		"state": "running",
		"message": "yes"
		},
		"created_at": "2018-11-15T16:00:11Z",
		"updated_at": "2018-11-15T16:00:11Z"
	}
  ]
}
`

	k8sListOutput = `
ID                 Name                 Region    Version              Auto Upgrade    Status     Node Pools
some-cluster-id    some-cluster-name    nyc3      some-kube-version    true            running    frontend-pool
`

	k8sListFormattedOutput = `
Tags          Created At
production    2018-11-15 16:00:11 +0000 UTC
`
)

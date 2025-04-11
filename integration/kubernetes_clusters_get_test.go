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

var _ = suite("kubernetes/clusters/get", func(t *testing.T, when spec.G, it spec.S) {
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

				w.Write([]byte(k8sGetResponse))
			default:
				dump, err := httputil.DumpRequest(req, true)
				if err != nil {
					t.Fatal("failed to dump request")
				}

				t.Fatalf("received unknown request: %s", dump)
			}
		}))
	})

	when("command is get", func() {
		it("gets the kubernetes cluster", func() {
			cmd = exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"kubernetes",
				"cluster",
				"get",
				"some-cluster-id",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(k8sGetOutput), strings.TrimSpace(string(output)))
		})
	})

	when("command is g", func() {
		it("gets the kubernetes cluster", func() {
			cmd = exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"kubernetes",
				"cluster",
				"g",
				"some-cluster-id",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(k8sGetOutput), strings.TrimSpace(string(output)))
		})
	})
})

var (
	k8sGetResponse = `
{
  "kubernetes_clusters": [
	{
		"id": "some-cluster-id",
		"name": "some-cluster-id",
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
		"cluster_autoscaler_configuration": {
    		"scale_down_utilization_threshold": 0.5,
    		"scale_down_unneeded_time": "1m30s"
    	},
		"created_at": "2018-11-15T16:00:11Z",
		"updated_at": "2018-11-15T16:00:11Z"
	}
  ]
}
`

	k8sGetOutput = `ID                 Name               Region    Version              Auto Upgrade    HA Control Plane    Status     Endpoint    IPv4    Cluster Subnet    Service Subnet    Tags          Created At                       Updated At                       Node Pools       Autoscaler Scale Down Utilization    Autoscaler Scale Down Unneeded Time    Routing Agent
some-cluster-id    some-cluster-id    nyc3      some-kube-version    true            false               running                                                            production    2018-11-15 16:00:11 +0000 UTC    2018-11-15 16:00:11 +0000 UTC    frontend-pool    50%                                  1m30s                                  false
`
)

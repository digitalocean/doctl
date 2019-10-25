package integration

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"os/exec"
	"testing"

	"github.com/sclevine/spec"
	"github.com/stretchr/testify/require"
)

var _ = suite("kubernetes/clusters/create", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		server *httptest.Server
	)

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/v2/kubernetes/options":
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				if req.Method != "GET" {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				w.Write([]byte(kubeClustersCreateOptResponse))
			case "/v2/kubernetes/clusters":
				if req.Method != "POST" {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}
				reqBody, err := ioutil.ReadAll(req.Body)
				expect.NoError(err)

				expect.JSONEq(string(reqBody), kubeClustersCreateJSONReq)

				w.Write([]byte(kubeClustersCreateResponse))
			case "/v2/kubernetes/clusters/some-cluster-id":
				if req.Method != "GET" {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				w.Write([]byte(kubeClustersWaitResponse))
			case "/v2/kubernetes/clusters/some-cluster-id/kubeconfig":
				if req.Method != "GET" {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				w.Write([]byte(kubeClustersConfigResponse))
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
		it("creates a kube cluster", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"kubernetes",
				"clusters",
				"create",
				"some-cluster-name",
				"--region", "mars",
				"--version", "some-kube-version",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))

			matchers := []func(string){
				func(s string) { expect.Equal("Notice: cluster is provisioning, waiting for cluster to be running", s) },
				func(s string) { expect.Equal("Notice: cluster created, fetching credentials", s) },
				func(s string) { expect.Regexp(`^Notice: adding cluster credentials to kubeconfig file found in.*`, s) },
				func(s string) { expect.Equal(`Notice: setting current-context to some-context`, s) },
				func(s string) {
					expect.Equal("ID                 Name                 Region    Version              Auto Upgrade    Status     Node Pools", s)
				},
				func(s string) {
					expect.Equal("some-cluster-id    some-cluster-name    mars      some-kube-version    false           running    frontend-pool", s)
				},
			}

			scanner := bufio.NewScanner(bytes.NewBuffer(output))

			var line int
			for scanner.Scan() {
				matcher := matchers[line]
				matcher(scanner.Text())
				line++
			}

			expect.NoError(scanner.Err())
		})
	})

	when("all the node-pool flag is passed", func() {
		it("creates a kube cluster", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"kubernetes",
				"clusters",
				"create",
				"some-cluster-name",
				"--region", "mars",
				"--version", "some-kube-version",
				"--node-pool", "'name=default;auto-scale=true;min-nodes=2;max-nodes=5;count=2'",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))

			matchers := []func(string){
				func(s string) { expect.Equal("Notice: cluster is provisioning, waiting for cluster to be running", s) },
				func(s string) { expect.Equal("Notice: cluster created, fetching credentials", s) },
				func(s string) { expect.Regexp(`^Notice: adding cluster credentials to kubeconfig file found in.*`, s) },
				func(s string) { expect.Equal(`Notice: setting current-context to some-context`, s) },
				func(s string) {
					expect.Equal("ID                 Name                 Region    Version              Auto Upgrade    Status     Node Pools", s)
				},
				func(s string) {
					expect.Equal("some-cluster-id    some-cluster-name    mars      some-kube-version    false           running    frontend-pool", s)
				},
			}

			scanner := bufio.NewScanner(bytes.NewBuffer(output))

			var line int
			for scanner.Scan() {
				matcher := matchers[line]
				matcher(scanner.Text())
				line++
			}

			expect.NoError(scanner.Err())
		})
	})
})

const (
	kubeClustersCreateOptResponse = `
{
"options":{
    "versions": [{"slug":"version-slug","kubernetes_version": "some-kube-version"}],
    "regions": [{"name": "region-name", "slug": "some-region-slug"}],
    "sizes": [{"name":"size-name", "slug": "some-size-slug"}]
  }
}
`
	kubeClustersCreateJSONReq = `
{
  "name": "some-cluster-name",
  "region": "mars",
  "version": "some-kube-version",
  "auto_upgrade": false,
  "maintenance_policy": {
    "day": "any",
    "duration": "",
    "start_time": "00:00"
  },
  "node_pools": [
    {
      "size": "s-1vcpu-2gb",
      "count": 3,
      "name": "some-cluster-name-default-pool"
    }
  ]
}
`
	kubeClustersCreateResponse = `
{
  "kubernetes_cluster": {
    "id": "some-cluster-id"
  }
}
`
	kubeClustersWaitResponse = `
{
  "kubernetes_cluster": {
    "id": "some-cluster-id",
    "name": "some-cluster-name",
    "region": "mars",
    "version": "some-kube-version",
    "tags": ["production"],
    "node_pools": [
      {
        "name": "frontend-pool"
      }
    ],
    "status": {
     "state": "running",
     "message": "yas"
    },
    "created_at": "2018-11-15T16:00:11Z",
    "updated_at": "2018-11-15T16:00:11Z"
  }
}
`
	kubeClustersConfigResponse = `
---
apiVersion: v1
kind: Config
users:
- name: some-user
  user:
    token: some-token
clusters:
- cluster:
    server: https://example.com
  name: some-cluster
contexts:
- context:
    cluster: some-cluster
    user: some-user
  name: some-context
current-context: some-context
`
)

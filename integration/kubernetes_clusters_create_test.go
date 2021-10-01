package integration

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"os"
	"os/exec"
	"strings"
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

				if req.Method != http.MethodGet {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				w.Write([]byte(kubeClustersCreateOptResponse))
			case "/v2/kubernetes/clusters":
				if req.Method != http.MethodPost {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}
				reqBody, err := ioutil.ReadAll(req.Body)
				expect.NoError(err)

				matchedRequest := kubeClustersCreateJSONReq
				if strings.Contains(string(reqBody), "some-node-pool-cluster") {
					matchedRequest = kubeNodePoolCreateJSONReq
				}

				expect.JSONEq(string(reqBody), matchedRequest)

				w.Write([]byte(kubeClustersCreateResponse))
			case "/v2/kubernetes/clusters/some-cluster-id":
				if req.Method != http.MethodGet {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				w.Write([]byte(kubeClustersWaitResponse))
			case "/v2/kubernetes/clusters/some-cluster-id/kubeconfig":
				if req.Method != http.MethodGet {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				w.Write([]byte(kubeClustersConfigResponse))
			case "/v2/1-clicks/kubernetes":
				if req.Method != http.MethodPost {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				w.Write([]byte(oneClickResponse))
			default:
				dump, err := httputil.DumpRequest(req, true)
				if err != nil {
					t.Fatal("failed to dump request")
				}

				t.Fatalf("received unknown request: %s", dump)
			}
		}))
	})

	when("not using node-pool", func() {
		it("creates a kube cluster with defaults", func() {
			f, err := ioutil.TempFile("", "fake-kube-config")
			expect.NoError(err)

			err = f.Close()
			expect.NoError(err)
			defer os.Remove(f.Name())

			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"kubernetes",
				"clusters",
				"create",
				"some-cluster-name",
				"--region", "mars",
				"--version", "some-kube-version",
				"--1-clicks", "slug1",
				"--ha",
			)

			cmd.Env = append(os.Environ(),
				fmt.Sprintf("KUBECONFIG=%s", f.Name()),
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(fmt.Sprintf(kubeClustersCreateOutput, f.Name())), strings.TrimSpace(string(output)))
		})
	})

	when("using node-pool", func() {
		it("creates a kube cluster with the node-pool", func() {
			f, err := ioutil.TempFile("", "fake-kube-config")
			expect.NoError(err)

			err = f.Close()
			expect.NoError(err)
			defer os.Remove(f.Name())

			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"kubernetes",
				"clusters",
				"create",
				"some-node-pool-cluster",
				"--region", "mars",
				"--version", "some-kube-version",
				"--node-pool", "name=default;auto-scale=true;min-nodes=2;max-nodes=5;count=2",
				"--ha",
			)

			cmd.Env = append(os.Environ(),
				fmt.Sprintf("KUBECONFIG=%s", f.Name()),
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
		})

		when("specifying size as well", func() {
			it("returns an error", func() {
				f, err := ioutil.TempFile("", "fake-kube-config")
				expect.NoError(err)

				err = f.Close()
				expect.NoError(err)
				defer os.Remove(f.Name())

				cmd := exec.Command(builtBinaryPath,
					"-t", "some-magic-token",
					"-u", server.URL,
					"kubernetes",
					"clusters",
					"create",
					"some-cluster-name",
					"--region", "mars",
					"--version", "some-kube-version",
					"--size", "the-biggest",
					"--node-pool", "name=default;auto-scale=true;min-nodes=2;max-nodes=5;count=2",
				)

				cmd.Env = append(os.Environ(),
					fmt.Sprintf("KUBECONFIG=%s", f.Name()),
				)

				output, err := cmd.CombinedOutput()
				expect.Error(err)
				expect.Equal(`Error: Flags "size" and "count" cannot be provided when "node-pool" is present`, strings.TrimSpace(string(output)))
			})
		})

		when("installing a one click on an existing k8s cluster", func() {
			it("displays the notice as expected", func() {
				cmd := exec.Command(builtBinaryPath,
					"-t", "some-magic-token",
					"-u", server.URL,
					"kubernetes",
					"1-click",
					"install",
					"12345",
					"--1-clicks",
					"moon",
				)
				output, err := cmd.CombinedOutput()
				expect.NoError(err)
				expect.Equal(`Notice: Some response message`, strings.TrimSpace(string(output)))
			})
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

	kubeClustersCreateOutput = `
Notice: Cluster is provisioning, waiting for cluster to be running
Notice: Cluster created, fetching credentials
Notice: Adding cluster credentials to kubeconfig file found in %q
Notice: Setting current-context to some-context
Notice: Some response message
ID                 Name                 Region    Version              Auto Upgrade    Status     Node Pools
some-cluster-id    some-cluster-name    mars      some-kube-version    false           running    frontend-pool
`
	kubeClustersCreateJSONReq = `
{
  "name": "some-cluster-name",
  "region": "mars",
  "version": "some-kube-version",
  "auto_upgrade": false,
  "surge_upgrade": true,
  "ha": true,
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
	kubeNodePoolCreateJSONReq = `
{
  "name": "some-node-pool-cluster",
  "region": "mars",
  "version": "some-kube-version",
  "auto_upgrade": false,
  "surge_upgrade": true,
  "ha": true,
  "maintenance_policy": {
    "day": "any",
    "duration": "",
    "start_time": "00:00"
  },
  "node_pools": [
    {
      "min_nodes": 2,
      "max_nodes": 5,
      "count": 2,
      "auto_scale": true,
      "name": "default",
      "size": "s-1vcpu-2gb"
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
	oneClickResponse = `
{
	"message": "Some response message"
}
`
)

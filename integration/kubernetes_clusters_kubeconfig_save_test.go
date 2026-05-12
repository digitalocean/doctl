package integration

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"os"
	"os/exec"
	"testing"

	"github.com/sclevine/spec"
	"github.com/stretchr/testify/require"
)

var _ = suite("kubernetes/clusters/kubeconfig/save", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		server *httptest.Server
	)

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/v2/kubernetes/clusters/some-cluster-id/kubeconfig":
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}
				if req.Method != http.MethodGet {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				switch req.URL.Query().Get("type") {
				case "sso":
					w.Write([]byte(kubeClustersSSOConfigResponse))
					return
				case "token", "":
					w.Write([]byte(kubeClustersTokenConfigResponse))
					return
				default:
					w.WriteHeader(http.StatusBadRequest)
					return
				}

			case "/v2/kubernetes/clusters":
				if req.Method != http.MethodGet {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				w.Write([]byte(kubeClustersListResponse))
			default:
				dump, err := httputil.DumpRequest(req, true)
				if err != nil {
					t.Fatal("failed to dump request")
				}

				t.Fatalf("received unknown request: %s", dump)
			}
		}))
	})

	when("passing defaults", func() {
		it("creates a kubeconfig using exec-credentials", func() {
			f, err := os.CreateTemp(t.TempDir(), "fake-kube-config")
			expect.NoError(err)

			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"kubernetes",
				"clusters",
				"kubeconfig",
				"save",
				"some-cluster-name",
			)

			cmd.Env = append(os.Environ(),
				fmt.Sprintf("KUBECONFIG=%s", f.Name()),
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))

			fileBytes, err := io.ReadAll(f)
			expect.NoError(err)
			err = f.Close()
			expect.NoError(err)
			expect.Contains(string(fileBytes), fmt.Sprintf("command: %s", builtBinaryPath))
		})
	})

	when("passing expiry-seconds", func() {
		it("creates a kubeconfig using a token", func() {
			f, err := os.CreateTemp(t.TempDir(), "fake-kube-config")
			expect.NoError(err)

			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"kubernetes",
				"clusters",
				"kubeconfig",
				"save",
				"--expiry-seconds", "60",
				"some-cluster-name",
			)

			cmd.Env = append(os.Environ(),
				fmt.Sprintf("KUBECONFIG=%s", f.Name()),
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))

			fileBytes, err := io.ReadAll(f)
			expect.NoError(err)
			err = f.Close()
			expect.NoError(err)
			expect.Contains(string(fileBytes), "token: some-token")
		})
	})
	when("passing alias", func() {
		it("creates an alias for a config", func() {
			f, err := os.CreateTemp(t.TempDir(), "fake-kube-config")
			expect.NoError(err)

			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"kubernetes",
				"clusters",
				"kubeconfig",
				"save",
				"--alias", "newalias_test",
				"some-cluster-name",
			)

			cmd.Env = append(os.Environ(),
				fmt.Sprintf("KUBECONFIG=%s", f.Name()),
			)

			output, err := cmd.CombinedOutput()

			expect.NoError(err, fmt.Sprintf("received error output: %s", output))

			fileBytes, err := io.ReadAll(f)
			expect.NoError(err)
			err = f.Close()
			expect.NoError(err)
			expect.Contains(string(fileBytes), fmt.Sprintf("current-context: %s", "newalias_test"))
			expect.Contains(string(fileBytes), fmt.Sprintf("name: %s", "newalias_test"))
		})
	})

	when("passing type sso", func() {
		it("merges an exec-based kubeconfig from the API", func() {
			f, err := os.CreateTemp(t.TempDir(), "fake-kube-config")
			expect.NoError(err)

			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"kubernetes",
				"clusters",
				"kubeconfig",
				"save",
				"--type", "sso",
				"some-cluster-name",
			)

			cmd.Env = append(os.Environ(),
				fmt.Sprintf("KUBECONFIG=%s", f.Name()),
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))

			fileBytes, err := io.ReadAll(f)
			expect.NoError(err)
			err = f.Close()
			expect.NoError(err)
			cfg := string(fileBytes)
			expect.Contains(cfg, fmt.Sprintf("command: %s", builtBinaryPath))
			expect.Contains(cfg, "--issuer-url")
			expect.NotContains(cfg, "token: some-token")
		})
	})
})

const kubeClustersSSOConfigResponse = `---
apiVersion: v1
kind: Config
users:
- name: some-user
  user:
    exec:
      apiVersion: client.authentication.k8s.io/v1beta1
      command: doctl
      args:
        - kubernetes
        - cluster
        - kubeconfig
        - exec-credential
        - --version
        - v1beta1
        - --issuer-url
        - https://sso.example.com
        - --client-id
        - some-client-id
clusters:
- cluster:
    server: https://testcuster.com
  name: some-cluster
contexts:
- context:
    cluster: some-cluster
    user: some-user
  name: some-context
current-context: some-context
`

const kubeClustersTokenConfigResponse = `---
apiVersion: v1
kind: Config
users:
- name: some-user
  user:
    token: some-token
clusters:
- cluster:
    server: https://testcluster.com
  name: some-cluster
contexts:
- context:
    cluster: some-cluster
    user: some-user
  name: some-context
current-context: some-context
`

const (
	kubeClustersListResponse = `{
  "kubernetes_clusters": [{
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
  }]
}`
)

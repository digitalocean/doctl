package integration

import (
	"fmt"
	"io/ioutil"
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

				w.Write([]byte(kubeClustersConfigResponse))
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
			f, err := ioutil.TempFile("", "fake-kube-config")
			expect.NoError(err)

			defer os.Remove(f.Name())

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

			fileBytes, err := ioutil.ReadAll(f)
			expect.NoError(err)
			err = f.Close()
			expect.NoError(err)
			expect.Contains(string(fileBytes), fmt.Sprintf("command: %s", builtBinaryPath))
		})
	})

	when("passing expiry-seconds", func() {
		it("creates a kubeconfig using a token", func() {
			f, err := ioutil.TempFile("", "fake-kube-config")
			expect.NoError(err)

			defer os.Remove(f.Name())

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

			fileBytes, err := ioutil.ReadAll(f)
			expect.NoError(err)
			err = f.Close()
			expect.NoError(err)
			expect.Contains(string(fileBytes), "token: some-token")
		})
	})
	when("passing alias", func() {
		it("creates an alias for a config", func() {
			f, err := ioutil.TempFile("", "fake-kube-config")
			expect.NoError(err)

			defer os.Remove(f.Name())

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

			fileBytes, err := ioutil.ReadAll(f)
			expect.NoError(err)
			err = f.Close()
			expect.NoError(err)
			expect.Contains(string(fileBytes), fmt.Sprintf("current-context: %s", "newalias_test"))
			expect.Contains(string(fileBytes), fmt.Sprintf("name: %s", "newalias_test"))
		})
	})
})

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

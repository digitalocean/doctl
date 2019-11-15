package integration

import (
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"os/exec"
	"testing"

	"github.com/sclevine/spec"
	"github.com/stretchr/testify/require"
)

var _ = suite("registry/kubernetes-manifest", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		server *httptest.Server
	)

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			w.Header().Add("content-type", "application/json")

			switch req.URL.Path {
			case "/v2/registry":
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				if req.Method != http.MethodGet {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				w.Write([]byte(registryGetResponse))
			case "/v2/registry/docker-credentials":
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				if req.Method != http.MethodGet {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				// kubernetes needs read-only access
				expect.Contains([]string{"false", "", "0"}, req.URL.Query().Get("read_write"))

				w.Write([]byte(registryDockerCredentialsResponse))
			default:
				dump, err := httputil.DumpRequest(req, true)
				if err != nil {
					t.Fatal("failed to dump request")
				}

				t.Fatalf("received unknown request: %s", dump)
			}
		}))
	})

	it("prints a kubernetes manifest for a secret with the registry credentials", func() {
		cmd := exec.Command(builtBinaryPath,
			"-t", "some-magic-token",
			"-u", server.URL,
			"registry",
			"kubernetes-manifest",
		)

		output, err := cmd.CombinedOutput()
		expect.NoError(err)

		expect.YAMLEq(registryKubernetesManifestOutput, string(output))
	})
})

const (
	registryDockerCredentialsResponse = `{"auths":{"registry.digitalocean.com":{"auth":"YjdkMDNhNjk0N2IyMTdlZmI2ZjNlYzNiZDM1MDQ1ODI6YjdkMDNhNjk0N2IyMTdlZmI2ZjNlYzNiZDM1MDQ1ODIK"}}}`
	registryKubernetesManifestOutput  = `
apiVersion: v1
data:
  .dockerconfigjson: eyJhdXRocyI6eyJyZWdpc3RyeS5kaWdpdGFsb2NlYW4uY29tIjp7ImF1dGgiOiJZamRrTUROaE5qazBOMkl5TVRkbFptSTJaak5sWXpOaVpETTFNRFExT0RJNllqZGtNRE5oTmprME4ySXlNVGRsWm1JMlpqTmxZek5pWkRNMU1EUTFPRElLIn19fQ==
kind: Secret
metadata:
  creationTimestamp: null
  name: registry-my-registry
  namespace: default
type: kubernetes.io/dockerconfigjson
`
)

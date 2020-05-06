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

const (
	registryDockerCredentialsReadOnlyResponse  = "read-only-config"
	registryDockerCredentialsReadWriteResponse = "read-write-config"
)

var _ = suite("registry/docker-config", func(t *testing.T, when spec.G, it spec.S) {
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

				readWriteParam := req.URL.Query().Get("read_write")
				if readWriteParam == "true" || readWriteParam == "1" {
					w.Write([]byte(registryDockerCredentialsReadWriteResponse))
				} else {
					w.Write([]byte(registryDockerCredentialsReadOnlyResponse))
				}

			default:
				dump, err := httputil.DumpRequest(req, true)
				if err != nil {
					t.Fatal("failed to dump request")
				}

				t.Fatalf("received unknown request: %s", dump)
			}
		}))
	})

	it("prints the returned read-only docker config", func() {
		cmd := exec.Command(builtBinaryPath,
			"-t", "some-magic-token",
			"-u", server.URL,
			"registry",
			"docker-config",
		)

		output, err := cmd.CombinedOutput()
		expect.NoError(err)

		expect.Equal(registryDockerCredentialsReadOnlyResponse+"\n", string(output))
	})

	it("prints the returned read-write docker config", func() {
		cmd := exec.Command(builtBinaryPath,
			"-t", "some-magic-token",
			"-u", server.URL,
			"registry",
			"docker-config",
			"--read-write",
		)

		output, err := cmd.CombinedOutput()
		expect.NoError(err)

		expect.Equal(registryDockerCredentialsReadWriteResponse+"\n", string(output))
	})
})

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
				expiryParam := req.URL.Query().Get("expiry_seconds")
				if readWriteParam == "true" || readWriteParam == "1" {
					w.Write([]byte(registryDockerCredentialsReadWriteResponse))
				} else {
					if expiryParam == "3600" {
						w.Write([]byte(registryDockerCredentialsExpiryResponse))
					} else if expiryParam == "" {
						w.Write([]byte(registryDockerCredentialsReadOnlyResponse))
					} else {
						t.Fatalf("received unknown value: %s", expiryParam)
					}
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

	when("all required flags are passed", func() {
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
	})

	when("read-write flag is passed", func() {
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

	when("expiry-seconds flag is passed", func() {
		it("add the correct query parameter", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"registry",
				"docker-config",
				"--expiry-seconds",
				"3600",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err)

			expect.Equal(registryDockerCredentialsExpiryResponse+"\n", string(output))
		})
	})
})

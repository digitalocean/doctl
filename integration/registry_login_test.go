package integration

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/sclevine/spec"
	"github.com/stretchr/testify/require"
)

type dockerConfig struct {
	Auths map[string]struct {
		Auth string `json:"auth,omitempty"`
	} `json:"auths"`
}

var _ = suite("registry/login", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		server *httptest.Server
	)

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			w.Header().Add("content-type", "application/json")

			switch req.URL.Path {
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
				if expiryParam == "3600" || expiryParam == "2592000" {
					w.Write([]byte(registryDockerCredentialsExpiryResponse))
				} else if expiryParam == "" {
					if readWriteParam == "false" {
						w.Write([]byte(registryDockerCredentialsReadOnlyRegistryResponse))
					} else {
						w.Write([]byte(registryDockerCredentialsResponse))
					}
				} else {
					t.Fatalf("received unknown value: %s", expiryParam)
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
		it("writes a docker config.json file", func() {
			tmpDir := t.TempDir()

			config := filepath.Join(tmpDir, "config.json")

			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"registry",
				"login",
			)
			cmd.Env = os.Environ()
			cmd.Env = append(cmd.Env, fmt.Sprintf("DOCKER_CONFIG=%s", tmpDir))

			output, err := cmd.CombinedOutput()
			expect.NoError(err)

			fileBytes, err := os.ReadFile(config)
			expect.NoError(err)

			var dc dockerConfig
			err = json.Unmarshal(fileBytes, &dc)
			expect.NoError(err)

			expect.Equal("Logging Docker in to registry.digitalocean.com\nNotice: Login valid for 30 days. Use the --expiry-seconds flag to set a shorter expiration or --never-expire for no expiration.\n", string(output))
			for host := range dc.Auths {
				expect.Equal("expiring.registry.com", host)
			}
		})
	})

	when("expiry-seconds flag is passed", func() {
		it("add the correct query parameter", func() {
			tmpDir := t.TempDir()

			config := filepath.Join(tmpDir, "config.json")

			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"registry",
				"login",
				"--expiry-seconds",
				"3600",
			)
			cmd.Env = os.Environ()
			cmd.Env = append(cmd.Env, fmt.Sprintf("DOCKER_CONFIG=%s", tmpDir))

			output, err := cmd.CombinedOutput()
			expect.NoError(err)

			fileBytes, err := os.ReadFile(config)
			expect.NoError(err)

			var dc dockerConfig
			err = json.Unmarshal(fileBytes, &dc)
			expect.NoError(err)

			expect.Equal("Logging Docker in to registry.digitalocean.com\n", string(output))
			for host := range dc.Auths {
				expect.Equal("expiring.registry.com", host)
			}
		})
	})

	when("read-only flag is passed and the token doesn't expire", func() {
		it("add the correct query parameter", func() {
			tmpDir := t.TempDir()

			config := filepath.Join(tmpDir, "config.json")

			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"registry",
				"login",
				"--read-only",
				"true",
				"--never-expire",
				"true",
			)
			cmd.Env = os.Environ()
			cmd.Env = append(cmd.Env, fmt.Sprintf("DOCKER_CONFIG=%s", tmpDir))

			output, err := cmd.CombinedOutput()
			expect.NoError(err)

			fileBytes, err := os.ReadFile(config)
			expect.NoError(err)

			var dc dockerConfig
			err = json.Unmarshal(fileBytes, &dc)
			expect.NoError(err)

			expect.Equal("Logging Docker in to registry.digitalocean.com\n", string(output))
			for host := range dc.Auths {
				expect.Equal("readonlyregistry.registry.com", host)
			}
		})
	})
})

const (
	registryDockerCredentialsExpiryResponse           = `{"auths":{"expiring.registry.com":{"auth":"Y3JlZGVudGlhbHM6dGhhdGV4cGlyZQ=="}}}`
	registryDockerCredentialsReadOnlyRegistryResponse = `{"auths":{"readonlyregistry.registry.com":{"auth":"Y3JlZGVudGlhbHM6dGhhdGV4cGlyZQ=="}}}`
)

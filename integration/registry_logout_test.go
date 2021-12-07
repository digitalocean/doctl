package integration

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sclevine/spec"
	"github.com/stretchr/testify/require"
)

var _ = suite("registry/logout", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect      *require.Assertions
		server      *httptest.Server
		oAuthServer *httptest.Server
	)

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			w.Header().Add("content-type", "application/json")

			dump, err := httputil.DumpRequest(req, true)
			if err != nil {
				t.Fatal("failed to dump request")
			}

			t.Fatalf("received unknown request: %s", dump)

		}))

		oAuthServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			if http.MethodPost != req.Method {
				t.Fatalf("method = %v, expected %v", req.Method, http.MethodPost)
			}

			authHeader := req.Header.Get("Authorization")
			token := strings.TrimPrefix(strings.ToLower(authHeader), "bearer ")
			if token == "" {
				t.Fatalf("no token in auth header")
			}

			req.ParseForm()
			bodyToken := req.Form.Get("token")
			if token != bodyToken {
				t.Fatalf("expected tokens to match: body  = %v, header %v", bodyToken, token)
			}

			w.WriteHeader(http.StatusOK)

		}))
	})

	it("removes the registry from the docker config.json file", func() {
		tmpDir, err := ioutil.TempDir("", "")
		expect.NoError(err)

		config := filepath.Join(tmpDir, "config.json")
		err = ioutil.WriteFile(config, []byte(registryDockerCredentialsResponse), 0600)
		expect.NoError(err)

		cmd := exec.Command(builtBinaryPath,
			"-t", "some-magic-token",
			"-u", server.URL,
			"registry",
			"logout",
			"--authorization-server-endpoint", oAuthServer.URL,
		)
		cmd.Env = os.Environ()
		cmd.Env = append(cmd.Env, fmt.Sprintf("DOCKER_CONFIG=%s", tmpDir))

		output, err := cmd.CombinedOutput()
		expect.NoError(err, string(output))

		fileBytes, err := ioutil.ReadFile(config)
		expect.NoError(err)

		expect.Equal("Removing login credentials for registry.digitalocean.com\n", string(output))
		expect.Equal(false, strings.Contains(string(fileBytes), "registry.digitalocean.com"))
	})
})

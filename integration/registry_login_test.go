package integration

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
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

	it("writes a docker config.json file", func() {
		tmpDir, err := ioutil.TempDir("", "")
		expect.NoError(err)

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

		fileBytes, err := ioutil.ReadFile(config)
		expect.NoError(err)

		var dc dockerConfig
		err = json.Unmarshal(fileBytes, &dc)
		expect.NoError(err)

		expect.Equal("Logging Docker in to registry.digitalocean.com\n", string(output))
		for host, _ := range dc.Auths {
			expect.Equal("registry.digitalocean.com", host)
		}
	})
})

package integration

import (
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"os/exec"
	"strings"
	"testing"

	"github.com/sclevine/spec"
	"github.com/stretchr/testify/require"
)

var _ = suite("registry/repository/delete-manifest", func(t *testing.T, when spec.G, it spec.S) {
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
			case "/v2/registry/my-registry/repositories/my-repo/digests/sha256:5b0bcabd1ed22e9fb1310cf6c2dec7cdef19f0ad69efa1f392e94a4333501270":
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				if req.Method != http.MethodDelete {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				w.WriteHeader(http.StatusNoContent)
			default:
				dump, err := httputil.DumpRequest(req, true)
				if err != nil {
					t.Fatal("failed to dump request")
				}

				t.Fatalf("received unknown request: %s", dump)
			}
		}))
	})

	it("deletes repository manifest", func() {
		cmd := exec.Command(builtBinaryPath,
			"-t", "some-magic-token",
			"-u", server.URL,
			"registry",
			"repository",
			"delete-manifest",
			"my-repo",
			"sha256:5b0bcabd1ed22e9fb1310cf6c2dec7cdef19f0ad69efa1f392e94a4333501270",
			"--force",
		)

		output, err := cmd.CombinedOutput()
		expect.NoError(err)

		expect.Equal("", strings.TrimSpace(string(output)))
	})
})

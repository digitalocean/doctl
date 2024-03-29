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

var _ = suite("registry/repository/list-v2", func(t *testing.T, when spec.G, it spec.S) {
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
			case "/v2/registry/my-registry/repositoriesV2":
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				if req.Method != http.MethodGet {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				w.Write([]byte(repositoryListV2Response))
			default:
				dump, err := httputil.DumpRequest(req, true)
				if err != nil {
					t.Fatal("failed to dump request")
				}

				t.Fatalf("received unknown request: %s", dump)
			}
		}))
	})

	it("returns list of repositories in registry", func() {
		cmd := exec.Command(builtBinaryPath,
			"-t", "some-magic-token",
			"-u", server.URL,
			"registry",
			"repository",
			"list-v2",
		)

		output, err := cmd.CombinedOutput()
		expect.NoError(err, "Output: %s", output)

		expect.Equal(strings.TrimSpace(repositoryListV2Output), strings.TrimSpace(string(output)))
	})
})

var (
	repositoryListV2Output = `
Name      Latest Manifest                                                            Latest Tag    Tag Count    Manifest Count    Updated At
repo-1    sha256:cb8a924afdf0229ef7515d9e5b3024e23b3eb03ddbba287f4a19c6ac90b8d221    v1            57           82                2021-04-09 23:54:25 +0000 UTC
repo-2                                                                               <none>        57           82                <nil>
`

	repositoryListV2Response = `{
  "repositories": [
    {
      "registry_name": "example",
      "name": "repo-1",
      "tag_count": 57,
      "manifest_count": 82,
      "latest_manifest": {
        "digest": "sha256:cb8a924afdf0229ef7515d9e5b3024e23b3eb03ddbba287f4a19c6ac90b8d221",
        "registry_name": "example",
        "repository": "repo-1",
        "compressed_size_bytes": 1972332,
        "size_bytes": 2816445,
        "updated_at": "2021-04-09T23:54:25Z",
        "tags": [
          "v1",
          "v2"
        ],
        "blobs": [
          {
            "digest": "sha256:14119a10abf4669e8cdbdff324a9f9605d99697215a0d21c360fe8dfa8471bab",
            "compressed_size_bytes": 1471
          },
          {
            "digest": "sha256:a0d0a0d46f8b52473982a3c466318f479767577551a53ffc9074c9fa7035982e",
            "compressed_size_byte": 2814446
          },
          {
            "digest": "sha256:69704ef328d05a9f806b6b8502915e6a0a4faa4d72018dc42343f511490daf8a",
            "compressed_size_bytes": 528
          }
        ]
      }
    },
    {
      "registry_name": "example-no-latest-manifest",
      "name": "repo-2",
      "tag_count": 57,
      "manifest_count": 82
    }
  ],
  "meta": {
    "total": 2
  }
}`
)

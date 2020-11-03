package integration

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"os/exec"
	"strings"
	"testing"

	"github.com/sclevine/spec"
	"github.com/stretchr/testify/require"
)

var _ = suite("registry/create", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect           *require.Assertions
		server           *httptest.Server
		expectedTierSlug string
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

				if req.Method != http.MethodPost {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				reqBody, err := ioutil.ReadAll(req.Body)
				expect.NoError(err)

				expectedJSON := fmt.Sprintf(registryCreateRequest, expectedTierSlug)
				expect.JSONEq(expectedJSON, string(reqBody))
				w.WriteHeader(http.StatusCreated)
				w.Write([]byte(registryCreateResponse))
			default:
				dump, err := httputil.DumpRequest(req, true)
				if err != nil {
					t.Fatal("failed to dump request")
				}

				t.Fatalf("received unknown request: %s", dump)
			}
		}))
	})

	it("creates a registry", func() {
		cmd := exec.Command(builtBinaryPath,
			"-t", "some-magic-token",
			"-u", server.URL,
			"registry",
			"create",
			"my-registry",
		)
		expectedTierSlug = "basic"

		output, err := cmd.CombinedOutput()
		expect.NoError(err)

		expect.Equal(strings.TrimSpace(registryGetOutput), strings.TrimSpace(string(output)))
	})

	it("creates a registry with subscription tier specified", func() {
		cmd := exec.Command(builtBinaryPath,
			"-t", "some-magic-token",
			"-u", server.URL,
			"registry",
			"create",
			"my-registry",
			"--subscription-tier", "starter",
		)
		expectedTierSlug = "starter"

		output, err := cmd.CombinedOutput()
		expect.NoError(err)

		expect.Equal(strings.TrimSpace(registryGetOutput), strings.TrimSpace(string(output)))
	})
})

const (
	registryCreateRequest = `
{
	"name": "my-registry",
	"subscription_tier_slug": "%s"
}
`
	registryCreateRequestWithTier = `
{
	"name": "my-registry",
	"subscription_tier_slug": "basic"
}
`
	registryCreateResponse = `
{
	"registry": {
		"name": "my-registry"
	}
}`
	registryCreateOutput = `
Name           Endpoint
my-registry    registry.digitalocean.com/my-registry
`
)

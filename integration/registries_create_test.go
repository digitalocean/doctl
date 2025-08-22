package integration

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"os/exec"
	"strings"
	"testing"

	"github.com/sclevine/spec"
	"github.com/stretchr/testify/require"
)

var _ = suite("registries/create", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect                 *require.Assertions
		server                 *httptest.Server
		reqRegion              string // region provided in http create req
		expectedRegion         string // region in response
		expectSubscriptionTier bool   // whether to expect subscription tier in request
	)

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			w.Header().Add("content-type", "application/json")

			switch req.URL.Path {
			case "/v2/registries":
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				if req.Method != http.MethodPost {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				reqBody, err := io.ReadAll(req.Body)
				expect.NoError(err)

				var expectedJSON string
				if expectSubscriptionTier {
					expectedJSON = registriesCreateRequestWithSubscriptionTier
				} else if reqRegion == "" {
					expectedJSON = registriesCreateRequest
				} else {
					expectedJSON = fmt.Sprintf(registriesCreateRequestWithRegion, reqRegion)
				}
				expect.JSONEq(expectedJSON, string(reqBody))
				w.WriteHeader(http.StatusCreated)
				w.Write([]byte(fmt.Sprintf(registriesCreateResponse, expectedRegion)))
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
			"registries",
			"create",
			"my-registry",
		)
		reqRegion = ""
		expectedRegion = "default"
		expectSubscriptionTier = false

		output, err := cmd.CombinedOutput()
		expect.NoError(err)

		expect.Equal(strings.TrimSpace(fmt.Sprintf(registryGetOutput, expectedRegion)), strings.TrimSpace(string(output)))
	})

	it("creates a registry with subscription tier specified", func() {
		cmd := exec.Command(builtBinaryPath,
			"-t", "some-magic-token",
			"-u", server.URL,
			"registries",
			"create",
			"my-registry",
			"--subscription-tier", "starter",
		)
		reqRegion = ""
		expectedRegion = "default"
		expectSubscriptionTier = true

		output, err := cmd.CombinedOutput()
		expect.NoError(err)

		expect.Equal(strings.TrimSpace(fmt.Sprintf(registryGetOutput, expectedRegion)), strings.TrimSpace(string(output)))
	})

	it("creates a registry with region specified", func() {
		cmd := exec.Command(builtBinaryPath,
			"-t", "some-magic-token",
			"-u", server.URL,
			"registries",
			"create",
			"my-registry",
			"--region", "r1",
		)
		reqRegion = "r1"
		expectedRegion = "r1"
		expectSubscriptionTier = false

		output, err := cmd.CombinedOutput()
		expect.NoError(err)

		expect.Equal(strings.TrimSpace(fmt.Sprintf(registryGetOutput, expectedRegion)), strings.TrimSpace(string(output)))
	})
})

const (
	registriesCreateRequest = `
{
	"name": "my-registry",
	"subscription_tier_slug": "basic"
}
`
	registriesCreateRequestWithRegion = `
{
	"name": "my-registry",
	"region": "%s",
	"subscription_tier_slug": "basic"
}
`
	registriesCreateRequestWithSubscriptionTier = `
{
	"name": "my-registry",
	"subscription_tier_slug": "starter"
}
`
	registriesCreateResponse = `
{
	"registry": {
		"name": "my-registry",
		"region": "%s"
	}
}`
)

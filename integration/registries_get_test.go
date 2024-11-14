package integration

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"os/exec"
	"regexp"
	"strings"
	"testing"

	"github.com/sclevine/spec"
	"github.com/stretchr/testify/require"
)

var _ = suite("registries/get", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect         *require.Assertions
		server         *httptest.Server
		expectedRegion string
	)

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			w.Header().Add("content-type", "application/json")

			// Use regex to match any registry name in the URL
			registryPathRegex := regexp.MustCompile(`^/v2/registries/([^/]+)$`)
			matches := registryPathRegex.FindStringSubmatch(req.URL.Path)

			if len(matches) == 2 {
				registryName := matches[1]

				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				if req.Method != http.MethodGet {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				w.Write([]byte(fmt.Sprintf(registriesGetResponse, registryName)))
			} else {
				// Handle unknown requests
				dump, err := httputil.DumpRequest(req, true)
				if err != nil {
					t.Fatal("failed to dump request")
				}
				t.Fatalf("received unknown request: %s", dump)
			}
		}))
	})

	it("returns registry named my-registry", func() {
		cmd := exec.Command(builtBinaryPath,
			"-t", "some-magic-token",
			"-u", server.URL,
			"registries",
			"get",
			"my-registry",
		)
		expectedRegion = "r1"

		output, err := cmd.CombinedOutput()
		expect.NoError(err)

		expect.Equal(strings.TrimSpace(fmt.Sprintf(registryGetOutput, expectedRegion)), strings.TrimSpace(string(output)))
	})

	it("returns multiple registries", func() {
		regs := []string{"reg1", "reg2", "reg3"}
		args := []string{
			"-t", "some-magic-token",
			"-u", server.URL,
			"registries",
			"get",
		}
		args = append(args, regs...)
		cmd := exec.Command(builtBinaryPath, args...)
		expectedRegion := "r1"

		output, err := cmd.CombinedOutput()
		expect.NoError(err)

		expect.Equal(strings.TrimSpace(fmt.Sprintf(registriesGetOutput, expectedRegion, expectedRegion, expectedRegion)), strings.TrimSpace(string(output)))
	})
})

const (
	registriesGetResponse = `
{
	"registry": {
		"name": "%s",
		"region": "r1"
	}
}`
	registriesGetOutput = `
Name    Endpoint                          Region slug
reg1    registry.digitalocean.com/reg1    %s
reg2    registry.digitalocean.com/reg2    %s
reg3    registry.digitalocean.com/reg3    %s
`
)

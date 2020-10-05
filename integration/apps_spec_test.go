package integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"os/exec"
	"strings"
	"testing"

	"github.com/digitalocean/godo"
	"github.com/mitchellh/copystructure"
	"github.com/sclevine/spec"
	"github.com/stretchr/testify/require"
)

var _ = suite("apps/spec/get", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		server *httptest.Server
	)

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			w.Header().Add("content-type", "application/json")

			switch req.URL.Path {
			case "/v2/apps/" + testAppUUID:
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				if req.Method != http.MethodGet {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				json.NewEncoder(w).Encode(testAppResponse)
			case "/v2/apps/" + testAppUUID + "/deployments/" + testDeploymentUUID:
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				if req.Method != http.MethodGet {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				res := struct {
					Deployment *godo.Deployment `json:"deployment"`
				}{
					Deployment: copystructure.Must(copystructure.Copy(testDeployment)).(*godo.Deployment),
				}
				res.Deployment.Spec.Services[0].GitHub.Branch = "new-branch"

				json.NewEncoder(w).Encode(res)
			default:
				dump, err := httputil.DumpRequest(req, true)
				if err != nil {
					t.Fatal("failed to dump request")
				}

				t.Fatalf("received unknown request: %s", dump)
			}
		}))
	})

	it("gets an app's spec", func() {
		cmd := exec.Command(builtBinaryPath,
			"-t", "some-magic-token",
			"-u", server.URL,
			"apps", "spec", "get",
			testAppUUID,
		)

		output, err := cmd.CombinedOutput()
		expect.NoError(err)

		expectedOutput := `name: test
services:
- github:
    branch: master
    repo: digitalocean/doctl
  name: service`
		expect.Equal(expectedOutput, strings.TrimSpace(string(output)))
	})

	it("gets a deployment's spec", func() {
		cmd := exec.Command(builtBinaryPath,
			"-t", "some-magic-token",
			"-u", server.URL,
			"apps", "spec", "get",
			"--deployment", testDeploymentUUID,
			testAppUUID,
		)

		output, err := cmd.CombinedOutput()
		expect.NoError(err)

		expectedOutput := `name: test
services:
- github:
    branch: new-branch
    repo: digitalocean/doctl
  name: service`
		expect.Equal(expectedOutput, strings.TrimSpace(string(output)))
	})
})

var _ = suite("apps/spec/validate", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		server *httptest.Server
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
	})

	it("accepts a valid spec", func() {
		cmd := exec.Command(builtBinaryPath,
			"-t", "some-magic-token",
			"-u", server.URL,
			"apps", "spec", "validate", "-",
		)
		byt, err := json.Marshal(testAppSpec)
		expect.NoError(err)

		cmd.Stdin = bytes.NewReader(byt)

		output, err := cmd.CombinedOutput()
		expect.NoError(err)

		expectedOutput := "The spec is valid."
		expect.Equal(expectedOutput, strings.TrimSpace(string(output)))
	})

	it("fails on invalid specs", func() {
		cmd := exec.Command(builtBinaryPath,
			"-t", "some-magic-token",
			"-u", server.URL,
			"apps", "spec", "validate", "-",
		)
		testSpec := `name: test
services:
  name: service
  github:
    repo: digitalocean/doctl
`
		cmd.Stdin = strings.NewReader(testSpec)

		output, err := cmd.CombinedOutput()
		expect.Equal("exit status 1", err.Error())

		expectedOutput := "Error: Failed to parse app spec: json: cannot unmarshal object into Go struct field AppSpec.services of type []*godo.AppServiceSpec"
		expect.Equal(expectedOutput, strings.TrimSpace(string(output)))
	})
})

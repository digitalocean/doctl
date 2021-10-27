package integration

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"os/exec"
	"strings"
	"testing"

	"github.com/sclevine/spec"
	"github.com/stretchr/testify/require"
)

var _ = suite("compute/firewall/delete", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		server *httptest.Server
	)

	it.Before(func() {
		expect = require.New(t)
		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/v2/firewalls/e4b9c960-d385-4950-84f3-d102162e6be5":
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
			case "/v2/firewalls/aaa-bbb-ccc-ddd-eee":
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

	when("all the required flags are passed", func() {
		it("deletes the firewall", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"compute",
				"firewall",
				"delete",
				"e4b9c960-d385-4950-84f3-d102162e6be5",
				"--force",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received unexpected error: %s", output))
			expect.Empty(output)
		})
	})

	when("multiple firewalls are provided and all the required flags are passed", func() {
		it("deletes the firewalls", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"compute",
				"firewall",
				"delete",
				"e4b9c960-d385-4950-84f3-d102162e6be5",
				"aaa-bbb-ccc-ddd-eee",
				"--force",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received unexpected error: %s", output))
			expect.Empty(output)
		})
	})

	when("deleting one firewall without force flag", func() {
		it("correctly prompts for confirmation", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"compute",
				"firewall",
				"delete",
				"e4b9c960-d385-4950-84f3-d102162e6be5",
			)

			output, err := cmd.CombinedOutput()
			expect.Error(err)
			expect.Equal(strings.TrimSpace(fwDelOutput), strings.TrimSpace(string(output)))
		})
	})

	when("deleting two firewalls without force flag", func() {
		it("correctly prompts for confirmation", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"compute",
				"firewall",
				"delete",
				"e4b9c960-d385-4950-84f3-d102162e6be5",
				"aaa-bbb-ccc-ddd-eee",
			)

			output, err := cmd.CombinedOutput()
			expect.Error(err)
			expect.Equal(strings.TrimSpace(multiFwDelOutput), strings.TrimSpace(string(output)))
		})
	})
})

const (
	fwDelOutput      = "Warning: Are you sure you want to delete this firewall? (y/N) ? Error: Operation aborted."
	multiFwDelOutput = "Warning: Are you sure you want to delete 2 firewalls? (y/N) ? Error: Operation aborted."
)

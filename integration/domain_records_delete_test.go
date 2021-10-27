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

var _ = suite("compute/domain/records/delete", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		server *httptest.Server
	)

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/v2/domains/example.com/records/1337":
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				if req.Method != http.MethodDelete {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				reqBody, err := ioutil.ReadAll(req.Body)
				expect.NoError(err)
				expect.Empty(string(reqBody))

				w.WriteHeader(http.StatusNoContent)

			case "/v2/domains/example.com/records/7331":
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				if req.Method != http.MethodDelete {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				reqBody, err := ioutil.ReadAll(req.Body)
				expect.NoError(err)
				expect.Empty(string(reqBody))

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

	when("all required flags are passed", func() {
		it("deletes the right domain record", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"compute",
				"domain",
				"records",
				"delete",
				"example.com",
				"1337",
				"--force",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Empty(output)
		})
	})

	when("multiple records are provided and all required flags are passed", func() {
		it("deletes the right domain records", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"compute",
				"domain",
				"records",
				"delete",
				"example.com",
				"1337",
				"7331",
				"--force",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Empty(output)
		})
	})

	when("deleting one domain record without force flag", func() {
		it("correctly prompts for confirmation", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"compute",
				"domain",
				"records",
				"delete",
				"example.com",
				"1337",
			)

			output, err := cmd.CombinedOutput()
			expect.Error(err)
			expect.Equal(strings.TrimSpace(domainRecDelOutput), strings.TrimSpace(string(output)))
		})
	})

	when("deleting two domain records without force flag", func() {
		it("correctly prompts for confirmation", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"compute",
				"domain",
				"records",
				"delete",
				"example.com",
				"1337",
				"7331",
			)

			output, err := cmd.CombinedOutput()
			expect.Error(err)
			expect.Equal(strings.TrimSpace(multiDomainRecDelOutput), strings.TrimSpace(string(output)))
		})
	})
})

const (
	domainRecDelOutput      = "Warning: Are you sure you want to delete this domain record? (y/N) ? Error: Operation aborted."
	multiDomainRecDelOutput = "Warning: Are you sure you want to delete 2 domain records? (y/N) ? Error: Operation aborted."
)

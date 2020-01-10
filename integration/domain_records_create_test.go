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

var _ = suite("compute/domain/records/create", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		server *httptest.Server
	)

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/v2/domains/example.com/records":
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
				expect.JSONEq(domainRecordsCreateRequest, string(reqBody))

				w.Write([]byte(domainRecordsCreateResponse))

			case "/v2/domains/test.com/records":
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
				expect.JSONEq(domainRecordsCreateWithoutPortRequest, string(reqBody))

				w.Write([]byte(domainRecordsCreateWithoutPortResponse))

			default:
				dump, err := httputil.DumpRequest(req, true)
				if err != nil {
					t.Fatal("failed to dump request")
				}

				t.Fatalf("received unknown request: %s", dump)
			}
		}))
	})

	when("command is create", func() {
		it("creates a domain record", func() {
			aliases := []string{"create", "c"}

			for _, alias := range aliases {
				cmd := exec.Command(builtBinaryPath,
					"-t", "some-magic-token",
					"-u", server.URL,
					"compute",
					"domain",
					"records",
					alias,
					"example.com",
					"--record-name", "example.com",
					"--record-type", "SRV",
					"--record-port", "0",
				)

				output, err := cmd.CombinedOutput()
				expect.NoError(err, fmt.Sprintf("received error output: %s", output))
				expect.Equal(strings.TrimSpace(domainRecordsCreateOutput), strings.TrimSpace(string(output)))
			}
		})
	})

	when("command is create without a port", func() {
		it("creates a domain record without sending the port", func() {
			aliases := []string{"create", "c"}

			for _, alias := range aliases {
				cmd := exec.Command(builtBinaryPath,
					"-t", "some-magic-token",
					"-u", server.URL,
					"compute",
					"domain",
					"records",
					alias,
					"test.com",
					"--record-type", "A",
					"--record-name", "www",
					"--record-data", "1.1.1.1",
				)

				output, err := cmd.CombinedOutput()
				expect.NoError(err, fmt.Sprintf("received error output: %s", output))
				expect.Equal(strings.TrimSpace(domainRecordsCreateWithoutPortOutput), strings.TrimSpace(string(output)))
			}
		})
	})
})

const (
	domainRecordsCreateOutput = `
ID    Type    Name           Data    Priority    Port    TTL    Weight
0             example.com            0           0       0      0
`
	domainRecordsCreateResponse = `
{
  "domain_record": {
	"flags": 0,
    "name": "example.com",
    "port": 0
  }
}
`
	domainRecordsCreateRequest = `
{"flags":0, "name":"example.com", "port":0, "priority":0, "ttl":1800, "type":"SRV", "weight":0}
`

	domainRecordsCreateWithoutPortOutput = `
ID    Type    Name        Data    Priority    Port    TTL    Weight
0             test.com            0           0       0      0
`
	domainRecordsCreateWithoutPortResponse = `
{
  "domain_record": {
	"flags": 0,
    "name": "test.com",
    "port": 0
  }
}
`
	domainRecordsCreateWithoutPortRequest = `
{"data":"1.1.1.1", "flags":0, "name":"www", "priority":0, "ttl":1800, "type":"A", "weight":0}
`
)

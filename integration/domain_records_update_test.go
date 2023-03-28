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

var _ = suite("compute/domain/records/update", func(t *testing.T, when spec.G, it spec.S) {
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

				if req.Method != http.MethodPatch {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				reqBody, err := ioutil.ReadAll(req.Body)
				expect.NoError(err)

				expect.JSONEq(domainRecordUpdateNameRequest, string(reqBody))

				w.Write([]byte(domainRecordUpdateNameResponse))

			default:
				dump, err := httputil.DumpRequest(req, true)
				if err != nil {
					t.Fatal("failed to dump request")
				}

				t.Fatalf("received unknown request: %s", dump)
			}
		}))
	})

	when("command is update no ttl", func() {
		it("update a domain record", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"compute",
				"domain",
				"records",
				"update",
				"example.com",
				"--record-id",
				"1337",
				"--record-name",
				"foo",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(domainRecordUpdateNameOutput), strings.TrimSpace(string(output)))

			expected := strings.TrimSpace(domainRecordUpdateNameOutput)
			actual := strings.TrimSpace(string(output))

			if expected != actual {
				t.Errorf("expected\n\n%s\n\nbut got\n\n%s\n\n", expected, actual)
			}
		})
	})

})

const (
	domainRecordUpdateNameRequest = `
	{
		"name": "foo",
		"flags": 0,
		"priority": 0,
		"weight": 0
	}`
	domainRecordUpdateNameResponse = `
{
	"domain_record": {
		"id": 3352896,
		"type": "A",
		"name": "blog",
		"data": "162.10.66.0",
		"priority": null,
		"port": null,
		"ttl": 650,
		"weight": null,
		"flags": null,
		"tag": null
	}
}`

	domainRecordUpdateNameOutput = `
ID         Type    Name    Data           Priority    Port    TTL    Weight
3352896    A       blog    162.10.66.0    0           0       650    0`
)

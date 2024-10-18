package integration

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"os/exec"
	"testing"

	"github.com/sclevine/spec"
	"github.com/stretchr/testify/require"
)

var _ = suite("database/replica", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		server *httptest.Server
	)

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/v2/databases/1111/replicas":
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				if req.Method != http.MethodPost {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				body, err := io.ReadAll(req.Body)
				expect.NoError(err)

				expect.JSONEq(databaseReplicaCreateRequest, string(body))

				w.WriteHeader(http.StatusCreated)
				w.Write([]byte(databaseReplicaCreateResponse))
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
		it("create a read-only replica database", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"databases",
				"replica",
				"create",
				"1111",
				"2222",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
		})

	})
})

const (
	// All of the values are default except for the name.
	databaseReplicaCreateRequest  = `{"name":"2222","private_network_uuid":"","region":"nyc1","size":"db-s-1vcpu-1gb"}`
	databaseReplicaCreateResponse = `{"replica":{"name":"2222","connection":{"uri":"","database":"defaultdb","host":"","port":25060,"user":"doadmin","password":"","ssl":true},"private_connection":{"uri":"","database":"","host":"","port":25060,"user":"doadmin","password":"","ssl":true},"region":"nyc1","status":"online","created_at":"2019-01-11T18:37:36Z"}}`
)

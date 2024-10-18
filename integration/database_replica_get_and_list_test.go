package integration

import (
	"fmt"
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
			auth := req.Header.Get("Authorization")
			if auth != "Bearer some-magic-token" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			if req.Method != http.MethodGet {
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}
			switch req.URL.Path {
			case "/v2/databases/1111/replicas/2222":
				w.Write([]byte(`{"replica":`))
				w.Write([]byte(replicaMetadata))
				w.Write([]byte(`}`))
			case "/v2/databases/1111/replicas":
				w.Write([]byte(`{"replicas":[`))
				w.Write([]byte(replicaMetadata))
				w.Write([]byte(`]}`))
			default:
				dump, err := httputil.DumpRequest(req, true)
				if err != nil {
					t.Fatal("failed to dump request")
				}

				t.Fatalf("received unknown request: %s", dump)
			}
		}))
	})

	when("command is get", func() {
		it("return metadata about a replica database", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"databases",
				"replica",
				"get",
				"1111",
				"2222",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
		})

	})

	when("command is list", func() {
		it("list all of the replica databases", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"databases",
				"replica",
				"list",
				"1111",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
		})

	})
})

const replicaMetadata = `{"name":"read-nyc3-01","connection":{"uri":"","database":"defaultdb","host":"read-nyc3-01-do-user-19081923-0.db.ondigitalocean.com","port":25060,"user":"doadmin","password":"","ssl":true},"private_connection":{"uri":"","database":"","host":"private-read-nyc3-01-do-user-19081923-0.db.ondigitalocean.com","port":25060,"user":"doadmin","password":"","ssl":true},"region":"nyc3","status":"online","created_at":"2019-01-11T18:37:36Z"}`

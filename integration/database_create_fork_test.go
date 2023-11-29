package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
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

var _ = suite("database/create/fork", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		server *httptest.Server
	)

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/v2/databases":
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

				request := struct {
					Name          string `json:"name"`
					Engine        string `json:"engine"`
					Version       string `json:"version"`
					Region        string `json:"region"`
					Nodes         int    `json:"num_nodes"`
					BackupRestore any    `json:"backup_restore"`
				}{}

				err = json.Unmarshal(reqBody, &request)
				expect.NoError(err)

				t, err := template.New("response").Parse(databaseRestoreBackUpCreateResponse)
				expect.NoError(err)

				var b []byte
				buffer := bytes.NewBuffer(b)
				err = t.Execute(buffer, request)
				expect.NoError(err)

				w.Write(buffer.Bytes())
			case "/v2/databases/some-id":
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				if req.Method != http.MethodGet {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				w.Write([]byte(databaseRestoreBackUpCreateResponse))
			default:
				dump, err := httputil.DumpRequest(req, true)
				if err != nil {
					t.Fatal("failed to dump request")
				}

				t.Fatalf("received unknown request: %s", dump)
			}
		}))
	})

	when("the minimum required flags are provided", func() {
		it("creates a backup database", func() {
			aliases := []string{"create", "c"}

			for _, alias := range aliases {
				cmd := exec.Command(builtBinaryPath,
					"-t", "some-magic-token",
					"-u", server.URL,
					"databases",
					"fork",
					"new-db-name",
					"--restore-from-cluster-id", "some-id",
					"--restore-from-timestamp", "2023-02-01 17:32:15 +0000 UTC",
				)

				output, err := cmd.CombinedOutput()
				expect.NoError(err, fmt.Sprintf("received error output from command %s: %s", alias, output))
				expect.Equal(strings.TrimSpace(databasesCreateRestoreBackUpOutput), strings.TrimSpace(string(output)))
			}
		})
	})

	when("the wrong time format is passed", func() {
		it("errors out with wrong time format", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"databases",
				"fork",
				"new-db-name",
				"--restore-from-cluster-id", "some-id",
				"--restore-from-timestamp", "2009-11-10T23:00:00Z",
			)

			output, err := cmd.CombinedOutput()
			expect.Error(err)
			expect.Equal(strings.TrimSpace(restoreFromTimestampError), strings.TrimSpace(string(output)))
		})
	})
})

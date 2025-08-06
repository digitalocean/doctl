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

var _ = suite("database/config/get", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		server *httptest.Server
	)

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/v2/databases/mysql-database-id/config":
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusTeapot)
				}

				if req.Method != http.MethodPatch {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				expected := `{"config":{"sql_mode":"ANSI"}}`
				b, err := io.ReadAll(req.Body)
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				expect.Equal(expected, strings.TrimSpace(string(b)))

				w.WriteHeader(http.StatusOK)
			case "/v2/databases/pg-database-id/config":
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusTeapot)
				}

				if req.Method != http.MethodPatch {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				expected := `{"config":{"pgbouncer":{"server_reset_query_always":false}}}`
				b, err := io.ReadAll(req.Body)
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				expect.Equal(expected, strings.TrimSpace(string(b)))

				w.WriteHeader(http.StatusOK)
			case "/v2/databases/redis-database-id/config":
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusTeapot)
				}

				if req.Method != http.MethodPatch {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				expected := `{"config":{"redis_timeout":1200}}`
				b, err := io.ReadAll(req.Body)
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				expect.Equal(expected, strings.TrimSpace(string(b)))

				w.WriteHeader(http.StatusOK)
			case "/v2/databases/valkey-database-id/config":
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusTeapot)
				}

				if req.Method != http.MethodPatch {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				expected := `{"config":{"valkey_timeout":1200}}`
				b, err := io.ReadAll(req.Body)
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				expect.Equal(expected, strings.TrimSpace(string(b)))

				w.WriteHeader(http.StatusOK)
			case "/v2/databases/mongodb-database-id/config":
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusTeapot)
				}

				if req.Method != http.MethodPatch {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				expected := `{"config":{"verbosity":2}}`
				b, err := io.ReadAll(req.Body)
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				expect.Equal(expected, strings.TrimSpace(string(b)))

				w.WriteHeader(http.StatusOK)
			case "/v2/databases/kafka-database-id/config":
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusTeapot)
				}

				if req.Method != http.MethodPatch {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				expected := `{"config":{"group_initial_rebalance_delay_ms":3000}}`
				b, err := io.ReadAll(req.Body)
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				expect.Equal(expected, strings.TrimSpace(string(b)))

				w.WriteHeader(http.StatusOK)
			case "/v2/databases/opensearch-database-id/config":
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusTeapot)
				}

				if req.Method != http.MethodPatch {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				expected := `{"config":{"ism_history_max_age_hours":12}}`
				b, err := io.ReadAll(req.Body)
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				expect.Equal(expected, strings.TrimSpace(string(b)))

				w.WriteHeader(http.StatusOK)
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
		it("updates the mysql database config", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"database",
				"configuration",
				"update",
				"--engine", "mysql",
				"mysql-database-id",
				"--config-json", `{"sql_mode": "ANSI"}`,
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Empty(strings.TrimSpace(string(output)))
		})
	})

	when("all required flags are passed", func() {
		it("updates the pg database config", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"database",
				"configuration",
				"update",
				"--engine", "pg",
				"pg-database-id",
				"--config-json", `{"pgbouncer":{"server_reset_query_always": false}}`,
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Empty(strings.TrimSpace(string(output)))
		})
	})

	when("all required flags are passed", func() {
		it("updates the redis database config", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"database",
				"configuration",
				"update",
				"--engine", "redis",
				"redis-database-id",
				"--config-json", `{"redis_timeout":1200}`,
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Empty(strings.TrimSpace(string(output)))
		})
	})

	when("all required flags are passed", func() {
		it("updates the valkey database config", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"database",
				"configuration",
				"update",
				"--engine", "valkey",
				"valkey-database-id",
				"--config-json", `{"valkey_timeout":1200}`,
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Empty(strings.TrimSpace(string(output)))
		})
	})

	when("all required flags are passed", func() {
		it("updates the mongodb database config", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"database",
				"configuration",
				"update",
				"--engine", "mongodb",
				"mongodb-database-id",
				"--config-json", `{"verbosity":2}`,
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Empty(strings.TrimSpace(string(output)))
		})
	})

	when("all required flags are passed", func() {
		it("updates the kafka database config", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"database",
				"configuration",
				"update",
				"--engine", "kafka",
				"kafka-database-id",
				"--config-json", `{"group_initial_rebalance_delay_ms":3000}`,
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Empty(strings.TrimSpace(string(output)))
		})
	})

	when("all required flags are passed", func() {
		it("updates the opensearch database config", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"database",
				"configuration",
				"update",
				"--engine", "opensearch",
				"opensearch-database-id",
				"--config-json", `{"ism_history_max_age_hours":12}`,
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Empty(strings.TrimSpace(string(output)))
		})
	})
})

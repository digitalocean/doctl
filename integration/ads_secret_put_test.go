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

var _ = suite("ads/secret/put", func(t *testing.T, when spec.G, it spec.S) {
	const workspace = "f81d4fae-7dec-11d0-a765-00a0c91e6bf6"

	var (
		expect *require.Assertions
		server *httptest.Server
		cmd    *exec.Cmd
	)

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/v2/data-cloud-platform/nyc3/workspaces/" + workspace + "/secrets/warehouse-db":
				if req.Header.Get("Authorization") != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}
				if req.Method != http.MethodPut {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				reqBody, err := io.ReadAll(req.Body)
				expect.NoError(err)
				expect.JSONEq(adsSecretPutRequest, string(reqBody))

				w.Write([]byte(adsSecretPutResponse))
			default:
				dump, err := httputil.DumpRequest(req, true)
				if err != nil {
					t.Fatal("failed to dump request")
				}

				t.Fatalf("received unknown request: %s", dump)
			}
		}))

		cmd = exec.Command(builtBinaryPath,
			"-t", "some-magic-token",
			"-u", server.URL,
			"ads", "secret", "put", "warehouse-db",
			"--region", "nyc3",
			"--workspace", workspace,
		)
	})

	it.After(func() {
		server.Close()
	})

	when("values contain brackets, commas, quotes, and keys that differ only in case", func() {
		it("sends every property exactly as typed", func() {
			cmd.Args = append(cmd.Args,
				"--data", "password=abc]",
				"--data", "[key=v",
				"--data", "wrapped=[abc]",
				"--data", "csv=a,b",
				"--data", `quote=a"b`,
				"--data", "pad=a=b==",
				"--data", "Password=x",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Contains(string(output), "warehouse-db")
			expect.NotContains(string(output), "abc]", "output echoed a secret value")
		})
	})

	when("a key is repeated", func() {
		it("fails without sending a request", func() {
			cmd.Args = append(cmd.Args, "--data", "password=first_value", "--data", "password=second_value")

			output, err := cmd.CombinedOutput()
			expect.Error(err)
			expect.Contains(string(output), `secret property "password" is given more than once in --data; pass each property once`)
			expect.NotContains(string(output), "_value", "the error echoed a secret value")
		})
	})
})

const (
	adsSecretPutRequest = `
{
	"region": "nyc3",
	"uuid": "f81d4fae-7dec-11d0-a765-00a0c91e6bf6",
	"secret_key": "warehouse-db",
	"data": {
		"password": "abc]",
		"[key": "v",
		"wrapped": "[abc]",
		"csv": "a,b",
		"quote": "a\"b",
		"pad": "a=b==",
		"Password": "x"
	}
}`
	adsSecretPutResponse = `{"data":{"version":1}}`
)

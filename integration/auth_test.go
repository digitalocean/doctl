// +build !windows

package integration

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/creack/pty"
	"github.com/sclevine/spec"
	"github.com/stretchr/testify/require"
)

var _ = suite("auth/init", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		server *httptest.Server
	)

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			w.Header().Add("content-type", "application/json")

			switch req.URL.Path {
			case "/v2/account":
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusTeapot)
				}

				w.Write([]byte(`{ "account":{}}`))
			default:
				dump, err := httputil.DumpRequest(req, true)
				if err != nil {
					t.Fatal("failed to dump request")
				}

				t.Fatalf("received unknown request: %s", dump)
			}
		}))
	})

	when("a custom config is provided", func() {
		it("validates and saves the provided auth token", func() {
			tmpDir, err := ioutil.TempDir("", "")
			expect.NoError(err)

			testConfig := filepath.Join(tmpDir, "test-config.yml")

			cmd := exec.Command(builtBinaryPath,
				"-u", server.URL,
				"--config", testConfig,
				"auth",
				"init",
			)

			ptmx, err := pty.Start(cmd)
			expect.NoError(err)

			go func() {
				ptmx.Write([]byte("some-magic-token\n"))
			}()

			buf := bytes.NewBuffer([]byte{})

			count, _ := io.Copy(buf, ptmx) // yes, ignore error intentionally
			expect.NotZero(count)
			ptmx.Close()

			expect.Contains(buf.String(), "Validating token... OK")

			fileBytes, err := ioutil.ReadFile(testConfig)
			expect.NoError(err)

			expect.Contains(string(fileBytes), "access-token: some-magic-token")
		})
	})

	when("no custom config is provided", func() {
		it("saves the auth token to the default config path", func() {
			cmd := exec.Command(builtBinaryPath,
				"-u", server.URL,
				"auth",
				"init",
			)

			ptmx, err := pty.Start(cmd)
			expect.NoError(err)

			go func() {
				ptmx.Write([]byte("some-magic-token\n"))
			}()

			buf := bytes.NewBuffer([]byte{})

			count, _ := io.Copy(buf, ptmx) // yes, ignore error intentionally
			expect.NotZero(count)
			ptmx.Close()

			expect.Contains(buf.String(), "Validating token... OK")

			location, err := getDefaultConfigLocation()
			expect.NoError(err)

			fileBytes, err := ioutil.ReadFile(location)
			expect.NoError(err)

			expect.Contains(string(fileBytes), "access-token: some-magic-token")

			err = os.Remove(location)
			expect.NoError(err)
		})
	})

	when("a token cannot be validated", func() {
		it("exits non-zero with an error", func() {
			tmpDir, err := ioutil.TempDir("", "")
			expect.NoError(err)

			testConfig := filepath.Join(tmpDir, "test-config.yml")

			cmd := exec.Command(builtBinaryPath,
				"-u", server.URL,
				"--config", testConfig,
				"auth",
				"init",
			)

			ptmx, err := pty.Start(cmd)
			expect.NoError(err)

			go func() {
				ptmx.Write([]byte("some-bad-token\n"))
			}()

			buf := bytes.NewBuffer([]byte{})

			count, _ := io.Copy(buf, ptmx) // yes, ignore error intentionally
			expect.NotZero(count)
			ptmx.Close()

			expect.Contains(buf.String(), "Validating token... invalid token")
			expect.Contains(buf.String(), fmt.Sprintf("unable to use supplied token to access API: GET %s/v2/account: 418", server.URL))
		})
	})
})

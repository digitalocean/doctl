package acceptance

import (
	"bytes"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/creack/pty"
	"github.com/sclevine/spec"
	"github.com/stretchr/testify/require"
)

func testAuthInit(t *testing.T, when spec.G, it spec.S) {
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
				w.Write([]byte(`{ "account":{}}`))
			default:
				dump, err := httputil.DumpRequest(req, true)
				if err != nil {
					t.Fatal("failed to dump request")
				}

				t.Fatalf("received unknown request: %s", string(dump))
			}
		}))
	})

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

		_, err = io.Copy(buf, ptmx)
		expect.NoError(err)

		ptmx.Close()

		expect.Contains(buf.String(), "Validating token... OK")

		fileBytes, err := ioutil.ReadFile(testConfig)
		expect.NoError(err)

		expect.Contains(string(fileBytes), "access-token: some-magic-token")
	})
}

package integration

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/sclevine/spec"
	"github.com/stretchr/testify/require"
)

var _ = suite("invoices/csv", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		server *httptest.Server
	)

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/v2/customers/my/invoices/example-invoice-uuid/csv":
				w.Write([]byte(invoiceCSVResponse))
			default:
				dump, err := httputil.DumpRequest(req, true)
				if err != nil {
					t.Fatal("failed to dump request")
				}

				t.Fatalf("received unknown request: %s", dump)
			}
		}))
	})

	it("gets the specified invoice UUID csv", func() {
		path := os.TempDir()
		fileUUID := uuid.New().String()
		fpath := filepath.Join(path, fileUUID)

		cmd := exec.Command(builtBinaryPath,
			"-t", "some-magic-token",
			"-u", server.URL,
			"invoice",
			"csv",
			"example-invoice-uuid",
			fpath,
		)

		output, err := cmd.CombinedOutput()
		expect.NoError(err, fmt.Sprintf("received error output: %s", output))
		expect.Equal(strings.TrimSpace(invoiceCSVOutput), strings.TrimSpace(string(output)))

		result, err := ioutil.ReadFile(fpath)
		expect.NoError(err)
		expect.Equal([]byte(invoiceCSVResponse), result)

		os.Remove(fpath)
	})

})

const invoiceCSVOutput string = ""
const invoiceCSVResponse string = "csv response"

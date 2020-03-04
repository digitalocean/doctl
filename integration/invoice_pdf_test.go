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

var _ = suite("invoices/pdf", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		server *httptest.Server
	)

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/v2/customers/my/invoices/example-invoice-uuid/pdf":
				w.Write([]byte(invoicePDFResponse))
			default:
				dump, err := httputil.DumpRequest(req, true)
				if err != nil {
					t.Fatal("failed to dump request")
				}

				t.Fatalf("received unknown request: %s", dump)
			}
		}))
	})

	it("gets the specified invoice UUID pdf", func() {
		path := os.TempDir()
		fileUUID := uuid.New().String()
		fpath := filepath.Join(path, fileUUID)

		cmd := exec.Command(builtBinaryPath,
			"-t", "some-magic-token",
			"-u", server.URL,
			"invoice",
			"pdf",
			"example-invoice-uuid",
			fpath,
		)

		output, err := cmd.CombinedOutput()
		expect.NoError(err, fmt.Sprintf("received error output: %s", output))
		expect.Equal(strings.TrimSpace(invoicePDFOutput), strings.TrimSpace(string(output)))

		result, err := ioutil.ReadFile(fpath)
		expect.NoError(err)
		expect.Equal([]byte(invoicePDFResponse), result)

		os.Remove(fpath)
	})

})

const invoicePDFOutput string = ""
const invoicePDFResponse string = "pdf response"

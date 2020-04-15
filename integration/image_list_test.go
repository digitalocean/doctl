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

var _ = suite("compute/image/list", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		server *httptest.Server
	)

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/v2/images":
				if req.Method != http.MethodGet {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				t := req.URL.Query().Get("type")

				if t == "distribution" || t == "application" {
					w.WriteHeader(http.StatusBadRequest)
					return
				}

				auth := req.Header.Get("Authorization")
				if auth == "Bearer some-magic-token" {
					// Test uses the same return from ListApplication
					// as the JSON returned is identical
					w.Write([]byte(imageListApplicationResponse))
				} else if auth == "Bearer token-for-account-with-no-images" {
					w.Write([]byte(`{"images":[]}`))
				} else {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

			default:
				dump, err := httputil.DumpRequest(req, true)
				if err != nil {
					t.Fatal("failed to dump request")
				}

				t.Fatalf("received unknown request: %s", dump)
			}
		}))
	})

	when("passing public flag", func() {
		it("lists all images", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"compute",
				"image",
				"list",
				"--public",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(imageListApplicationOutput), strings.TrimSpace(string(output)))
		})
	})

	when("passing no flags", func() {
		it("lists private images", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"compute",
				"image",
				"list",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(imageListPrivateOutput), strings.TrimSpace(string(output)))
		})
	})

	when("no private images exist and public flag not passed", func() {
		it("print notice", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "token-for-account-with-no-images",
				"-u", server.URL,
				"compute",
				"image",
				"list",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(imageListNoticeWithHeader), strings.TrimSpace(string(output)))
		})
	})

	when("no private images exist and no-header is passed", func() {
		it("notice does not go to stdout", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "token-for-account-with-no-images",
				"-u", server.URL,
				"compute",
				"image",
				"list",
				"--no-header",
			)

			stdout, err := cmd.StdoutPipe()
			if err != nil {
				t.Fatal(err)
			}
			stderr, err := cmd.StderrPipe()
			if err != nil {
				t.Fatal(err)
			}
			if err := cmd.Start(); err != nil {
				t.Fatal(err)
			}

			stdoutString, err := ioutil.ReadAll(stdout)
			if err != nil {
				t.Fatal(err)
			}

			stderrString, err := ioutil.ReadAll(stderr)
			if err != nil {
				t.Fatal(err)
			}

			expect.Empty(stdoutString)
			expect.Equal(strings.TrimSpace(imageListNotice), strings.TrimSpace(string(stderrString)))
		})
	})
})

const (
	imageListPrivateOutput = `
ID         Name                                        Type    Distribution    Slug             Public    Min Disk
6376602    Ruby on Rails on 14.04 (Nginx + Unicorn)            Ubuntu          ruby-on-rails    false     20
	`
	imageListNoticeWithHeader = `
Notice: Listing private images. Use '--public' to include all images.
ID    Name    Type    Distribution    Slug    Public    Min Disk
`
	imageListNotice = `
Notice: Listing private images. Use '--public' to include all images.
`
)

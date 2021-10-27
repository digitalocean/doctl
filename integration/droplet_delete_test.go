package integration

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"os/exec"
	"strings"
	"testing"

	"github.com/sclevine/spec"
	"github.com/stretchr/testify/require"
)

var _ = suite("compute/droplet/delete", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		server *httptest.Server
	)

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/v2/droplets":
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}
				if req.URL.RawQuery == "page=1&per_page=200&tag_name=one" {
					if req.Method != http.MethodGet {
						w.WriteHeader(http.StatusMethodNotAllowed)
						return
					}

					w.Write([]byte(`{"droplets":[{"name":"some-droplet-name", "id": 1337}]}`))
				} else if req.URL.RawQuery == "tag_name=one" {
					if req.Method == http.MethodDelete {
						w.WriteHeader(http.StatusNoContent)
					} else {
						w.WriteHeader(http.StatusMethodNotAllowed)
						return
					}
				} else if req.URL.RawQuery == "page=1&per_page=200&tag_name=two" {
					if req.Method != http.MethodGet {
						w.WriteHeader(http.StatusMethodNotAllowed)
						return
					}

					w.Write([]byte(`{"droplets":[{"name":"some-droplet-name", "id": 1337}, {"name":"another-droplet-name", "id": 7331}]}`))
				} else if req.URL.RawQuery == "tag_name=two" {
					if req.Method == http.MethodDelete {
						w.WriteHeader(http.StatusNoContent)
					} else {
						w.WriteHeader(http.StatusMethodNotAllowed)
						return
					}
				} else {
					if req.Method != http.MethodGet {
						w.WriteHeader(http.StatusMethodNotAllowed)
						return
					}

					w.Write([]byte(`{"droplets":[{"name":"some-droplet-name", "id": 1337}]}`))
				}
			case "/v2/droplets/1337":
				if req.Method != http.MethodDelete {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				w.WriteHeader(http.StatusNoContent)
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
		base := []string{
			"-t", "some-magic-token",
			"compute",
			"droplet",
		}

		cases := []struct {
			desc string
			args []string
		}{
			{desc: "command is delete", args: append(base, []string{"delete", "some-droplet-name", "--force"}...)},
			{desc: "command is rm", args: append(base, []string{"rm", "some-droplet-name", "--force"}...)},
			{desc: "command is d", args: append(base, []string{"d", "some-droplet-name", "--force"}...)},
			{desc: "command is del", args: append(base, []string{"del", "some-droplet-name", "--force"}...)},
		}

		for _, c := range cases {
			commandArgs := c.args

			when(c.desc, func() {
				it("deletes a droplet", func() {
					finalArgs := append([]string{"-u", server.URL}, commandArgs...)
					cmd := exec.Command(builtBinaryPath, finalArgs...)

					output, err := cmd.CombinedOutput()
					expect.NoError(err, fmt.Sprintf("received error output: %s", output))
					expect.Empty(output)
				})
			})
		}
	})

	when("deleting by tag name", func() {
		it("deletes the right Droplet", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"compute",
				"droplet",
				"delete",
				"--tag-name", "one",
				"--force",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Empty(output)
		})
	})

	when("deleting one Droplet without force flag", func() {
		it("correctly prompts for confirmation", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"compute",
				"droplet",
				"delete",
				"some-droplet-name",
			)

			output, err := cmd.CombinedOutput()
			expect.Error(err)
			expect.Equal(strings.TrimSpace(dropletDelOutput), strings.TrimSpace(string(output)))
		})
	})

	when("deleting two Droplet without force flag", func() {
		it("correctly prompts for confirmation", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"compute",
				"droplet",
				"delete",
				"some-droplet-name",
				"another-droplet-name",
			)

			output, err := cmd.CombinedOutput()
			expect.Error(err)
			expect.Equal(strings.TrimSpace(multiDropletDelOutput), strings.TrimSpace(string(output)))
		})
	})

	when("deleting one Droplet by tag without force flag", func() {
		it("correctly prompts for confirmation", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"compute",
				"droplet",
				"delete",
				"--tag-name", "one",
			)

			output, err := cmd.CombinedOutput()
			expect.Error(err)
			expect.Equal(strings.TrimSpace(tagDropletDelOutput), strings.TrimSpace(string(output)))
		})
	})

	when("deleting two Droplet by tag without force flag", func() {
		it("correctly prompts for confirmation", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"compute",
				"droplet",
				"delete",
				"--tag-name", "two",
			)

			output, err := cmd.CombinedOutput()
			expect.Error(err)
			expect.Equal(strings.TrimSpace(tagMultiDropletDelOutput), strings.TrimSpace(string(output)))
		})
	})
})

const (
	dropletDelOutput         = "Warning: Are you sure you want to delete this Droplet? (y/N) ? Error: Operation aborted."
	multiDropletDelOutput    = "Warning: Are you sure you want to delete 2 Droplets? (y/N) ? Error: Operation aborted."
	tagDropletDelOutput      = `Warning: Are you sure you want to delete 1 Droplet tagged "one"? [affected Droplet: 1337] (y/N) ? Error: Operation aborted.`
	tagMultiDropletDelOutput = `Warning: Are you sure you want to delete 2 Droplets tagged "two"? [affected Droplets: 1337 7331] (y/N) ? Error: Operation aborted.`
)

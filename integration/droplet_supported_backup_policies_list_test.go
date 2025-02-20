package integration

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sclevine/spec"
	"github.com/stretchr/testify/require"
)

var _ = suite("compute/droplet/backup-policies/list-supported", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect     *require.Assertions
		server     *httptest.Server
		configPath string
	)

	it.Before(func() {
		expect = require.New(t)

		dir := t.TempDir()

		configPath = filepath.Join(dir, "config.yaml")

		err := os.WriteFile(configPath, []byte(dropletSupportedBackupPoliciesListConfig), 0644)
		expect.NoError(err)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/v2/droplets/backups/supported_policies":
				auth := req.Header.Get("Authorization")
				if auth != "Bearer special-broken" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				if req.Method != http.MethodGet {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				w.Write([]byte(dropletSupportedBackupPoliciesListResponse))
			default:
				dump, err := httputil.DumpRequest(req, true)
				if err != nil {
					t.Fatal("failed to dump request")
				}

				t.Fatalf("received unknown request: %s", dump)
			}
		}))
	})

	it.After(func() {
		err := os.RemoveAll(configPath)
		expect.NoError(err)
	})

	when("all required flags are passed", func() {
		it("list supported droplet backup policies", func() {
			cmd := exec.Command(builtBinaryPath,
				"-c", configPath,
				"-u", server.URL,
				"compute",
				"droplet",
				"backup-policies",
				"list-supported",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(dropletSupportedBackupPoliciesListOutput), strings.TrimSpace(string(output)))
		})
	})
})

const (
	dropletSupportedBackupPoliciesListConfig = `
---
access-token: special-broken
`
	dropletSupportedBackupPoliciesListOutput = `
Name      Possible Window Starts    Window Length Hours    Retention Period Days    Possible Days
weekly    [0 4 8 12 16 20]          4                      28                       [SUN MON TUE WED THU FRI SAT]
daily     [0 4 8 12 16 20]          4                      7                        []
`
	dropletSupportedBackupPoliciesListResponse = `
{
  "supported_policies": [
    {
      "name": "weekly",
      "possible_window_starts": [
        0,
        4,
        8,
        12,
        16,
        20
      ],
      "window_length_hours": 4,
      "retention_period_days": 28,
      "possible_days": [
        "SUN",
        "MON",
        "TUE",
        "WED",
        "THU",
        "FRI",
        "SAT"
      ]
    },
    {
      "name": "daily",
      "possible_window_starts": [
        0,
        4,
        8,
        12,
        16,
        20
      ],
      "window_length_hours": 4,
      "retention_period_days": 7,
      "possible_days": []
    }
  ]
}`
)

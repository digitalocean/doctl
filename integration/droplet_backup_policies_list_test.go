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

var _ = suite("compute/droplet/backup-policies/list", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect     *require.Assertions
		server     *httptest.Server
		configPath string
	)

	it.Before(func() {
		expect = require.New(t)

		dir := t.TempDir()

		configPath = filepath.Join(dir, "config.yaml")

		err := os.WriteFile(configPath, []byte(dropletBackupPoliciesListConfig), 0644)
		expect.NoError(err)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/v2/droplets/backups/policies":
				auth := req.Header.Get("Authorization")
				if auth != "Bearer special-broken" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				if req.Method != http.MethodGet {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				w.Write([]byte(dropletBackupPoliciesListResponse))
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
		it("list backup policies for all droplets", func() {
			cmd := exec.Command(builtBinaryPath,
				"-c", configPath,
				"-u", server.URL,
				"compute",
				"droplet",
				"backup-policies",
				"list",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(dropletBackupPoliciesListOutput), strings.TrimSpace(string(output)))
		})
	})
})

const (
	dropletBackupPoliciesListConfig = `
---
access-token: special-broken
`
	dropletBackupPoliciesListOutput = `
Droplet ID    Enabled    Plan      Weekday    Hour    Window Length Hours    Retention Period Days    Next Window Start                Next Window End
5555          true       weekly    SUN        20      4                      28                       2024-11-17 20:00:00 +0000 UTC    2024-11-18 00:00:00 +0000 UTC
`
	dropletBackupPoliciesListResponse = `
{
  "policies": {
		"5555": {
			"droplet_id": 5555,
			"backup_enabled": true,
			"backup_policy": {
				"plan": "weekly",
				"weekday": "SUN",
				"hour": 20,
				"window_length_hours": 4,
				"retention_period_days": 28
			},
			"next_backup_window": {
				"start": "2024-11-17T20:00:00Z",
				"end": "2024-11-18T00:00:00Z"
			}
		}
  },
	"links": {},
	"meta": {
		"total": 1
	}
}`
)

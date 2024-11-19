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

var _ = suite("compute/droplet/backup-policies/get", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect     *require.Assertions
		server     *httptest.Server
		configPath string
	)

	it.Before(func() {
		expect = require.New(t)

		dir := t.TempDir()

		configPath = filepath.Join(dir, "config.yaml")

		err := os.WriteFile(configPath, []byte(dropletBackupPoliciesGetConfig), 0644)
		expect.NoError(err)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/v2/droplets/5555/backups/policy":
				auth := req.Header.Get("Authorization")
				if auth != "Bearer special-broken" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				if req.Method != http.MethodGet {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				w.Write([]byte(dropletBackupPoliciesGetResponse))
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
		it("gets backup policy for the specified droplet ID", func() {
			cmd := exec.Command(builtBinaryPath,
				"-c", configPath,
				"-u", server.URL,
				"compute",
				"droplet",
				"backup-policies",
				"get",
				"5555",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(dropletBackupPoliciesGetOutput), strings.TrimSpace(string(output)))
		})
	})

	when("passing a format", func() {
		it("displays only those columns", func() {
			cmd := exec.Command(builtBinaryPath,
				"-c", configPath,
				"-u", server.URL,
				"compute",
				"droplet",
				"backup-policies",
				"get",
				"5555",
				"--format", "DropletID,BackupPolicyPlan",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(dropletBackupPoliciesGetFormatOutput), strings.TrimSpace(string(output)))
		})
	})

	when("passing a template", func() {
		it("renders the template with the values", func() {
			cmd := exec.Command(builtBinaryPath,
				"-c", configPath,
				"-u", server.URL,
				"compute",
				"droplet",
				"backup-policies",
				"get",
				"5555",
				"--template", "this droplet id {{.DropletID}} is making a backup {{.BackupPolicy.Plan}}",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(dropletBackupPoliciesGetTemplateOutput), strings.TrimSpace(string(output)))
		})
	})
})

const (
	dropletBackupPoliciesGetConfig = `
---
access-token: special-broken
`
	dropletBackupPoliciesGetOutput = `
Droplet ID    Enabled    Plan      Weekday    Hour    Window Length Hours    Retention Period Days    Next Window Start                Next Window End
5555          true       weekly    SUN        20      4                      28                       2024-11-17 20:00:00 +0000 UTC    2024-11-18 00:00:00 +0000 UTC
`
	dropletBackupPoliciesGetFormatOutput = `
Droplet ID    Plan
5555          weekly
	`

	dropletBackupPoliciesGetTemplateOutput = `
	this droplet id 5555 is making a backup weekly
	`
	dropletBackupPoliciesGetResponse = `
{
  "policy": {
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
}`
)

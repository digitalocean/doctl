package integration

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"os/exec"
	"testing"

	"github.com/sclevine/spec"
	"github.com/stretchr/testify/require"
)

var _ = suite("compute/volume/snapshot", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect   *require.Assertions
		cmd      *exec.Cmd
		baseArgs = []string{
			"my-volume-id",
			"--snapshot-desc", "some magical description",
			"--snapshot-name", "my-snapshot-name",
			"--tag", "hey",
		}
	)

	it.Before(func() {
		expect = require.New(t)

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/v2/volumes/my-volume-id/snapshots":
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				if req.Method != http.MethodPost {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				reqBody, err := ioutil.ReadAll(req.Body)
				expect.NoError(err)

				expect.JSONEq(volumeSnapshotRequest, string(reqBody))

				w.Write([]byte(volumeSnapshotResponse))
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
			"compute",
			"volume")
	})

	when("command is snapshot", func() {
		it("snapshots the volume", func() {
			args := append([]string{"snapshot"}, baseArgs...)
			cmd.Args = append(cmd.Args, args...)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Empty(output)
		})
	})

	when("command is s", func() {
		it("snapshots the volume", func() {
			args := append([]string{"s"}, baseArgs...)
			cmd.Args = append(cmd.Args, args...)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Empty(output)
		})
	})
})

const (
	volumeSnapshotResponse = `
{
  "snapshot": {
    "id": "8fa70202-873f-11e6-8b68-000f533176b1",
    "name": "big-data-snapshot1475261774",
    "regions": [
      "nyc1"
    ],
    "created_at": "2016-09-30T18:56:14Z",
    "resource_id": "82a48a18-873f-11e6-96bf-000f53315a41",
    "resource_type": "volume",
    "min_disk_size": 10,
    "size_gigabytes": 0,
    "tags": [
      "aninterestingtag"
    ]
  }
}`
	volumeSnapshotRequest = `{
  "volume_id":"my-volume-id",
  "name":"my-snapshot-name",
  "description":"some magical description",
  "tags":["hey"]
}`
)

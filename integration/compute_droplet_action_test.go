package integration

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/sclevine/spec"
	"github.com/stretchr/testify/require"
)

type request struct {
	method string
	body   string
}

var _ = suite("compute/droplet-action", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		server *httptest.Server
	)

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			pathMatch := map[string]request{
				"/v2/droplets/34/actions":      {method: http.MethodPost, body: `{"type":"shutdown"}`},
				"/v2/droplets/4444/actions":    {method: http.MethodPost, body: `{"type":"disable_backups"}`},
				"/v2/droplets/999/actions":     {method: http.MethodPost, body: `{"type":"reboot"}`},
				"/v2/droplets/48/actions":      {method: http.MethodPost, body: `{"name":"best-snapshot","type":"snapshot"}`},
				"/v2/droplets/112/actions":     {method: http.MethodPost, body: `{"name":"yes","type":"rename"}`},
				"/v2/droplets/383/actions":     {method: http.MethodPost, body: `{"image":1234,"type":"restore"}`},
				"/v2/droplets/11234/actions":   {method: http.MethodPost, body: `{"type":"power_off"}`},
				"/v2/droplets/8/actions":       {method: http.MethodPost, body: `{"type":"power_cycle"}`},
				"/v2/droplets/234/actions":     {method: http.MethodPost, body: `{"type":"password_reset"}`},
				"/v2/droplets/591/actions":     {method: http.MethodPost, body: `{"type":"enable_private_networking"}`},
				"/v2/droplets/247/actions":     {method: http.MethodPost, body: `{"type":"enable_ipv6"}`},
				"/v2/droplets/45/actions":      {method: http.MethodPost, body: `{"type":"power_on"}`},
				"/v2/droplets/1111/actions":    {method: http.MethodPost, body: `{"kernel":7777,"type":"change_kernel"}`},
				"/v2/droplets/65/actions":      {method: http.MethodPost, body: `{"type":"enable_backups"}`},
				"/v2/droplets/4743/actions":    {method: http.MethodPost, body: `{"image":9999,"type":"rebuild"}`},
				"/v2/droplets/884/actions":     {method: http.MethodPost, body: `{"disk":true,"size":"bigger","type":"resize"}`},
				"/v2/droplets/789/actions/954": {method: http.MethodGet, body: `{}`},
			}

			auth := req.Header.Get("Authorization")
			if auth != "Bearer some-magic-token" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			matchRequest, ok := pathMatch[req.URL.Path]
			if !ok {
				dump, err := httputil.DumpRequest(req, true)
				if err != nil {
					t.Fatal("failed to dump request")
				}

				t.Fatalf("received unknown request: %s", dump)
			}

			if req.Method != matchRequest.method {
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}

			if matchRequest.body != "{}" {
				reqBody, err := ioutil.ReadAll(req.Body)
				expect.NoError(err)

				expect.JSONEq(matchRequest.body, string(reqBody))
			}

			w.Write([]byte(dropletActionResponse))
		}))
	})

	cmd := exec.Command(builtBinaryPath,
		"-t", "some-magic-token",
		"compute",
		"droplet-action",
	)

	cases := []struct {
		desc string
		args []string
	}{
		{desc: "disabling backups", args: []string{"disable-backups", "4444"}},
		{desc: "change kernel", args: []string{"change-kernel", "1111", "--kernel-id", "7777"}},
		{desc: "enable backups", args: []string{"enable-backups", "65"}},
		{desc: "enable ipv6", args: []string{"enable-ipv6", "247"}},
		{desc: "enable private networking", args: []string{"enable-private-networking", "591"}},
		{desc: "password reset", args: []string{"password-reset", "234"}},
		{desc: "power cycle", args: []string{"power-cycle", "08"}},
		{desc: "power off", args: []string{"power-off", "11234"}},
		{desc: "power on", args: []string{"power-on", "045"}},
		{desc: "reboot", args: []string{"reboot", "999"}},
		{desc: "rebuild", args: []string{"rebuild", "4743", "--image", "9999"}},
		{desc: "rename", args: []string{"rename", "112", "--droplet-name", "yes"}},
		{desc: "resize", args: []string{"resize", "884", "--resize-disk", "--size", "bigger"}},
		{desc: "restore", args: []string{"restore", "383", "--image-id", "1234"}},
		{desc: "shutdown", args: []string{"shutdown", "34"}},
		{desc: "snapshot", args: []string{"snapshot", "48", "--snapshot-name", "best-snapshot"}},
		{desc: "get", args: []string{"get", "789", "--action-id", "954"}},
		{desc: "g", args: []string{"get", "789", "--action-id", "954"}},
	}

	for _, c := range cases {
		commandArgs := c.args

		when(c.desc, func() {
			it("completes the action", func() {
				cmd.Args = append(cmd.Args, commandArgs...)

				cmd.Env = append(os.Environ(),
					fmt.Sprintf("DIGITALOCEAN_API_URL=%s", server.URL),
				)

				output, err := cmd.CombinedOutput()
				expect.NoError(err, fmt.Sprintf("received error output: %s", output))
				expect.Equal(strings.TrimSpace(dropletActionOutput), strings.TrimSpace(string(output)))
			})
		})
	}
})

const (
	dropletActionOutput = `
ID          Status         Type              Started At                       Completed At    Resource ID    Resource Type    Region
36804745    in-progress    enable_backups    2014-11-14 16:30:56 +0000 UTC    <nil>           3164450        droplet          nyc3
	`
	dropletActionResponse = `
{
  "action": {
    "id": 36804745,
    "status": "in-progress",
    "type": "enable_backups",
    "started_at": "2014-11-14T16:30:56Z",
    "completed_at": null,
    "resource_id": 3164450,
    "resource_type": "droplet",
    "region": {
      "name": "New York 3",
      "slug": "nyc3",
      "sizes": [ "s-24vcpu-128gb" ],
      "features": [ "image_transfer" ],
      "available": true
    },
    "region_slug": "nyc3"
  }
}
`
)

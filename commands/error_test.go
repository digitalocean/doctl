/*
Copyright 2018 The Doctl Authors All rights reserved.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package commands

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/internal/ui"
	"github.com/digitalocean/godo"
)

// withActiveCommand installs a command for defaultNextStep to name, so a
// test can tell "no suggestion offered" apart from "no command running".
func withActiveCommand(t *testing.T, path ...string) {
	t.Helper()

	var parent *cobra.Command
	for _, use := range path {
		cmd := &cobra.Command{Use: use}
		if parent != nil {
			parent.AddCommand(cmd)
		}
		parent = cmd
	}

	prev := activeCommand
	activeCommand = parent
	t.Cleanup(func() { activeCommand = prev })
}

// apiErr builds the godo error a failed request would carry.
func apiErr(status, attempts int, message string) error {
	u, _ := url.Parse("https://api.digitalocean.com/v2/droplets/123")

	return &godo.ErrorResponse{
		Response: &http.Response{
			StatusCode: status,
			Request:    &http.Request{Method: "GET", URL: u},
		},
		Message:  message,
		Attempts: attempts,
	}
}

// The next step is a suggestion, not a reflex: --help answers a question
// about usage, so it is offered for failures doctl raised itself and
// withheld from failures the API or the network produced.
func Test_NextStep_OnlySuggestsHelpForLocalFailures(t *testing.T) {
	const help = "run doctl compute droplet get --help"

	tests := []struct {
		name string
		err  error
		want string
	}{
		{
			name: "a local validation failure is what --help explains",
			err:  errors.New("Only one of `--region` or `--droplet-id` may be specified"),
			want: help,
		},
		{
			name: "so is the wrong number of arguments",
			err:  doctl.NewMissingArgsErr("droplet.get"),
			want: help,
		},
		{
			name: "a 404 has no suggestion that fits, and does not borrow --help",
			err:  apiErr(http.StatusNotFound, 0, "not found"),
			want: "",
		},
		{
			name: "neither does a conflict",
			err:  apiErr(http.StatusConflict, 0, "droplet is locked"),
			want: "",
		},
		{
			name: "an exhausted retry has nothing left to suggest",
			err:  apiErr(http.StatusInternalServerError, 4, "broke"),
			want: "",
		},
		{
			name: "but a 500 that was never retried is worth retrying",
			err:  apiErr(http.StatusInternalServerError, 0, "broke"),
			want: "run doctl compute droplet get",
		},
		{
			name: "a status the table knows keeps its own advice",
			err:  apiErr(http.StatusUnauthorized, 0, "bad token"),
			want: "run doctl auth init",
		},
		{
			name: "an unmapped status still beats a misleading --help",
			err:  apiErr(http.StatusTeapot, 0, "teapot"),
			want: "",
		},
		{
			name: "a refused connection is not a usage question",
			err:  &url.Error{Op: "Get", URL: "https://api.digitalocean.com", Err: errors.New("connection refused")},
			want: "",
		},
		{
			name: "nor is a timeout",
			err:  fmt.Errorf("fetching droplet: %w", &net.DNSError{Err: "no such host", Name: "api.digitalocean.com"}),
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withActiveCommand(t, "doctl", "compute", "droplet", "get")

			assert.Equal(t, tt.want, genericStructuredError{err: tt.err}.NextStep())
		})
	}
}

// An error that implements NextStepper has already decided, so an empty
// string from it means "say nothing" rather than "fall back to --help".
func Test_NextStep_HonorsAnEmptyOverride(t *testing.T) {
	withActiveCommand(t, "doctl", "compute", "droplet", "get")

	for _, err := range []error{errOperationAborted, errConfirmationRequired} {
		assert.Empty(t, genericStructuredError{err: err}.NextStep(), err.Error())
	}

	assert.Equal(t, "run doctl auth list",
		genericStructuredError{err: errUnknownAuthContext}.NextStep())
}

// godo puts the raw body in Message when that body was not the JSON it
// expected, so anything between doctl and the API can land an HTML error
// page in the reason line. The canned reason for the status beats that.
func Test_Reason_RefusesAPayloadMasqueradingAsAMessage(t *testing.T) {
	const gatewayReason = "the API gateway could not reach the upstream service"

	nginxPage := "<html>\n<head><title>502 Bad Gateway</title></head>\n<body>\n" +
		strings.Repeat("<!-- padding -->\n", 40) + "</body>\n</html>"

	tests := []struct {
		name string
		msg  string
		want string
	}{
		{
			name: "a sentence the API wrote is quoted as-is",
			msg:  "Droplet is currently locked by an in-progress event",
			want: "Droplet is currently locked by an in-progress event",
		},
		{
			name: "an HTML error page is not",
			msg:  nginxPage,
			want: gatewayReason,
		},
		{
			name: "neither is a single long line of markup",
			msg:  "<html><body>" + strings.Repeat("x", 300) + "</body></html>",
			want: gatewayReason,
		},
		{
			name: "nor anything that runs past a reasonable sentence",
			msg:  strings.Repeat("verbose ", 40),
			want: gatewayReason,
		},
		{
			name: "an empty message falls back too",
			msg:  "",
			want: gatewayReason,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reason := genericStructuredError{
				err: apiErr(http.StatusBadGateway, 0, tt.msg),
			}.Reason()

			assert.Equal(t, tt.want, reason)
			assert.NotContains(t, reason, "\n", "the reason stays one line")
		})
	}
}

// The attempt count has to survive that fallback, or a 502 behind a proxy
// loses the one detail saying the retries were already spent.
func Test_Reason_KeepsAttemptCountOnTheCannedReason(t *testing.T) {
	err := apiErr(http.StatusBadGateway, 4, "<html>a proxy page</html>")

	assert.Equal(t,
		"the API gateway could not reach the upstream service (gave up after 4 attempt(s))",
		genericStructuredError{err: err}.Reason())
}

// A call site that wrapped the API error to say what it was attempting
// knows something the status name does not - which droplet, which key.
func Test_Title_PrefersWhatTheCommandWasAttempting(t *testing.T) {
	bare := apiErr(http.StatusConflict, 0, "locked")
	assert.Equal(t, "Conflict", genericStructuredError{err: bare}.Title())

	wrapped := fmt.Errorf("Unable to delete Droplet 111: %w", bare)
	title := genericStructuredError{err: wrapped}.Title()

	assert.Equal(t, "Unable to delete Droplet 111", title)
	// The method and URL godo puts in Error() stay out of the title.
	assert.NotContains(t, title, "http")
}

func Test_checkErr(t *testing.T) {
	defer func(a func()) { errAction = a }(errAction)

	errAction = func() {
	}

	t.Run("a redirected stream gets the word", func(t *testing.T) {
		var b bytes.Buffer
		withUIEnv(t, ui.Plain(&b, &b))

		checkErr(errors.New("an error"))

		// Same label a FlagValidationError carries, so both read as one voice.
		assert.Equal(t, "Error: an error\n", b.String())
	})

	// The glyph is a screen affordance, so it is gated on stderr being a
	// terminal rather than on color. Everything parsing doctl's stderr - the
	// integration suite included - matches on the plain form above.
	t.Run("a terminal is led by the glyph", func(t *testing.T) {
		var b bytes.Buffer
		withUIEnv(t, ui.Env{Out: &b, Err: &b, ErrTTY: true})

		checkErr(errors.New("an error"))

		assert.Equal(t, ui.GlyphFailure+" Error: an error\n", b.String())
	})
}

func Test_checkErr_FlagValidationJSONKeepsErrorsEnvelope(t *testing.T) {
	defer func(a func()) { errAction = a }(errAction)
	defer func() { viper.Set("output", "") }()

	errAction = func() {}
	viper.Set("output", "json")

	// Capture stdout where JSON is printed.
	old := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = w

	checkErr(&FlagValidationError{
		Command: "doctl compute droplet create",
		Issues: []FlagIssue{
			{Flag: "size", Problem: "is required but was not set", Purpose: "Droplet size"},
		},
	})

	require.NoError(t, w.Close())
	os.Stdout = old
	out, err := io.ReadAll(r)
	require.NoError(t, err)

	var payload outputErrors
	require.NoError(t, json.Unmarshal(out, &payload))
	require.Len(t, payload.Errors, 1)
	// The detail is the one-line summary, not the block a terminal is shown:
	// automation parsing this envelope wants a sentence rather than a rendered
	// layout complete with glyphs and suggested next commands.
	assert.Equal(t, "missing required flag --size for doctl compute droplet create", payload.Errors[0].Detail)
	assert.NotContains(t, string(out), `"issues"`)
}

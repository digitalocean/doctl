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

package doctl

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// traceRoundTrip sends one request through a recorder and returns the
// request and response dumps it emitted, plus the body the caller received.
func traceRoundTrip(t *testing.T, method, path, reqBody, respBody string) (string, string, string) {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, respBody)
	}))
	t.Cleanup(server.Close)

	rec := newRecorder(http.DefaultTransport)
	var reqDump, respDump string
	done := make(chan struct{})
	go func() {
		reqDump = <-rec.req
		respDump = <-rec.resp
		close(done)
	}()

	req, err := http.NewRequest(method, server.URL+path, strings.NewReader(reqBody))
	require.NoError(t, err)
	resp, err := rec.RoundTrip(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	got, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	<-done

	return reqDump, respDump, string(got)
}

func TestRecorderRedactsADSQueryPassword(t *testing.T) {
	_, respDump, body := traceRoundTrip(t, http.MethodGet,
		"/v2/data-cloud-platform/nyc3/workspaces/ws-uuid/query/credentials", "",
		`{"user":"reader","password":"pa\"ss_value"}`)

	assert.NotContains(t, respDump, "ss_value")
	assert.Contains(t, respDump, `"password":"[REDACTED]"`)
	assert.Contains(t, respDump, `"user":"reader"`)
	// The caller still receives the real response.
	assert.Equal(t, `{"user":"reader","password":"pa\"ss_value"}`, body)
}

func TestRecorderRedactsADSAccessToken(t *testing.T) {
	_, respDump, _ := traceRoundTrip(t, http.MethodPost,
		"/v2/data-cloud-platform/nyc3/tokens", `{"region":"nyc3"}`,
		`{"id":"token-id","access_token": "secret_value"}`)

	assert.NotContains(t, respDump, "secret_value")
	assert.Contains(t, respDump, `"access_token": "[REDACTED]"`)
	assert.Contains(t, respDump, `"id":"token-id"`)
}

func TestRecorderRedactsADSSecretWriteBody(t *testing.T) {
	reqDump, _, _ := traceRoundTrip(t, http.MethodPut,
		"/v2/data-cloud-platform/nyc3/workspaces/ws-uuid/secrets/db",
		`{"region":"nyc3","uuid":"ws-uuid","secret_key":"db","data":{"password":"secret}_value"}}`,
		`{"data":{"version":1}}`)

	assert.NotContains(t, reqDump, "secret}_value")
	assert.NotContains(t, reqDump, "_value")
	assert.True(t, strings.HasSuffix(reqDump, "[REDACTED]"), reqDump)
	assert.Contains(t, reqDump, "PUT /v2/data-cloud-platform/nyc3/workspaces/ws-uuid/secrets/db")
}

func TestRecorderLeavesOtherRoutesUnchanged(t *testing.T) {
	_, respDump, _ := traceRoundTrip(t, http.MethodGet,
		"/v2/databases/db-uuid", "", `{"password":"shown"}`)

	assert.Contains(t, respDump, `"password":"shown"`)
}

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
	"fmt"
	"net/http"
	"net/http/httputil"
	"regexp"
	"strings"
)

// recorder traces http connections. It sends the output to a request and
// response channels.
type recorder struct {
	wrap http.RoundTripper
	req  chan string
	resp chan string
}

func newRecorder(transport http.RoundTripper) *recorder {
	return &recorder{
		wrap: transport,
		req:  make(chan string),
		resp: make(chan string),
	}
}

func (rec *recorder) RoundTrip(req *http.Request) (*http.Response, error) {
	reqBytes, err := httputil.DumpRequestOut(req, true)
	if err != nil {
		return nil, fmt.Errorf("transport.Recorder: dumping request, %v", err)
	}
	rec.req <- redactADSTrace(req, string(reqBytes), true)

	resp, rerr := rec.wrap.RoundTrip(req)
	if rerr != nil {
		return nil, rerr
	}

	respBytes, err := httputil.DumpResponse(resp, true)
	if err != nil {
		return nil, fmt.Errorf("transport.Recorder: dumping response, %v", err)
	}
	rec.resp <- redactADSTrace(req, string(respBytes), false)

	return resp, nil
}

// adsTracePath marks Agent Data Services routes, whose bodies can carry
// secrets that --trace must not print.
const adsTracePath = "/v2/data-cloud-platform/"

const adsTraceRedacted = "[REDACTED]"

// adsSecretFieldPattern matches the query password and the one-time access
// token, including JSON strings with escaped quotes.
var adsSecretFieldPattern = regexp.MustCompile(`("(?:password|access_token)"\s*:\s*)"(?:[^"\\]|\\.)*"`)

// redactADSTrace hides secret values in a dumped ADS request or response.
// Workspace secret writes have their whole body replaced, because every
// value in them is secret. Other traffic is returned unchanged.
func redactADSTrace(req *http.Request, dump string, isRequest bool) string {
	if req == nil || req.URL == nil || !strings.Contains(req.URL.Path, adsTracePath) {
		return dump
	}

	if isRequest && strings.Contains(req.URL.Path, "/secrets/") && req.Method != http.MethodGet && req.Method != http.MethodDelete {
		if i := strings.Index(dump, "\r\n\r\n"); i >= 0 && i+4 < len(dump) {
			return dump[:i+4] + adsTraceRedacted
		}
		return dump
	}

	return adsSecretFieldPattern.ReplaceAllString(dump, `${1}"`+adsTraceRedacted+`"`)
}

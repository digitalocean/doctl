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
	rec.req <- string(reqBytes)

	resp, rerr := rec.wrap.RoundTrip(req)
	if rerr != nil {
		return nil, rerr
	}

	respBytes, err := httputil.DumpResponse(resp, true)
	if err != nil {
		return nil, fmt.Errorf("transport.Recorder: dumping response, %v", err)
	}
	rec.resp <- string(respBytes)

	return resp, nil
}

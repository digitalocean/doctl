package doit

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

	respBytes, err := httputil.DumpResponse(resp, true)
	if err != nil {
		return nil, fmt.Errorf("transport.Recorder: dumping response, %v", err)
	}
	rec.resp <- string(respBytes)

	return resp, rerr
}

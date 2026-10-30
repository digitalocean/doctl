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
	"bytes"
	"context"
	"errors"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/digitalocean/doctl/internal/deviceid"
	"github.com/digitalocean/godo"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/spf13/viper"
	"golang.org/x/oauth2"
)

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}

func TestUserAgent(t *testing.T) {
	pattern := `doctl\/([0-9]+\.?){3}(-dev)? \(([\w ]+){2}\)`
	re := regexp.MustCompile(pattern)

	t.Run("release version", func(t *testing.T) {
		dv := DoitVersion
		DoitVersion = Version{Major: 0, Minor: 1, Patch: 2}

		ua := userAgent()
		if !re.MatchString(ua) {
			t.Errorf("expected %s to match %s", ua, pattern)
		}
		DoitVersion = dv
	})

	t.Run("dev version", func(t *testing.T) {
		ua := userAgent()
		if !re.MatchString(ua) {
			t.Errorf("expected %s to match %s", ua, pattern)
		}
	})
}

func TestVersion(t *testing.T) {
	slr1 := &stubLatestRelease{version: "0.1.0"}
	slr2 := &stubLatestRelease{version: "1.0.0"}

	cases := []struct {
		v    Version
		s    string
		json string
		ver  string
		slr  LatestVersioner
	}{
		// version with no label
		{
			v: Version{Major: 0, Minor: 1, Patch: 2},
			s: `doctl version 0.1.2
API endpoint: https://api.digitalocean.com/`,
			json: "{\n  \"version\": \"0.1.2\",\n  \"apiEndpoint\": \"https://api.digitalocean.com/\",\n  \"latestRelease\": \"0.1.0\"\n}",
			ver:  "0.1.2",
			slr:  slr1,
		},
		// version with label
		{
			v: Version{Major: 0, Minor: 1, Patch: 2, Label: "dev"},
			s: `doctl version 0.1.2-dev
API endpoint: https://api.digitalocean.com/`,
			json: "{\n  \"version\": \"0.1.2-dev\",\n  \"apiEndpoint\": \"https://api.digitalocean.com/\",\n  \"latestRelease\": \"0.1.0\"\n}",
			ver:  "0.1.2-dev",
			slr:  slr1,
		},
		// version with label and build
		{
			v:    Version{Major: 0, Minor: 1, Patch: 2, Label: "dev", Build: "12345"},
			s:    "doctl version 0.1.2-dev\nGit commit hash: 12345\nAPI endpoint: https://api.digitalocean.com/",
			json: "{\n  \"version\": \"0.1.2-dev\",\n  \"commit\": \"12345\",\n  \"apiEndpoint\": \"https://api.digitalocean.com/\",\n  \"latestRelease\": \"0.1.0\"\n}",
			ver:  "0.1.2-dev",
			slr:  slr1,
		},
		// version with no label and higher released version
		{
			v:    Version{Major: 0, Minor: 1, Patch: 2},
			s:    "doctl version 0.1.2\nAPI endpoint: https://api.digitalocean.com/\nrelease 1.0.0 is available, check it out! ",
			json: "{\n  \"version\": \"0.1.2\",\n  \"apiEndpoint\": \"https://api.digitalocean.com/\",\n  \"latestRelease\": \"1.0.0\",\n  \"notification\": \"release 1.0.0 is available, check it out!\"\n}",
			ver:  `0.1.2`,
			slr:  slr2,
		},
		// version with dev label and released version
		{
			v:    Version{Major: 1, Minor: 0, Patch: 0, Label: "dev"},
			s:    "doctl version 1.0.0-dev\nAPI endpoint: https://api.digitalocean.com/\nrelease 1.0.0 is available, check it out! ",
			json: "{\n  \"version\": \"1.0.0-dev\",\n  \"apiEndpoint\": \"https://api.digitalocean.com/\",\n  \"latestRelease\": \"1.0.0\",\n  \"notification\": \"release 1.0.0 is available, check it out!\"\n}",
			ver:  `1.0.0-dev`,
			slr:  slr2,
		},
		// version with release label and released version available
		{
			v:    Version{Major: 1, Minor: 0, Patch: 0, Label: "release"},
			s:    "doctl version 1.0.0-release\nAPI endpoint: https://api.digitalocean.com/",
			json: "{\n  \"version\": \"1.0.0-release\",\n  \"apiEndpoint\": \"https://api.digitalocean.com/\",\n  \"latestRelease\": \"1.0.0\"\n}",
			ver:  `1.0.0-release`,
			slr:  slr2,
		},
	}

	for _, c := range cases {
		if got, want := c.v.String(), c.ver; got != want {
			t.Errorf("version string for %#v = %q; want = %q", c.v, got, want)
		}
		if got, want := c.v.Complete(c.slr), c.s; got != want {
			t.Errorf("complete version string for %#v = %q; want = %q", c.v, got, want)
		}
		if got, want := c.v.CompleteJSON(c.slr), c.json; got != want {
			t.Errorf("complete version json for %#v = %q; want = %q", c.v, got, want)
		}
	}
}

type stubLatestRelease struct {
	version string
}

func (slr stubLatestRelease) LatestVersion() (string, error) {
	return slr.version, nil
}

// TestGetGodoClientBaseURL pins the precedence that lets a caller-supplied
// base URL act as a default while --api-url (DIGITALOCEAN_API_URL) still wins.
func TestGetGodoClientBaseURL(t *testing.T) {
	const customDefault = "https://agents.example.test/"
	cases := []struct {
		name   string
		apiURL string
		opts   []godo.ClientOpt
		want   string
	}{
		{
			name: "no options and no api-url leaves godo's default",
			want: "https://api.digitalocean.com/",
		},
		{
			name: "caller-supplied base URL applies",
			opts: []godo.ClientOpt{godo.SetBaseURL(customDefault)},
			want: customDefault,
		},
		{
			name:   "api-url wins over a caller-supplied base URL",
			apiURL: "https://ohr-agent.do-ai-test.run/",
			opts:   []godo.ClientOpt{godo.SetBaseURL(customDefault)},
			want:   "https://ohr-agent.do-ai-test.run/",
		},
		{
			name:   "api-url applies with no caller options",
			apiURL: "https://example.test/",
			want:   "https://example.test/",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			viper.Set("api-url", c.apiURL)
			t.Cleanup(func() { viper.Set("api-url", "") })

			client, err := (&LiveConfig{}).GetGodoClient(false, false, "fake-token", c.opts...)
			if err != nil {
				t.Fatalf("GetGodoClient() unexpected error: %v", err)
			}
			if got := client.BaseURL.String(); got != c.want {
				t.Errorf("BaseURL = %q; want %q", got, c.want)
			}
		})
	}
}

func TestResolvedAPIURL(t *testing.T) {
	t.Run("default", func(t *testing.T) {
		viper.Set("api-url", "")
		t.Cleanup(func() { viper.Set("api-url", "") })
		if got, want := ResolvedAPIURL(), DefaultAPIURL; got != want {
			t.Errorf("ResolvedAPIURL() = %q; want %q", got, want)
		}
	})

	t.Run("override", func(t *testing.T) {
		const custom = "https://ohr-agent.do-ai-test.run/"
		viper.Set("api-url", custom)
		t.Cleanup(func() { viper.Set("api-url", "") })
		if got, want := ResolvedAPIURL(), custom; got != want {
			t.Errorf("ResolvedAPIURL() = %q; want %q", got, want)
		}
	})
}

func TestCommandName(t *testing.T) {
	t.Run("snap name set to goland", func(t *testing.T) {
		const snapName = "goland"

		t.Setenv("SNAP_NAME", snapName)

		// When run under `go test`, os.Args[0] will be different every time,
		// so only check that the Snap name is not "goland".
		if actual := CommandName(); actual == snapName {
			t.Errorf("expected name not to equal %s, got %s", snapName, actual)
		}
	})

	t.Run("snap name set to doctl", func(t *testing.T) {
		const expected = "doctl"

		t.Setenv("SNAP_NAME", expected)

		if actual := CommandName(); actual != expected {
			t.Errorf("got %s, want %s", actual, expected)
		}
	})
}

// TestGetGodoClientSetsResponseHeaderTimeout pins the header timeout onto the
// transport that actually dials, on both the retrying and non-retrying paths.
//
// The retrying one is the reason this test exists: godo discards the transport
// it is handed and builds its own, so setRetryResponseHeaderTimeout has to walk
// godo's wrapping to reach it. That walk is coupled to godo's internals, and a
// mismatch would silently restore the unbounded header wait of MARSOHS-2367
// rather than fail anywhere visible. This is what fails instead.
func TestGetGodoClientSetsResponseHeaderTimeout(t *testing.T) {
	const want = 30 * time.Second

	viper.Set("http-response-header-timeout", int(want.Seconds()))
	defer viper.Set("http-response-header-timeout", nil)

	for _, tc := range []struct {
		name         string
		retryMax     int
		allowRetries bool
	}{
		{name: "with retries", retryMax: 5, allowRetries: true},
		{name: "retries disallowed", retryMax: 5, allowRetries: false},
		{name: "retries not configured", retryMax: 0, allowRetries: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// GetGodoClient reads this off viper, and the flag that normally
			// binds it lives in the commands package.
			viper.Set("http-retry-max", tc.retryMax)
			defer viper.Set("http-retry-max", nil)

			client, err := (&LiveConfig{}).GetGodoClient(false, tc.allowRetries, "fake-token")
			if err != nil {
				t.Fatalf("GetGodoClient: %v", err)
			}

			transport := dialingTransport(t, client.HTTPClient.Transport)
			if transport.ResponseHeaderTimeout != want {
				t.Errorf("ResponseHeaderTimeout = %v, want %v", transport.ResponseHeaderTimeout, want)
			}
		})
	}
}

// TestIsResponseHeaderTimeout drives a real transport into its header timeout on
// both protocols and checks the error is recognised.
//
// Deliberately not synthesized errors. Both implementations return an error type
// that is unexported and that nothing distinguishes from a dial or TLS timeout
// except its message, so the check matches on text — and a change to that text
// upstream has to fail here. Silently failing to match would leave the retries
// unreported, which is the whole thing reportResponseHeaderTimeouts exists to
// prevent. HTTP/2 is covered because that is what the API negotiates.
func TestIsResponseHeaderTimeout(t *testing.T) {
	for _, tc := range []struct {
		name      string
		http2     bool
		wantProto string
	}{
		{name: "http/1.1", wantProto: "HTTP/1.1"},
		{name: "http/2", http2: true, wantProto: "HTTP/2.0"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			blocked := make(chan struct{})

			mux := http.NewServeMux()
			mux.HandleFunc("/fast", func(w http.ResponseWriter, r *http.Request) {})
			mux.HandleFunc("/slow", func(w http.ResponseWriter, r *http.Request) {
				<-blocked
			})

			srv := httptest.NewUnstartedServer(mux)
			srv.EnableHTTP2 = tc.http2
			if tc.http2 {
				srv.StartTLS()
			} else {
				srv.Start()
			}
			// Registered before the unblock so it runs after it: Close waits on
			// the in-flight /slow handler, which only returns once blocked is.
			t.Cleanup(srv.Close)
			t.Cleanup(func() { close(blocked) })

			client := srv.Client()
			transport, ok := client.Transport.(*http.Transport)
			if !ok {
				t.Fatalf("test server client transport is %T, want *http.Transport", client.Transport)
			}
			transport.ResponseHeaderTimeout = 50 * time.Millisecond

			// Confirms the protocol under test is the one actually negotiated,
			// so the HTTP/2 case cannot pass on an HTTP/1 error.
			resp, err := client.Get(srv.URL + "/fast")
			if err != nil {
				t.Fatalf("GET /fast: %v", err)
			}
			resp.Body.Close()
			if resp.Proto != tc.wantProto {
				t.Fatalf("negotiated %s, want %s", resp.Proto, tc.wantProto)
			}

			if _, err := client.Get(srv.URL + "/slow"); err == nil {
				t.Fatal("GET /slow returned no error, want a header timeout")
			} else if !isResponseHeaderTimeout(err) {
				t.Errorf("isResponseHeaderTimeout(%v) = false, want true", err)
			}
		})
	}

	t.Run("other errors", func(t *testing.T) {
		for _, err := range []error{
			nil,
			errors.New("boom"),
			// A timeout, but not this one: the message is what separates them.
			&net.OpError{Op: "dial", Err: fakeTimeout{}},
		} {
			if isResponseHeaderTimeout(err) {
				t.Errorf("isResponseHeaderTimeout(%v) = true, want false", err)
			}
		}
	})
}

type fakeTimeout struct{}

func (fakeTimeout) Error() string { return "i/o timeout" }
func (fakeTimeout) Timeout() bool { return true }

// TestReportResponseHeaderTimeouts checks the hook reports the timeouts it is
// there to make visible, and defers to the policy it wrapped for everything
// else rather than replacing it.
func TestReportResponseHeaderTimeouts(t *testing.T) {
	headerTimeout := &url.Error{
		Op:  "Get",
		URL: "https://api.digitalocean.com/v2/account",
		Err: fakeHeaderTimeout{},
	}

	for _, tc := range []struct {
		name       string
		err        error
		policy     bool
		wantReport bool
	}{
		{name: "header timeout", err: headerTimeout, policy: true, wantReport: true},
		// retryablehttp consults the policy on every failed attempt and only
		// then checks the budget, so a timeout on the final attempt of a losing
		// request arrives here too and still has to be reported.
		{name: "header timeout, not retried", err: headerTimeout, policy: false, wantReport: true},
		{name: "other error", err: errors.New("boom"), policy: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var out bytes.Buffer
			var wrapped bool

			rt := &retryablehttp.RoundTripper{Client: retryablehttp.NewClient()}
			rt.Client.CheckRetry = func(context.Context, *http.Response, error) (bool, error) {
				wrapped = true
				return tc.policy, nil
			}

			reportResponseHeaderTimeouts(rt, log.New(&out, "doctl: ", 0), 30*time.Second)

			retry, err := rt.Client.CheckRetry(context.Background(), nil, tc.err)
			if err != nil {
				t.Fatalf("CheckRetry: %v", err)
			}
			if !wrapped {
				t.Error("the wrapped policy was not consulted")
			}
			if retry != tc.policy {
				t.Errorf("retry = %v, want the wrapped policy's %v", retry, tc.policy)
			}

			reported := out.Len() > 0
			if reported != tc.wantReport {
				t.Errorf("reported = %v (%q), want %v", reported, out.String(), tc.wantReport)
			}
			if tc.wantReport && !strings.Contains(out.String(), "30s") {
				t.Errorf("report %q does not name the timeout that fired", out.String())
			}
		})
	}
}

type fakeHeaderTimeout struct{}

func (fakeHeaderTimeout) Error() string { return "net/http: timeout awaiting response headers" }
func (fakeHeaderTimeout) Timeout() bool { return true }

// dialingTransport unwraps the round trippers doctl and godo layer over the
// transport that owns the connection, failing on anything unexpected so the
// assertion above cannot pass against a layer that is merely absent.
func dialingTransport(t *testing.T, rt http.RoundTripper) *http.Transport {
	t.Helper()

	for {
		switch v := rt.(type) {
		case *http.Transport:
			return v
		case *deviceid.Transport:
			rt = v.Base
		case *oauth2.Transport:
			rt = v.Base
		case *retryablehttp.RoundTripper:
			if v.Client == nil || v.Client.HTTPClient == nil {
				t.Fatal("retryablehttp round tripper has no client")
			}
			rt = v.Client.HTTPClient.Transport
		default:
			t.Fatalf("unexpected round tripper %T in the chain", rt)
			return nil
		}
	}
}

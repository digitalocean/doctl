/*
Copyright 2026 The Doctl Authors All rights reserved.
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
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/doctl/internal/agentproxy"
	"github.com/digitalocean/doctl/internal/deviceid"
	"github.com/digitalocean/godo"
	"github.com/pkg/browser"
)

// canvasPath is where the openhands template serves Agent Canvas on the
// session's published port: the UI under /canvas, the agent-server's API
// paths beside it on the same origin.
const canvasPath = "/canvas"

// canvasPort is the guest port the openhands template publishes through
// ingress and serves Canvas on; the port-forward fallback targets it.
const canvasPort = 8000

// RunAgentsCanvas serves the session's Agent Canvas UI on a local port.
//
// The session's public URL (its ingress hostname) accepts one credential,
// the team's token as `Authorization: Bearer`, and nothing a browser can send
// on its own: no cookie, no query string. So doctl listens on localhost, adds
// that header to every request and to WebSocket upgrades, and forwards to the
// public URL. The browser opens http://127.0.0.1:<port>/canvas; the token never
// reaches the page. Same idea as `port-forward`, without the tunnel: the
// public URL already carries HTTP and WebSocket.
func RunAgentsCanvas(c *CmdConfig) error {
	if len(c.Args) < 1 {
		return doctl.NewMissingArgsErr(c.NS)
	}
	sessionID, err := resolveSessionRef(c.HostedAgents(), c.Args[0])
	if err != nil {
		return err
	}
	if err := ensureSessionAwake(c, sessionID, "served to Agent Canvas"); err != nil {
		return err
	}
	target, err := resolveCanvasTarget(c.HostedAgents(), c.Args[0], sessionID)
	if err != nil {
		return err
	}
	address, err := c.Doit.GetString(c.NS, doctl.ArgAgentForwardAddress)
	if err != nil {
		return err
	}
	port, err := c.Doit.GetInt(c.NS, doctl.ArgAgentCanvasPort)
	if err != nil {
		return err
	}
	open, err := c.Doit.GetBool(c.NS, doctl.ArgAgentCanvasOpen)
	if err != nil {
		return err
	}
	if address != "127.0.0.1" && address != "localhost" && address != "::1" {
		warn("binding %s lets anyone who can reach this address use your token against the session", address)
	}

	ln, err := net.Listen("tcp", net.JoinHostPort(address, strconv.Itoa(port)))
	if err != nil {
		return fmt.Errorf("cannot listen on %s:%d: %w", address, port, err)
	}
	localPort := ln.Addr().(*net.TCPAddr).Port
	local := fmt.Sprintf("http://%s%s", net.JoinHostPort(address, strconv.Itoa(localPort)), canvasPath)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	server := &http.Server{
		Handler:           loopbackOnly(newCanvasProxy(target, c.getContextAccessToken(), deviceid.Get())),
		ReadHeaderTimeout: 30 * time.Second,
		// Requests are rooted in the signal context, so an in-flight
		// request (a long WebSocket, say) is cancelled on Ctrl-C rather than
		// left to the shutdown grace period.
		BaseContext: func(net.Listener) context.Context { return ctx },
	}
	errCh := make(chan error, 1)
	go func() { errCh <- server.Serve(ln) }()

	fmt.Fprintf(c.Out, "Agent Canvas for session %s\n  local:  %s\n  public: %s (token added by doctl, never sent to the browser)\n",
		sessionID, local, target)
	fmt.Fprintln(c.Out, "Ready. Press Ctrl-C to stop.")
	if open {
		if err := browser.OpenURL(local); err != nil {
			warn("could not open a browser (%v); open %s yourself", err, local)
		}
	}

	select {
	case <-ctx.Done():
	case err := <-errCh:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("canvas proxy: %w", err)
		}
	}
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_ = server.Shutdown(shutdownCtx)
	return nil
}

// resolveCanvasTarget turns a session into the public URL the proxy forwards
// to, or an error that says why there is none: an environment that does not
// publish session URLs yet (harness-api answers 501 or 404) gets the
// port-forward route spelled out, since that reaches the same Canvas.
func resolveCanvasTarget(svc do.HostedAgentsService, sessionRef, sessionID string) (*url.URL, error) {
	ing, err := svc.GetSessionIngress(sessionID)
	if err != nil {
		return nil, canvasIngressError(sessionRef, err)
	}
	if ing.State != do.IngressReady {
		return nil, fmt.Errorf("the session's public URL is not ready yet (state %s); try again in a moment", ing.State)
	}
	target, err := url.Parse(ing.URL)
	if err != nil || target.Scheme == "" || target.Host == "" {
		return nil, fmt.Errorf("the session's public URL %q is not usable: %v", ing.URL, err)
	}
	return target, nil
}

// canvasIngressError explains a failed ingress lookup. Two answers are known
// states with a known way around, not faults, and they differ: a 501 means
// this environment does not publish session URLs at all (prod today), a 404
// means this session has none (not an openhands session, or its sandbox was
// not provisioned for ingress). Both get the tunnel route, which reaches the
// same Canvas; the port and path are the openhands template's.
func canvasIngressError(sessionRef string, err error) error {
	tunnel := fmt.Sprintf("Reach Agent Canvas through a tunnel instead:\n"+
		"  %s port-forward %s %d:%d\n"+
		"then open http://127.0.0.1:%d%s", agentCLI, sessionRef, canvasPort, canvasPort, canvasPort, canvasPath)
	var er *godo.ErrorResponse
	if errors.As(err, &er) && er.Response != nil {
		switch er.Response.StatusCode {
		case http.StatusNotImplemented:
			return fmt.Errorf("this environment does not publish public URLs for sessions yet (HTTP 501).\n%s", tunnel)
		case http.StatusNotFound:
			return fmt.Errorf("this session has no public URL (HTTP 404): only openhands sessions serve Agent Canvas on port %d, "+
				"and the session's sandbox must have been provisioned with ingress.\n%s", canvasPort, tunnel)
		}
	}
	return beautifyAgentError(err)
}

// loopbackOnly refuses what a token-adding proxy on localhost must refuse: a
// request from a browser page on another origin (Origin header present and
// not loopback), and a request whose Host is not loopback (DNS rebinding: a
// page resolving its own name to 127.0.0.1 would otherwise reach the proxy
// with a loopback Origin check that never fires). Plain tools with no Origin,
// and pages served from localhost, pass. The same Origin rule start-proxy
// applies (agentproxy.AllowedProxyOrigin).
func loopbackOnly(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !agentproxy.AllowedProxyOrigin(r.Header.Get("Origin")) {
			http.Error(w, "forbidden: this proxy only serves pages on localhost", http.StatusForbidden)
			return
		}
		host := r.Host
		if h, _, err := net.SplitHostPort(host); err == nil {
			host = h
		}
		switch strings.Trim(host, "[]") {
		case "localhost", "127.0.0.1", "::1":
		default:
			http.Error(w, "forbidden: this proxy only answers on localhost", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// newCanvasProxy forwards every request, WebSocket upgrades included, to the
// session's public URL with the team's token added. The browser never sees the
// token: it talks to localhost, and the Authorization header is set on the
// outbound leg only. The standard ReverseProxy drops hop-by-hop headers on
// both legs and splices an upgraded connection on a 101.
func newCanvasProxy(target *url.URL, token, deviceID string) http.Handler {
	return &httputil.ReverseProxy{
		Rewrite: func(pr *httputil.ProxyRequest) {
			pr.SetURL(target)
			pr.Out.Host = target.Host
			if token != "" {
				pr.Out.Header.Set("Authorization", "Bearer "+token)
			}
			if deviceID != "" {
				pr.Out.Header.Set(deviceid.HeaderName, deviceID)
			}
		},
		// The dial or TLS error goes to the terminal, not to the page: the
		// browser gets a plain 502, as the stdlib default would give.
		ErrorHandler: func(w http.ResponseWriter, _ *http.Request, err error) {
			warn("canvas proxy: upstream request failed: %v", err)
			http.Error(w, "the session's public URL did not answer; see the doctl terminal", http.StatusBadGateway)
		},
	}
}

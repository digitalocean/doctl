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
	"syscall"
	"time"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/doctl/internal/deviceid"
	"github.com/digitalocean/godo"
	"github.com/pkg/browser"
)

// canvasPath is where the openhands template serves Agent Canvas on the
// session's published port: the UI under /canvas, the agent-server's API
// paths beside it on the same origin.
const canvasPath = "/canvas"

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

	server := &http.Server{
		Handler:           newCanvasProxy(target, c.getContextAccessToken(), deviceid.Get()),
		ReadHeaderTimeout: 30 * time.Second,
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
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

// canvasIngressError explains a failed ingress lookup. A 501 (not enabled in
// this environment) or 404 (no URL for this session) is a known state with a
// known way around, not a fault.
func canvasIngressError(sessionRef string, err error) error {
	var er *godo.ErrorResponse
	if errors.As(err, &er) && er.Response != nil &&
		(er.Response.StatusCode == http.StatusNotImplemented || er.Response.StatusCode == http.StatusNotFound) {
		return fmt.Errorf("this environment does not publish a public URL for sessions yet (HTTP %d).\n"+
			"Reach Agent Canvas through a tunnel instead:\n"+
			"  %s port-forward %s 8000:8000\n"+
			"then open http://127.0.0.1:8000%s",
			er.Response.StatusCode, agentCLI, sessionRef, canvasPath)
	}
	return beautifyAgentError(err)
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
		ErrorHandler: func(w http.ResponseWriter, _ *http.Request, err error) {
			http.Error(w, "upstream error: "+err.Error(), http.StatusBadGateway)
		},
	}
}

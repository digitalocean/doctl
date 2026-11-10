package commands

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/doctl/internal/deviceid"
	"github.com/digitalocean/godo"
	"github.com/gorilla/websocket"
)

// doctl agents port-forward — POC client for the port-forward WebSocket API
// (docs/design/port-forward-rfc.md § User experience). One local TCP listener
// per requested pair; each accepted connection dials its own WebSocket to
// GET /v2/agents/sessions/{session_id}/port-forward/{port} and pumps binary
// frames as an opaque byte stream.

const portForwardCopyBuf = 32 << 10

const (
	// handshakeBodyLimit bounds what we read from a rejected upgrade. gorilla
	// already caps its replayable copy at 1 KiB; this guards the rest.
	handshakeBodyLimit = 4 << 10
	// rejectionMessageMax keeps an HTML error page from flooding the terminal.
	rejectionMessageMax = 300
)

// Close codes harness-api sends when a tunnel ends
// (docs/design/port-forward-rfc.md § Public API). The close *reason* on the
// wire is a fixed, non-diagnostic string by contract, so the text a user can
// act on has to be built here, keyed on the code alone.
const (
	closeCodeSessionEnded = 4001 // session ended or sandbox gone — give up
	closeCodeDialFailed   = 4002 // guest dial failed (nothing listening)
	closeCodeInternal     = 4011 // unexpected server error — retryable
)

// defaultAPIURL mirrors godo's default base URL, which godo does not export.
// The tunnel has to resolve to the same host the rest of the agents commands
// talk to: a session is region-scoped to the host that created it, so dialing
// a tunnel at a different host 404s on every session.
const defaultAPIURL = "https://api.digitalocean.com/"

// forwardPair is one [<local>:]<remote> mapping. Local 0 lets the OS pick.
type forwardPair struct {
	local  int
	remote int
}

// parseForwardPair parses "[<local-port>:]<remote-port>". A bare port forwards
// the same port on both ends; "0:<remote>" lets the OS pick the local port.
func parseForwardPair(arg string) (forwardPair, error) {
	bad := func() (forwardPair, error) {
		return forwardPair{}, fmt.Errorf("invalid port pair %q: expected [<local-port>:]<remote-port>", arg)
	}
	localRaw, remoteRaw := "", arg
	if i := strings.IndexByte(arg, ':'); i >= 0 {
		localRaw, remoteRaw = arg[:i], arg[i+1:]
	}
	remote, err := strconv.Atoi(remoteRaw)
	if err != nil {
		return bad()
	}
	if remote < 1024 || remote > 65535 {
		return forwardPair{}, fmt.Errorf("remote port must be between 1024 and 65535 (got %d)", remote)
	}
	local := remote
	if localRaw != "" {
		local, err = strconv.Atoi(localRaw)
		if err != nil || local < 0 || local > 65535 {
			return bad()
		}
	}
	return forwardPair{local: local, remote: remote}, nil
}

// hostedAgentsWSURL builds the WebSocket URL for one remote port, honoring the
// --api-url / DIGITALOCEAN_API_URL override the HTTP client also uses, and the
// same --region host selection. This URL is built here rather than by the godo
// client, so without the region it would keep dialing the default region and
// 404 on every session held elsewhere.
func hostedAgentsWSURL(sessionID string, remotePort int) (string, error) {
	raw, err := agentRegionBaseURL()
	if err != nil {
		return "", err
	}
	u, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("invalid API URL %q: %w", raw, err)
	}
	switch u.Scheme {
	case "https":
		u.Scheme = "wss"
	case "http":
		u.Scheme = "ws"
	default:
		return "", fmt.Errorf("unsupported API URL scheme %q", u.Scheme)
	}
	u.Path = strings.TrimSuffix(u.Path, "/") +
		fmt.Sprintf("/v2/agents/sessions/%s/port-forward/%d", sessionID, remotePort)
	return u.String(), nil
}

// handshakeBody reads what the server said when it answered the upgrade.
// gorilla preserves up to 1 KiB of a rejected handshake's body precisely so
// callers can report it, and replaces the body with an empty reader on
// success, so this is safe to call either way.
func handshakeBody(resp *http.Response) string {
	if resp == nil || resp.Body == nil {
		return ""
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(io.LimitReader(resp.Body, handshakeBodyLimit))
	if err != nil {
		return ""
	}
	return string(b)
}

// serverRejection explains a refused upgrade in the server's own words. The
// status code alone cannot tell harness-api declining the guest dial apart
// from a proxy in front of it failing to reach harness-api at all, and that
// distinction is usually the entire diagnosis.
func serverRejection(resp *http.Response, body string, err error) error {
	if resp == nil {
		return fmt.Errorf("dialing tunnel: %w", err)
	}
	if msg := rejectionMessage(body); msg != "" {
		return fmt.Errorf("server rejected tunnel (%s): %s", resp.Status, msg)
	}
	return fmt.Errorf("server rejected tunnel (%s): %w", resp.Status, err)
}

// rejectionMessage extracts a human-readable reason from an error body: the
// {"id","message"} shape the API returns, or the raw text collapsed to one
// line for anything else (a proxy answering with plain text or HTML).
func rejectionMessage(body string) string {
	body = strings.TrimSpace(body)
	if body == "" {
		return ""
	}
	var apiErr struct {
		Message string `json:"message"`
	}
	if err := json.Unmarshal([]byte(body), &apiErr); err == nil && apiErr.Message != "" {
		return apiErr.Message
	}
	return collapseToLine(body, rejectionMessageMax)
}

// collapseToLine squashes whitespace and caps length so an error page stays
// readable on one terminal line.
func collapseToLine(s string, max int) string {
	s = strings.Join(strings.Fields(s), " ")
	r := []rune(s)
	if len(r) > max {
		return string(r[:max]) + "..."
	}
	return s
}

// wsTracer logs the tunnel handshake under --trace. doctl's recorder wraps
// godo's RoundTripper, but gorilla dials its own connection and never goes
// through it, so without this the one request worth tracing is the only one
// --trace cannot see. A nil tracer is a no-op, which is the untraced case.
type wsTracer struct {
	logger *log.Logger
}

func newWSTracer() *wsTracer {
	if !Trace {
		return nil
	}
	return &wsTracer{logger: log.New(os.Stderr, "doctl: ", log.LstdFlags)}
}

// handshake logs the upgrade request and whatever came back, using the same
// "->" / "<-" framing as the godo recorder. gorilla adds the Upgrade,
// Connection and Sec-WebSocket-* headers itself, so they are absent here.
func (t *wsTracer) handshake(wsURL string, header http.Header, resp *http.Response, body string) {
	if t == nil {
		return
	}
	t.logger.Println("->", strconv.Quote(fmt.Sprintf("GET %s %s", wsURL, formatHeader(header))))
	if resp == nil {
		return
	}
	line := fmt.Sprintf("%s %s %s", resp.Proto, resp.Status, formatHeader(resp.Header))
	if trimmed := strings.TrimSpace(body); trimmed != "" {
		line += " " + collapseToLine(trimmed, handshakeBodyLimit)
	}
	t.logger.Println("<-", strconv.Quote(line))
}

// formatHeader renders headers on one line with sensitive values masked;
// trace output gets pasted into tickets.
func formatHeader(h http.Header) string {
	keys := make([]string, 0, len(h))
	for k := range h {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		v := strings.Join(h[k], ",")
		switch {
		case strings.EqualFold(k, "Authorization"):
			v = "[redacted]"
		case strings.EqualFold(k, deviceid.HeaderName):
			v = redactDeviceID(v)
		}
		parts = append(parts, k+": "+v)
	}
	return strings.Join(parts, "; ")
}

// redactDeviceID masks a hardware id the way harness-api's device.Redact does,
// so a trace line and the server's log line for the same request stay
// correlatable without either recording the full identifier.
func redactDeviceID(id string) string {
	s := strings.TrimSpace(id)
	switch {
	case s == "":
		return ""
	case len(s) <= 8:
		return "***"
	default:
		return s[:4] + "***" + s[len(s)-4:]
	}
}

// tunnelHeader builds the WebSocket upgrade headers. Both values are
// injected rather than read here so this stays testable: deviceid.Get caches
// its lookup in a sync.Once.
//
// The device id is what the deviceid transport stamps on every /v2/agents
// request, but that transport wraps godo's RoundTripper and gorilla dials its
// own connection. Without it the tunnel is the one agents call harness-api's
// device middleware logs with an empty id, and it is also the one that opens
// a raw TCP channel into the sandbox.
func tunnelHeader(token, deviceID string) http.Header {
	header := http.Header{}
	if token != "" {
		header.Set("Authorization", "Bearer "+token)
	}
	if deviceID != "" {
		header.Set(deviceid.HeaderName, deviceID)
	}
	return header
}

// ensureSessionAwakeForPortForward resumes a paused session before opening
// tunnels, matching launch/attach so users don't have to sequence resume
// themselves (BUG-044). Terminal sessions are rejected; already-ready
// sessions are left alone.
func ensureSessionAwakeForPortForward(c *CmdConfig, sessionID string) (*do.HostedAgentSession, error) {
	return ensureSessionAwakeSession(c, sessionID, "port-forwarded")
}

// ensureSessionAwake is the shared half of the above: `verb` names what the
// caller is about to do with the session, for the terminal-status error
// ("cannot be port-forwarded", "cannot serve Agent Canvas").
func ensureSessionAwake(c *CmdConfig, sessionID string, verb string) error {
	_, err := ensureSessionAwakeSession(c, sessionID, verb)
	return err
}

// ensureSessionAwakeSession is ensureSessionAwake that also hands back the
// session as it was read (before any resume), for callers that need its
// agent kind.
func ensureSessionAwakeSession(c *CmdConfig, sessionID string, verb string) (*do.HostedAgentSession, error) {
	svc := c.HostedAgents()
	sess, err := svc.GetSession(sessionID)
	if err != nil {
		return nil, beautifyAgentError(err)
	}
	if isTerminalSessionStatus(sess.Status) {
		return nil, fmt.Errorf("session %s is %s and cannot be %s",
			displaySessionRef(sess), humanSessionStatus(sess.Status), verb)
	}
	if sess.Status != godo.HostedAgentSessionStatusPaused {
		return sess, nil
	}

	stylingEnabled = detectStyling()
	fmt.Fprintf(c.Out, "%s\n", colorize(
		fmt.Sprintf("Session %s is paused — resuming…", displaySessionRef(sess)), colMuted))
	// port-forward is the one agent command outside the pretty-error wrapper,
	// so the prepay card has to be built here or a gated resume would print a
	// raw REST dump.
	if err := svc.ResumeSession(sessionID); err != nil {
		return nil, beautifyAgentErrorFor(c, err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), runWaitTimeout)
	defer cancel()
	prog := newCreationProgress(c.Out)
	defer prog.stop()
	if _, err := waitForSessionReady(ctx, svc, sessionID, prog); err != nil {
		return nil, beautifyAgentError(err)
	}
	return sess, nil
}

// RunAgentsPortForward opens local TCP tunnels to ports inside the session's
// sandbox and blocks until interrupted.
func RunAgentsPortForward(c *CmdConfig) error {
	if len(c.Args) < 2 {
		return doctl.NewMissingArgsErr(c.NS)
	}
	sessionID, err := resolveSessionRef(c.HostedAgents(), c.Args[0])
	if err != nil {
		return err
	}
	sess, err := ensureSessionAwakeForPortForward(c, sessionID)
	if err != nil {
		return err
	}
	pairs := make([]forwardPair, 0, len(c.Args)-1)
	for _, arg := range c.Args[1:] {
		p, err := parseForwardPair(arg)
		if err != nil {
			return err
		}
		pairs = append(pairs, p)
	}
	address, err := c.Doit.GetString(c.NS, doctl.ArgAgentForwardAddress)
	if err != nil {
		return err
	}
	if address != "127.0.0.1" && address != "localhost" && address != "::1" {
		warn("binding %s exposes the tunnel to anyone who can reach this address", address)
	}
	// An openhands session's Canvas port is served as HTTP rather than as raw
	// bytes, so the Automate tab's /api/automation calls can be answered here
	// (canvasAutomationServer); everything else still crosses the tunnel. The
	// tab's answers carry actions taken with the user's token, so they are
	// given on a loopback bind only.
	canvasHTTP := sess != nil && sess.HostedAgentSession != nil && sess.AgentKind == hostedAgentKindOpenHands
	var automation http.Handler
	if canvasHTTP && isLoopbackBind(address) {
		automation = loopbackOnly(newCanvasAutomationServer(sessionID, c.HostedAgentTriggers()))
	}

	header := tunnelHeader(c.getContextAccessToken(), deviceid.Get())

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	tracer := newWSTracer()

	var wg sync.WaitGroup
	listeners := make([]net.Listener, 0, len(pairs))
	for _, pair := range pairs {
		wsURL, err := hostedAgentsWSURL(sessionID, pair.remote)
		if err != nil {
			return err
		}
		ln, err := net.Listen("tcp", net.JoinHostPort(address, strconv.Itoa(pair.local)))
		if err != nil {
			for _, l := range listeners {
				l.Close()
			}
			return fmt.Errorf("cannot listen on %s:%d: %w", address, pair.local, err)
		}
		listeners = append(listeners, ln)
		localPort := ln.Addr().(*net.TCPAddr).Port
		fmt.Fprintf(c.Out, "Forwarding %s:%d -> port %d in session %s\n",
			address, localPort, pair.remote, sessionID)

		warner := newTunnelWarner(pair.remote)
		if canvasHTTP && pair.remote == canvasPort {
			if automation != nil {
				fmt.Fprintf(c.Out, "  Agent Canvas: http://%s%s (its Automate tab is answered by doctl through the triggers API)\n",
					net.JoinHostPort(address, strconv.Itoa(localPort)), canvasPath)
			} else {
				fmt.Fprintf(c.Out, "  Agent Canvas: http://%s%s (Automate tab off: it needs a loopback --address)\n",
					net.JoinHostPort(address, strconv.Itoa(localPort)), canvasPath)
			}
			wg.Add(1)
			go func() {
				defer wg.Done()
				serveCanvasForward(ctx, ln, canvasForwardHandler(wsURL, header, tracer, warner, automation))
			}()
			continue
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			acceptForwardLoop(ctx, ln, wsURL, header, tracer, warner)
		}()
	}
	fmt.Fprintln(c.Out, "Ready. Press Ctrl-C to stop.")

	<-ctx.Done()
	for _, ln := range listeners {
		ln.Close()
	}
	wg.Wait()
	return nil
}

// explainTunnelClose says what a failed tunnel means for the port the user
// asked for. The server's close reason is deliberately terse and says nothing
// about what to do next, and "websocket: close 4002: guest port unreachable"
// reads as a platform fault when the usual cause is that the process meant to
// serve the port was never started.
func explainTunnelClose(err error, remotePort int) string {
	var ce *websocket.CloseError
	if !errors.As(err, &ce) {
		return fmt.Sprintf("port %d: connection closed: %v", remotePort, err)
	}
	switch ce.Code {
	case closeCodeDialFailed:
		return fmt.Sprintf("port %d: nothing is listening on that port inside the session — "+
			"start the process that serves it, then retry", remotePort)
	case closeCodeSessionEnded:
		return fmt.Sprintf("port %d: the session ended while forwarding — "+
			"start it again, then re-run %s port-forward", remotePort, agentCLI)
	case closeCodeInternal:
		return fmt.Sprintf("port %d: the tunnel failed inside the service — retry; "+
			"if it keeps happening, re-run with --trace", remotePort)
	case websocket.CloseAbnormalClosure:
		return fmt.Sprintf("port %d: the tunnel dropped without closing cleanly — "+
			"check your network; new connections still work", remotePort)
	default:
		return fmt.Sprintf("port %d: connection closed: %v", remotePort, err)
	}
}

// tunnelWarner reports tunnel failures for one forwarded port, printing each
// distinct message once. One page load opens several connections, so an
// unstarted server repeats the same warning once per connection otherwise —
// the second one tells the user nothing the first did not.
type tunnelWarner struct {
	remotePort int
	mu         sync.Mutex
	seen       map[string]int
}

func newTunnelWarner(remotePort int) *tunnelWarner {
	return &tunnelWarner{remotePort: remotePort, seen: map[string]int{}}
}

func (w *tunnelWarner) report(err error) {
	msg := explainTunnelClose(err, w.remotePort)

	w.mu.Lock()
	seen := w.seen[msg]
	w.seen[msg] = seen + 1
	w.mu.Unlock()

	switch seen {
	case 0:
		warn("%s", msg)
	case 1:
		warn("port %d: same failure again; further identical warnings are suppressed", w.remotePort)
	}
}

// acceptForwardLoop accepts local connections and bridges each over its own
// WebSocket. A per-connection failure never kills the listener.
func acceptForwardLoop(ctx context.Context, ln net.Listener, wsURL string, header http.Header, tracer *wsTracer, warner *tunnelWarner) {
	for {
		conn, err := ln.Accept()
		if err != nil {
			return // listener closed on shutdown
		}
		go func() {
			defer conn.Close()
			if err := bridgeLocalConn(ctx, conn, wsURL, header, tracer); err != nil && ctx.Err() == nil {
				warner.report(err)
			}
		}()
	}
}

// bridgeLocalConn dials the port-forward WebSocket for one accepted TCP
// connection and pumps bytes both ways until either side closes.
func bridgeLocalConn(ctx context.Context, local net.Conn, wsURL string, header http.Header, tracer *wsTracer) error {
	ws, resp, err := websocket.DefaultDialer.DialContext(ctx, wsURL, header)
	body := handshakeBody(resp)
	tracer.handshake(wsURL, header, resp, body)
	if err != nil {
		return serverRejection(resp, body, err)
	}
	defer ws.Close()

	errCh := make(chan error, 2)

	// WS → local. Read errors include server close frames (normal closure when
	// the guest hangs up, 4xxx codes for session-end/dial-failure).
	go func() {
		for {
			mt, data, err := ws.ReadMessage()
			if err != nil {
				if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
					errCh <- nil
				} else {
					errCh <- err
				}
				return
			}
			if mt != websocket.BinaryMessage {
				continue
			}
			if _, err := local.Write(data); err != nil {
				errCh <- nil // local reader went away; not an error worth printing
				return
			}
		}
	}()

	// local → WS. This goroutine owns all WS writes (gorilla allows one
	// concurrent writer), including the close frame when the local side ends.
	go func() {
		buf := make([]byte, portForwardCopyBuf)
		for {
			n, err := local.Read(buf)
			if n > 0 {
				if werr := ws.WriteMessage(websocket.BinaryMessage, buf[:n]); werr != nil {
					errCh <- werr
					return
				}
			}
			if err != nil {
				_ = ws.WriteMessage(websocket.CloseMessage,
					websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
				errCh <- nil
				return
			}
		}
	}()

	err = <-errCh
	local.Close()
	ws.Close()
	return err
}

// canvasForwardHandler is the HTTP face of a tunnel to an openhands
// session's Canvas port: a request under /api/automation goes to the
// automation handler (when given), any other request, WebSocket upgrades
// included, is proxied through the tunnel to the guest. The guest sees the
// request as it came, Host and all, since that is what the raw tunnel would
// carry; the ReverseProxy only owns hop-by-hop headers and the 101 splice.
func canvasForwardHandler(wsURL string, header http.Header, tracer *wsTracer, warner *tunnelWarner, automation http.Handler) http.Handler {
	transport := &http.Transport{
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			return dialTunnelConn(ctx, wsURL, header, tracer)
		},
		DisableCompression: true,
		MaxIdleConns:       8,
		IdleConnTimeout:    90 * time.Second,
	}
	proxy := &httputil.ReverseProxy{
		Rewrite: func(pr *httputil.ProxyRequest) {
			pr.SetURL(&url.URL{Scheme: "http", Host: "canvas.tunnel"})
			pr.Out.Host = pr.In.Host
		},
		Transport:     transport,
		FlushInterval: -1,
		ErrorHandler: func(w http.ResponseWriter, _ *http.Request, err error) {
			if warner != nil {
				warner.report(err)
			}
			http.Error(w, "the session did not answer through the tunnel; see the doctl terminal", http.StatusBadGateway)
		},
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if automation != nil && (r.URL.Path == canvasAutomationPrefix || strings.HasPrefix(r.URL.Path, canvasAutomationPrefix+"/")) {
			automation.ServeHTTP(w, r)
			return
		}
		proxy.ServeHTTP(w, r)
	})
}

// serveCanvasForward runs the HTTP server for one Canvas pair until ctx ends.
func serveCanvasForward(ctx context.Context, ln net.Listener, handler http.Handler) {
	server := &http.Server{
		Handler:           handler,
		ReadHeaderTimeout: 30 * time.Second,
		BaseContext:       func(net.Listener) context.Context { return ctx },
	}
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()
	_ = server.Serve(ln)
}

// dialTunnelConn opens one port-forward WebSocket and presents it as a
// net.Conn: binary frames in are bytes out, bytes in are binary frames out.
func dialTunnelConn(ctx context.Context, wsURL string, header http.Header, tracer *wsTracer) (net.Conn, error) {
	ws, resp, err := websocket.DefaultDialer.DialContext(ctx, wsURL, header)
	body := handshakeBody(resp)
	if tracer != nil {
		tracer.handshake(wsURL, header, resp, body)
	}
	if err != nil {
		return nil, serverRejection(resp, body, err)
	}
	return &wsTunnelConn{ws: ws}, nil
}

// wsTunnelConn adapts a gorilla WebSocket to net.Conn for http.Transport.
type wsTunnelConn struct {
	ws      *websocket.Conn
	readBuf []byte
	writeMu sync.Mutex
	closed  bool
}

func (c *wsTunnelConn) Read(p []byte) (int, error) {
	for len(c.readBuf) == 0 {
		mt, data, err := c.ws.ReadMessage()
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
				return 0, io.EOF
			}
			return 0, err
		}
		if mt == websocket.BinaryMessage {
			c.readBuf = data
		}
	}
	n := copy(p, c.readBuf)
	c.readBuf = c.readBuf[n:]
	return n, nil
}

func (c *wsTunnelConn) Write(p []byte) (int, error) {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	if c.closed {
		return 0, net.ErrClosed
	}
	if err := c.ws.WriteMessage(websocket.BinaryMessage, p); err != nil {
		return 0, err
	}
	return len(p), nil
}

func (c *wsTunnelConn) Close() error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	if c.closed {
		return nil
	}
	c.closed = true
	_ = c.ws.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	return c.ws.Close()
}

func (c *wsTunnelConn) LocalAddr() net.Addr  { return c.ws.UnderlyingConn().LocalAddr() }
func (c *wsTunnelConn) RemoteAddr() net.Addr { return c.ws.UnderlyingConn().RemoteAddr() }
func (c *wsTunnelConn) SetDeadline(t time.Time) error {
	return errors.Join(c.ws.SetReadDeadline(t), c.ws.SetWriteDeadline(t))
}
func (c *wsTunnelConn) SetReadDeadline(t time.Time) error  { return c.ws.SetReadDeadline(t) }
func (c *wsTunnelConn) SetWriteDeadline(t time.Time) error { return c.ws.SetWriteDeadline(t) }

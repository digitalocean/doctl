package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/internal/deviceid"
	"github.com/digitalocean/godo"
	"github.com/gorilla/websocket"
	"github.com/spf13/viper"
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
// --api-url / DIGITALOCEAN_API_URL override the HTTP client also uses.
func hostedAgentsWSURL(sessionID string, remotePort int) (string, error) {
	raw := viper.GetString("api-url")
	if raw == "" {
		raw = defaultAPIURL
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
func ensureSessionAwakeForPortForward(c *CmdConfig, sessionID string) error {
	svc := c.HostedAgents()
	sess, err := svc.GetSession(sessionID)
	if err != nil {
		return beautifyAgentError(err)
	}
	if isTerminalSessionStatus(sess.Status) {
		return fmt.Errorf("session %s is %s and cannot be port-forwarded",
			displaySessionRef(sess), humanSessionStatus(sess.Status))
	}
	if sess.Status != godo.HostedAgentSessionStatusPaused {
		return nil
	}

	stylingEnabled = detectStyling()
	fmt.Fprintf(c.Out, "%s\n", colorize(
		fmt.Sprintf("Session %s is paused — resuming…", displaySessionRef(sess)), colMuted))
	if err := svc.ResumeSession(sessionID); err != nil {
		return beautifyAgentError(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), runWaitTimeout)
	defer cancel()
	prog := newCreationProgress(c.Out)
	defer prog.stop()
	if _, err := waitForSessionReady(ctx, svc, sessionID, prog); err != nil {
		return beautifyAgentError(err)
	}
	return nil
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
	if err := ensureSessionAwakeForPortForward(c, sessionID); err != nil {
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

		wg.Add(1)
		go func() {
			defer wg.Done()
			acceptForwardLoop(ctx, ln, wsURL, header, tracer)
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

// acceptForwardLoop accepts local connections and bridges each over its own
// WebSocket. A per-connection failure never kills the listener.
func acceptForwardLoop(ctx context.Context, ln net.Listener, wsURL string, header http.Header, tracer *wsTracer) {
	for {
		conn, err := ln.Accept()
		if err != nil {
			return // listener closed on shutdown
		}
		go func() {
			defer conn.Close()
			if err := bridgeLocalConn(ctx, conn, wsURL, header, tracer); err != nil && ctx.Err() == nil {
				warn("connection closed: %v", err)
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

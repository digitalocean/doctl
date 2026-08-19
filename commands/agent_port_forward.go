package commands

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"

	"github.com/digitalocean/doctl"
	"github.com/gorilla/websocket"
	"github.com/spf13/viper"
)

// doctl agents port-forward — POC client for the port-forward WebSocket API
// (docs/design/port-forward.md § User experience). One local TCP listener per
// requested pair; each accepted connection dials its own WebSocket to
// GET /v2/agents/sessions/{session_id}/port-forward/{port} and pumps binary
// frames as an opaque byte stream.

const portForwardCopyBuf = 32 << 10

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
		raw = doctl.HostedAgentsAPIURL
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

	header := http.Header{}
	if token := c.getContextAccessToken(); token != "" {
		header.Set("Authorization", "Bearer "+token)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

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
			acceptForwardLoop(ctx, c, ln, wsURL, header)
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
func acceptForwardLoop(ctx context.Context, c *CmdConfig, ln net.Listener, wsURL string, header http.Header) {
	for {
		conn, err := ln.Accept()
		if err != nil {
			return // listener closed on shutdown
		}
		go func() {
			defer conn.Close()
			if err := bridgeLocalConn(ctx, conn, wsURL, header); err != nil && ctx.Err() == nil {
				warn("connection closed: %v", err)
			}
		}()
	}
}

// bridgeLocalConn dials the port-forward WebSocket for one accepted TCP
// connection and pumps bytes both ways until either side closes.
func bridgeLocalConn(ctx context.Context, local net.Conn, wsURL string, header http.Header) error {
	ws, resp, err := websocket.DefaultDialer.DialContext(ctx, wsURL, header)
	if err != nil {
		if resp != nil {
			return fmt.Errorf("server rejected tunnel (%s): %w", resp.Status, err)
		}
		return fmt.Errorf("dialing tunnel: %w", err)
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

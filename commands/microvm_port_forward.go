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
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/godo"
	"github.com/gorilla/websocket"
)

// doctl compute microvm port-forward: one local TCP listener per port pair.
// Each accepted connection opens its own WebSocket to
// GET /v2/microvms/{id}/port_forward?port=N. The server upgrades first and
// then sends a text frame: {"status":{"state":"connected"}} when the guest
// port is open, or {"error":...} followed by a normal close. The first frame
// therefore decides whether the tunnel opened; without reading it, a refused
// port looks like an empty reply.
//
// This file does not share code with `doctl agents port-forward`: the two
// tunnels speak different protocols and are owned separately.

// microVMAgentPort is the guest port of the MicroVM agent. The API refuses to
// forward it; doctl checks first so the error comes before a listener opens.
const microVMAgentPort = 50051

const (
	// microVMCopyBuf is the read size for local → tunnel copies.
	microVMCopyBuf = 32 << 10
	// microVMRejectionBodyLimit bounds what is read from a refused upgrade.
	microVMRejectionBodyLimit = 4 << 10
	// microVMRejectionMax keeps an HTML error page to one terminal line.
	microVMRejectionMax = 300
)

var (
	// microVMResumeTimeout bounds the wait for a paused MicroVM to run.
	microVMResumeTimeout = 2 * time.Minute
	// microVMPollInterval is the delay between state checks while resuming.
	microVMPollInterval = 2 * time.Second
	// microVMTunnelOpenTimeout bounds the wait for the first control frame.
	microVMTunnelOpenTimeout = time.Minute
)

// microVMForwardPair is one [<local>:]<remote> mapping. Local 0 lets the OS pick.
type microVMForwardPair struct {
	local  int
	remote int
}

// parseMicroVMForwardPair parses "[<local-port>:]<remote-port>". A bare port
// forwards the same port on both ends.
func parseMicroVMForwardPair(arg string) (microVMForwardPair, error) {
	bad := fmt.Errorf("invalid port pair %q: expected [<local-port>:]<remote-port>", arg)
	localRaw, remoteRaw, hasLocal := strings.Cut(arg, ":")
	if !hasLocal {
		localRaw, remoteRaw = "", arg
	}
	remote, err := strconv.Atoi(remoteRaw)
	if err != nil {
		return microVMForwardPair{}, bad
	}
	if remote < 1 || remote > 65535 {
		return microVMForwardPair{}, fmt.Errorf("remote port must be between 1 and 65535 (got %d)", remote)
	}
	if remote == microVMAgentPort {
		return microVMForwardPair{}, fmt.Errorf("remote port %d is reserved for the MicroVM agent and cannot be forwarded", microVMAgentPort)
	}
	local := remote
	if hasLocal {
		local, err = strconv.Atoi(localRaw)
		if err != nil || local < 0 || local > 65535 {
			return microVMForwardPair{}, bad
		}
	}
	return microVMForwardPair{local: local, remote: remote}, nil
}

// RunMicroVMPortForward opens local TCP tunnels to ports inside a MicroVM and
// blocks until interrupted.
func RunMicroVMPortForward(c *CmdConfig) error {
	if len(c.Args) < 2 {
		return doctl.NewMissingArgsErr(c.NS)
	}
	id := c.Args[0]
	pairs := make([]microVMForwardPair, 0, len(c.Args)-1)
	for _, arg := range c.Args[1:] {
		p, err := parseMicroVMForwardPair(arg)
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
	token := c.getContextAccessToken()
	if token == "" {
		return errors.New("access token is required for MicroVM port-forward")
	}
	if err := ensureMicroVMRunning(c, id); err != nil {
		return err
	}

	header := http.Header{}
	header.Set("Authorization", "Bearer "+token)
	var tracer *log.Logger
	if Trace {
		tracer = log.New(os.Stderr, "doctl: ", log.LstdFlags)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	var wg sync.WaitGroup
	var listeners []net.Listener
	defer func() {
		for _, ln := range listeners {
			ln.Close()
		}
		wg.Wait()
	}()

	for _, pair := range pairs {
		wsURL, err := c.MicroVMs().PortForwardURL(id, pair.remote)
		if err != nil {
			return err
		}
		ln, err := net.Listen("tcp", net.JoinHostPort(address, strconv.Itoa(pair.local)))
		if err != nil {
			return fmt.Errorf("cannot listen on %s:%d: %w", address, pair.local, err)
		}
		listeners = append(listeners, ln)
		localPort := ln.Addr().(*net.TCPAddr).Port
		fmt.Fprintf(c.Out, "Forwarding %s -> port %d in MicroVM %s\n",
			net.JoinHostPort(address, strconv.Itoa(localPort)), pair.remote, id)

		warner := &microVMTunnelWarner{remotePort: pair.remote, seen: map[string]int{}}
		wg.Add(1)
		go func() {
			defer wg.Done()
			microVMAcceptLoop(ctx, ln, wsURL, header, tracer, warner)
		}()
	}
	fmt.Fprintln(c.Out, "Ready. Press Ctrl-C to stop.")

	<-ctx.Done()
	return nil
}

// ensureMicroVMRunning resumes a paused MicroVM and waits until it runs, so
// the first tunnel works when auto-resume is off. States that cannot serve a
// tunnel are refused.
func ensureMicroVMRunning(c *CmdConfig, id string) error {
	svc := c.MicroVMs()
	md, err := svc.Get(id)
	if err != nil {
		return err
	}
	switch md.State {
	case godo.MicroVMStateRunning:
		return nil
	case godo.MicroVMStatePaused:
		fmt.Fprintf(c.Out, "MicroVM %s is paused. Resuming it...\n", id)
		if _, err := svc.Resume(id); err != nil {
			return err
		}
	case godo.MicroVMStateCreating, godo.MicroVMStateResuming:
		// Wait below.
	default:
		return fmt.Errorf("MicroVM %s is %s; port-forward needs it running", id, md.State)
	}

	deadline := time.Now().Add(microVMResumeTimeout)
	for {
		time.Sleep(microVMPollInterval)
		md, err = svc.Get(id)
		if err != nil {
			return err
		}
		if md.State == godo.MicroVMStateRunning {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("MicroVM %s is still %s after %s", id, md.State, microVMResumeTimeout)
		}
	}
}

// microVMAcceptLoop accepts local connections and bridges each over its own
// WebSocket. A per-connection failure never stops the listener.
func microVMAcceptLoop(ctx context.Context, ln net.Listener, wsURL string, header http.Header, tracer *log.Logger, warner *microVMTunnelWarner) {
	for {
		conn, err := ln.Accept()
		if err != nil {
			return // listener closed on shutdown
		}
		go func() {
			defer conn.Close()
			if err := bridgeMicroVMConn(ctx, conn, wsURL, header, tracer); err != nil && ctx.Err() == nil {
				warner.report(err)
			}
		}()
	}
}

// microVMTunnelError is an {"error":...} frame sent instead of "connected".
type microVMTunnelError struct {
	code, message string
}

func (e *microVMTunnelError) Error() string {
	return e.code + ": " + e.message
}

// bridgeMicroVMConn opens one tunnel for an accepted local connection, waits
// for the server to confirm the guest port is open, then copies bytes.
func bridgeMicroVMConn(ctx context.Context, local net.Conn, wsURL string, header http.Header, tracer *log.Logger) error {
	ws, resp, err := websocket.DefaultDialer.DialContext(ctx, wsURL, header)
	body := microVMRejectionBody(resp)
	if tracer != nil {
		// gorilla dials its own connection, so doctl's HTTP trace never sees
		// the handshake. The token is never printed.
		status := "no response"
		if resp != nil {
			status = resp.Status
		}
		tracer.Println("->", "GET "+wsURL)
		tracer.Println("<-", status, collapseMicroVMLine(body, microVMRejectionMax))
	}
	if err != nil {
		if resp == nil {
			return fmt.Errorf("dialing tunnel: %w", err)
		}
		if msg := microVMRejectionMessage(body); msg != "" {
			return fmt.Errorf("server rejected tunnel (%s): %s", resp.Status, msg)
		}
		return fmt.Errorf("server rejected tunnel (%s): %w", resp.Status, err)
	}
	defer ws.Close()
	if err := awaitMicroVMTunnel(ws); err != nil {
		return err
	}
	return pumpMicroVMTunnel(local, ws)
}

// microVMRejectionBody reads what the server said when it refused the upgrade.
// gorilla keeps up to 1 KiB of a rejected handshake's body for this purpose.
func microVMRejectionBody(resp *http.Response) string {
	if resp == nil || resp.Body == nil {
		return ""
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(io.LimitReader(resp.Body, microVMRejectionBodyLimit))
	if err != nil {
		return ""
	}
	return string(b)
}

// microVMRejectionMessage takes the "message" field of an API error body, or
// the raw text on one line for anything else (a proxy's text or HTML page).
func microVMRejectionMessage(body string) string {
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
	return collapseMicroVMLine(body, microVMRejectionMax)
}

// collapseMicroVMLine squashes whitespace and caps the length.
func collapseMicroVMLine(s string, max int) string {
	s = strings.Join(strings.Fields(s), " ")
	if r := []rune(s); len(r) > max {
		return string(r[:max]) + "..."
	}
	return s
}

// awaitMicroVMTunnel reads control frames until the server reports the guest
// port as connected, or sends an error.
func awaitMicroVMTunnel(ws *websocket.Conn) error {
	_ = ws.SetReadDeadline(time.Now().Add(microVMTunnelOpenTimeout))
	defer ws.SetReadDeadline(time.Time{})
	for {
		mt, data, err := ws.ReadMessage()
		if err != nil {
			return err
		}
		if mt != websocket.TextMessage {
			return errors.New("server sent data before the tunnel opened")
		}
		ctrl, err := godo.ParseMicroVMConsoleControl(data)
		if err != nil {
			continue // a control frame this doctl does not know
		}
		if ctrl.Error != nil {
			return &microVMTunnelError{code: ctrl.Error.Code, message: ctrl.Error.Message}
		}
		if ctrl.Status != nil && ctrl.Status.State == godo.MicroVMPortForwardConnected {
			return nil
		}
	}
}

// pumpMicroVMTunnel copies bytes between local and an open tunnel until either
// side closes. Text frames after "connected" are skipped.
func pumpMicroVMTunnel(local net.Conn, ws *websocket.Conn) error {
	errCh := make(chan error, 2)

	// Tunnel → local. A normal close means the guest side hung up.
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
				errCh <- nil // the local reader went away; nothing to report
				return
			}
		}
	}()

	// Local → tunnel. This goroutine owns all data writes (gorilla allows one
	// concurrent writer), including the close frame when the local side ends.
	go func() {
		buf := make([]byte, microVMCopyBuf)
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

	err := <-errCh
	local.Close()
	ws.Close()
	return err
}

// explainMicroVMTunnelErr says what a failed tunnel means for the port the
// user asked for.
func explainMicroVMTunnelErr(err error, remotePort int) string {
	var te *microVMTunnelError
	if errors.As(err, &te) {
		if te.code == godo.MicroVMConsoleErrAgentError {
			return fmt.Sprintf("port %d: the MicroVM could not connect to that port (%s) — "+
				"start the process that listens on it, then retry", remotePort, te.message)
		}
		return fmt.Sprintf("port %d: the tunnel did not open (%s) — retry; "+
			"if it keeps happening, re-run with --trace", remotePort, te)
	}
	if websocket.IsCloseError(err, websocket.CloseAbnormalClosure) {
		return fmt.Sprintf("port %d: the tunnel dropped without closing cleanly — "+
			"check your network; new connections still work", remotePort)
	}
	return fmt.Sprintf("port %d: %v", remotePort, err)
}

// microVMTunnelWarner prints each distinct tunnel failure once per port. One
// page load opens several connections, so a stopped server would otherwise
// repeat the same warning for each of them.
type microVMTunnelWarner struct {
	remotePort int
	mu         sync.Mutex
	seen       map[string]int
}

func (w *microVMTunnelWarner) report(err error) {
	msg := explainMicroVMTunnelErr(err, w.remotePort)

	w.mu.Lock()
	n := w.seen[msg]
	w.seen[msg] = n + 1
	w.mu.Unlock()

	switch n {
	case 0:
		warn("%s", msg)
	case 1:
		warn("port %d: same failure again; further identical warnings are suppressed", w.remotePort)
	}
}

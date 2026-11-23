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
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/godo"
	"github.com/gorilla/websocket"
)

// doctl compute microvm port-forward: one local TCP listener per port pair.
// Each accepted connection opens its own WebSocket to
// GET /v2/microvms/{id}/port_forward?port=N. Unlike the sandbox tunnel, the
// server upgrades first and then sends a text frame: {"status":{"state":
// "connected"}} when the guest port is open, or {"error":...} followed by a
// normal close. The first frame therefore decides whether the tunnel opened;
// without reading it, a refused port looks like an empty reply.

// microVMAgentPort is the guest port of the MicroVM agent. The API refuses to
// forward it; doctl checks first so the error comes before a listener opens.
const microVMAgentPort = 50051

var (
	// microVMResumeTimeout bounds the wait for a paused MicroVM to run.
	microVMResumeTimeout = 2 * time.Minute
	// microVMPollInterval is the delay between state checks while resuming.
	microVMPollInterval = 2 * time.Second
	// microVMTunnelOpenTimeout bounds the wait for the first control frame.
	microVMTunnelOpenTimeout = time.Minute
)

// RunMicroVMPortForward opens local TCP tunnels to ports inside a MicroVM and
// blocks until interrupted.
func RunMicroVMPortForward(c *CmdConfig) error {
	if len(c.Args) < 2 {
		return doctl.NewMissingArgsErr(c.NS)
	}
	id := c.Args[0]
	pairs := make([]forwardPair, 0, len(c.Args)-1)
	for _, arg := range c.Args[1:] {
		p, err := parsePortPair(arg, 1)
		if err != nil {
			return err
		}
		if p.remote == microVMAgentPort {
			return fmt.Errorf("remote port %d is reserved for the MicroVM agent and cannot be forwarded", microVMAgentPort)
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
	tracer := newWSTracer()

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

		warner := newTunnelWarner(pair.remote)
		warner.explain = explainMicroVMTunnelErr
		bridge := func(ctx context.Context, conn net.Conn) error {
			return bridgeMicroVMConn(ctx, conn, wsURL, header, tracer)
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			acceptLoop(ctx, ln, bridge, warner)
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

// microVMTunnelError is an {"error":...} frame sent instead of "connected".
type microVMTunnelError struct {
	code, message string
}

func (e *microVMTunnelError) Error() string {
	return e.code + ": " + e.message
}

// bridgeMicroVMConn opens one tunnel for an accepted local connection, waits
// for the server to confirm the guest port is open, then copies bytes.
func bridgeMicroVMConn(ctx context.Context, local net.Conn, wsURL string, header http.Header, tracer *wsTracer) error {
	ws, resp, err := websocket.DefaultDialer.DialContext(ctx, wsURL, header)
	body := handshakeBody(resp)
	tracer.handshake(wsURL, header, resp, body)
	if err != nil {
		return serverRejection(resp, body, err)
	}
	defer ws.Close()
	if err := awaitMicroVMTunnel(ws); err != nil {
		return err
	}
	return pumpTunnel(local, ws)
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

// explainMicroVMTunnelErr says what a failed tunnel means for the port the
// user asked for.
func explainMicroVMTunnelErr(err error, remotePort int) string {
	var te *microVMTunnelError
	if !errors.As(err, &te) {
		return explainTunnelClose(err, remotePort)
	}
	if te.code == godo.MicroVMConsoleErrAgentError {
		return fmt.Sprintf("port %d: the MicroVM could not connect to that port (%s) — "+
			"start the process that listens on it, then retry", remotePort, te.message)
	}
	return fmt.Sprintf("port %d: the tunnel did not open (%s) — retry; "+
		"if it keeps happening, re-run with --trace", remotePort, te)
}

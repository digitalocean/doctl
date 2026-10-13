package agentproxy

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"
)

// ServeHTTP binds a plain HTTP listener on 127.0.0.1:port — the same
// loopback-only rule as Serve — and serves handler until ctx is canceled.
//
// This is the transport for facades whose native protocol is REST + SSE
// (opencode) rather than JSON-RPC over a WebSocket (codex). There is no
// framing to pump and no request/reply correlation to own at this layer, so
// unlike Serve there is no Facade/Notifier machinery here: the facade IS the
// http.Handler, and anything connection-scoped (like a single-consumer event
// stream) is the facade's own concern, not the transport's.
func ServeHTTP(ctx context.Context, port int, handler http.Handler) error {
	ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return err
	}
	return ServeHTTPListener(ctx, ln, handler)
}

// ServeHTTPListener is ServeHTTP over an already-bound listener. It exists
// for the same reason ServeListener does: tests that pick an ephemeral port
// with net.Listen and must not race a separate re-bind of that port number.
func ServeHTTPListener(ctx context.Context, ln net.Listener, handler http.Handler) error {
	server := &http.Server{
		Handler: handler,
		// Root every request context in ctx: a long-lived SSE handler watches
		// r.Context() to know when to stop, and Shutdown below only returns
		// once handlers do. Without this, canceling ctx would leave the SSE
		// handler blocked forever on a Background()-rooted request context and
		// shutdown would hang until the 2s timeout force-closed it — the same
		// class of bug ServeListener's connCtx comment records for WebSocket.
		BaseContext: func(net.Listener) context.Context { return ctx },
	}

	errCh := make(chan error, 1)
	go func() { errCh <- server.Serve(ln) }()

	select {
	case err := <-errCh:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		return server.Shutdown(shutdownCtx)
	}
}

package agentproxy

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServeHTTPListenerServesAndShutsDown(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- ServeHTTPListener(ctx, ln, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprint(w, "ok")
		}))
	}()

	resp, err := http.Get("http://" + ln.Addr().String() + "/")
	require.NoError(t, err)
	resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	cancel()
	select {
	case err := <-done:
		assert.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("ServeHTTPListener did not return after ctx cancel")
	}
}

// A handler blocked on r.Context() (the SSE shape) must be released by
// canceling the server ctx — that's what BaseContext is for; without it,
// Shutdown would hang until its timeout force-closed the connection.
func TestServeHTTPListenerCancelReleasesStreamingHandler(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	entered := make(chan struct{})
	done := make(chan error, 1)
	go func() {
		done <- ServeHTTPListener(ctx, ln, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.(http.Flusher).Flush()
			close(entered)
			<-r.Context().Done()
		}))
	}()

	resp, err := http.Get("http://" + ln.Addr().String() + "/stream")
	require.NoError(t, err)
	defer resp.Body.Close()
	<-entered

	cancel()
	select {
	case err := <-done:
		assert.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("streaming handler was not released by ctx cancel")
	}
}

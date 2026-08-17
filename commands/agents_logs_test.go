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
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strconv"
	"sync"
	"testing"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// Tests for `doctl agents logs` history paging. One replay request only covers
// the newest window of a long transcript, so logs walks backwards through the
// rest with `before`, one bounded page per request, and stitches the pages back
// into chronological order.

// logsHistoryServer serves GET .../stream the way harness-api does: a
// cursorless replay_only read returns only the newest `budget` events, and
// before=<event id> returns one page of the events older than it, ending with
// a `: has_more=` comment.
//
// Every event is a token chunk carrying its own id as text, so the rendered
// transcript spells out the order the events reached the client in.
type logsHistoryServer struct {
	total  int
	budget int

	mu       sync.Mutex
	requests []url.Values
}

func (s *logsHistoryServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	s.mu.Lock()
	s.requests = append(s.requests, q)
	s.mu.Unlock()

	w.Header().Set("Content-Type", "text/event-stream")
	w.WriteHeader(http.StatusOK)

	before := q.Get("before")
	if before == "" {
		start := 0
		if s.budget > 0 && s.total > s.budget {
			start = s.total - s.budget
		}
		for i := start; i < s.total; i++ {
			fmt.Fprint(w, logsHistoryFrame(i))
		}
		return
	}

	end, err := strconv.Atoi(before[len("evt-"):])
	if err != nil {
		http.Error(w, "unknown before cursor", http.StatusBadRequest)
		return
	}
	limit := 200
	if raw := q.Get("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n <= 0 {
			http.Error(w, "limit must be a positive integer", http.StatusBadRequest)
			return
		}
		limit = n
	}
	start := max(end-limit, 0)
	for i := start; i < end; i++ {
		fmt.Fprint(w, logsHistoryFrame(i))
	}
	fmt.Fprintf(w, ": has_more=%t\n\n", start > 0)
}

func logsHistoryFrame(i int) string {
	id := "evt-" + strconv.Itoa(i)
	return fmt.Sprintf("id: %s\ndata: {\"event_id\":%q,\"type\":%q,\"data\":{\"text\":\"%s \"}}\n\n",
		id, id, godo.HostedAgentEventKindTokenChunk, id)
}

func (s *logsHistoryServer) recorded() []url.Values {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.requests
}

// serveLogsHistory points the mocked service's StreamSession at a real godo
// client talking to srv, so query encoding, SSE parsing and has_more parsing
// all run for real rather than being stubbed out.
func serveLogsHistory(t *testing.T, tm *tcMocks, total, budget int) *logsHistoryServer {
	t.Helper()
	history := &logsHistoryServer{total: total, budget: budget}
	srv := httptest.NewServer(history)
	t.Cleanup(srv.Close)

	client, err := godo.New(nil, godo.SetBaseURL(srv.URL+"/"))
	require.NoError(t, err)

	tm.hostedAgents.EXPECT().
		StreamSession(gomock.Any(), "sess_x", gomock.Any()).
		DoAndReturn(func(ctx context.Context, sessionID string, opt *godo.HostedAgentSessionStreamOptions) (*godo.HostedAgentSessionStream, error) {
			stream, _, err := client.HostedAgents.StreamSession(ctx, sessionID, opt)
			return stream, err
		}).
		AnyTimes()
	return history
}

// captureStderr redirects os.Stderr for the duration of fn and returns what it
// received — the paging cursor hint goes there, keeping stdout clean.
func captureStderr(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stderr
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stderr = w
	defer func() { os.Stderr = old }()

	fn()

	require.NoError(t, w.Close())
	os.Stderr = old
	out, err := io.ReadAll(r)
	require.NoError(t, err)
	return string(out)
}

// A history longer than one replay window is printed whole and in order: the
// cursorless read covers the newest events and each page back picks up from
// the previous page's oldest event.
func TestRunAgentsLogs_WalksHistoryBackwards(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		history := serveLogsHistory(t, tm, 8, 3)

		var buf bytes.Buffer
		config.Out = &buf
		config.Args = []string{"sess_x"}
		config.Doit.Set(config.NS, doctl.ArgAgentPageSize, 3)

		require.NoError(t, RunAgentsLogs(config))
		assert.Contains(t, buf.String(), "evt-0 evt-1 evt-2 evt-3 evt-4 evt-5 evt-6 evt-7")

		reqs := history.recorded()
		require.Len(t, reqs, 3)
		assert.Empty(t, reqs[0].Get("before"))
		assert.Equal(t, "true", reqs[0].Get("replay_only"))
		assert.Equal(t, "evt-5", reqs[1].Get("before"))
		assert.Equal(t, "3", reqs[1].Get("limit"))
		assert.Equal(t, "evt-2", reqs[2].Get("before"))
	})
}

// --tail prints only the newest events, and stops the walk as soon as it has
// them rather than reading the whole transcript first.
func TestRunAgentsLogs_TailStopsEarly(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		history := serveLogsHistory(t, tm, 8, 3)

		var buf bytes.Buffer
		config.Out = &buf
		config.Args = []string{"sess_x"}
		config.Doit.Set(config.NS, doctl.ArgAgentLogsTail, 2)

		require.NoError(t, RunAgentsLogs(config))
		out := buf.String()
		assert.Contains(t, out, "evt-6 evt-7")
		assert.NotContains(t, out, "evt-5")
		assert.Len(t, history.recorded(), 1)
	})
}

// --before reads exactly one page of older events, no walk, and reports the
// cursor for the page before it on stderr.
func TestRunAgentsLogs_BeforeReadsOnePage(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		history := serveLogsHistory(t, tm, 8, 3)

		var buf bytes.Buffer
		config.Out = &buf
		config.Args = []string{"sess_x"}
		config.Doit.Set(config.NS, doctl.ArgAgentLogsBefore, "evt-5")
		config.Doit.Set(config.NS, doctl.ArgAgentPageSize, 2)

		var runErr error
		stderr := captureStderr(t, func() { runErr = RunAgentsLogs(config) })
		require.NoError(t, runErr)

		out := buf.String()
		assert.Contains(t, out, "evt-3 evt-4")
		assert.NotContains(t, out, "evt-2")
		assert.NotContains(t, out, "evt-5")
		assert.Contains(t, stderr, "--before evt-3")

		reqs := history.recorded()
		require.Len(t, reqs, 1)
		assert.Equal(t, "evt-5", reqs[0].Get("before"))
		assert.Equal(t, "2", reqs[0].Get("limit"))
		assert.Equal(t, "true", reqs[0].Get("replay_only"))
	})
}

// The oldest page says so, instead of handing back a cursor that would return
// nothing.
func TestRunAgentsLogs_BeforeReachesBeginning(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		serveLogsHistory(t, tm, 8, 3)

		var buf bytes.Buffer
		config.Out = &buf
		config.Args = []string{"sess_x"}
		config.Doit.Set(config.NS, doctl.ArgAgentLogsBefore, "evt-2")
		config.Doit.Set(config.NS, doctl.ArgAgentPageSize, 5)

		var runErr error
		stderr := captureStderr(t, func() { runErr = RunAgentsLogs(config) })
		require.NoError(t, runErr)

		assert.Contains(t, buf.String(), "evt-0 evt-1")
		assert.Contains(t, stderr, "Reached the beginning")
		assert.NotContains(t, stderr, "--before")
	})
}

// Negative sizes are rejected before anything is streamed: they have no
// meaning to pass on to the server.
func TestRunAgentsLogs_RejectsNegativeSizes(t *testing.T) {
	for _, flag := range []string{doctl.ArgAgentLogsTail, doctl.ArgAgentPageSize} {
		t.Run(flag, func(t *testing.T) {
			withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
				// No StreamSession expectation: nothing may be streamed.
				config.Args = []string{"sess_x"}
				config.Doit.Set(config.NS, flag, -1)

				require.Error(t, RunAgentsLogs(config))
			})
		})
	}
}

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

package do_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// historyFake serves GET /v2/agents/sessions/{id}/stream the way harness-api
// does once replay is bounded: a cursorless replay_only read returns only the
// newest `budget` events, and before=<event id> returns one page of the events
// older than it, oldest-first, ending with a `: has_more=` comment.
//
// Event ids are "evt-<position in history>", so a cursor is just an index.
type historyFake struct {
	total  int // history length; ids evt-0 (oldest) .. evt-(total-1) (newest)
	budget int // cursorless replay window; 0 serves everything

	// controlFrame prepends an id-less stream.state frame to every response,
	// the transport control frame a real stream carries.
	controlFrame bool

	// stuck serves the same single event with has_more=true for every page,
	// however far back the cursor claims to be — a server whose cursor never
	// advances, which a backward walk must not follow forever.
	stuck bool

	// emptyPage serves pages with no events at all but has_more=true.
	emptyPage bool

	// pageStatus, when set, is returned instead of a page (cursorless reads
	// still succeed).
	pageStatus int

	mu       sync.Mutex
	requests []url.Values
}

func (f *historyFake) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	f.mu.Lock()
	f.requests = append(f.requests, q)
	requests := len(f.requests)
	f.mu.Unlock()

	before := q.Get("before")
	if before != "" && f.pageStatus != 0 {
		http.Error(w, "history page unavailable", f.pageStatus)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.WriteHeader(http.StatusOK)
	if f.controlFrame {
		fmt.Fprint(w, "data: {\"type\":\"stream.state\",\"data\":{\"state\":\"live\"}}\n\n")
	}

	if before == "" {
		start := 0
		if f.budget > 0 && f.total > f.budget {
			start = f.total - f.budget
		}
		for i := start; i < f.total; i++ {
			fmt.Fprint(w, historyFrame(i))
		}
		return
	}

	if f.stuck {
		// Always the same event, always more behind it. Capped so a walk that
		// failed to notice would fail its assertions rather than hang.
		if requests < 20 {
			fmt.Fprint(w, historyFrame(0))
			fmt.Fprint(w, ": has_more=true\n\n")
		}
		return
	}
	if f.emptyPage {
		fmt.Fprint(w, ": has_more=true\n\n")
		return
	}

	end, ok := historyIndex(before)
	if !ok || end < 0 || end > f.total {
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
		fmt.Fprint(w, historyFrame(i))
	}
	fmt.Fprintf(w, ": has_more=%t\n\n", start > 0)
}

func historyFrame(i int) string {
	id := "evt-" + strconv.Itoa(i)
	return fmt.Sprintf("id: %s\ndata: {\"event_id\":%q,\"seq\":%d,\"type\":\"run.log\",\"data\":{}}\n\n", id, id, i)
}

func historyIndex(eventID string) (int, bool) {
	raw, ok := strings.CutPrefix(eventID, "evt-")
	if !ok {
		return 0, false
	}
	i, err := strconv.Atoi(raw)
	return i, err == nil
}

// newHistoryService points a real HostedAgentsService at the fake, so the whole
// stack under test (godo query encoding, SSE parsing, has_more parsing) runs
// for real.
func newHistoryService(t *testing.T, fake *historyFake) do.HostedAgentsService {
	t.Helper()
	srv := httptest.NewServer(fake)
	t.Cleanup(srv.Close)

	client, err := godo.New(nil, godo.SetBaseURL(srv.URL+"/"))
	require.NoError(t, err)
	return do.NewHostedAgentsService(client)
}

// eventIDs is the readable form of a history result for assertions.
func eventIDs(events []godo.HostedAgentEvent) []string {
	out := make([]string, len(events))
	for i := range events {
		out[i] = events[i].EventID
	}
	return out
}

func (f *historyFake) recorded() []url.Values {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.requests
}

// A history longer than one replay window is stitched back together in
// chronological order: the cursorless read covers the newest events, and each
// backward page picks up where the previous one's oldest event left off.
func TestLoadSessionHistory_WalksBackwardsToTheBeginning(t *testing.T) {
	fake := &historyFake{total: 25, budget: 10}
	svc := newHistoryService(t, fake)

	history, err := do.LoadSessionHistory(context.Background(), svc, "sess_x", do.SessionHistoryOptions{PageSize: 10})
	require.NoError(t, err)

	want := make([]string, 25)
	for i := range want {
		want[i] = "evt-" + strconv.Itoa(i)
	}
	assert.Equal(t, want, eventIDs(history))

	// One cursorless read, then a page per remaining chunk, each cursored on
	// the previous chunk's oldest event, stopping on the page that reports
	// nothing older.
	reqs := fake.recorded()
	require.Len(t, reqs, 3)
	assert.Empty(t, reqs[0].Get("before"))
	assert.Equal(t, "true", reqs[0].Get("replay_only"))
	assert.Equal(t, "evt-15", reqs[1].Get("before"))
	assert.Equal(t, "10", reqs[1].Get("limit"))
	assert.Equal(t, "true", reqs[1].Get("replay_only"))
	assert.Equal(t, "evt-5", reqs[2].Get("before"))
}

// A cursorless read that already covers the whole history needs no pages: the
// first page back reports nothing older.
func TestLoadSessionHistory_ShortHistoryNeedsOnePage(t *testing.T) {
	fake := &historyFake{total: 3}
	svc := newHistoryService(t, fake)

	history, err := do.LoadSessionHistory(context.Background(), svc, "sess_x", do.SessionHistoryOptions{})
	require.NoError(t, err)
	assert.Equal(t, []string{"evt-0", "evt-1", "evt-2"}, eventIDs(history))

	// The walk still asks once, since a cursorless read carries no has_more to
	// say it was complete, and stops on that answer.
	reqs := fake.recorded()
	require.Len(t, reqs, 2)
	assert.Equal(t, "evt-0", reqs[1].Get("before"))
	assert.Equal(t, strconv.Itoa(do.DefaultHistoryPageSize), reqs[1].Get("limit"))
}

// MaxEvents keeps the newest events and stops the walk early, asking the last
// page only for what's still missing rather than a full page.
func TestLoadSessionHistory_StopsAtMaxEvents(t *testing.T) {
	fake := &historyFake{total: 25, budget: 10}
	svc := newHistoryService(t, fake)

	history, err := do.LoadSessionHistory(context.Background(), svc, "sess_x", do.SessionHistoryOptions{
		PageSize:  10,
		MaxEvents: 13,
	})
	require.NoError(t, err)

	want := make([]string, 0, 13)
	for i := 12; i < 25; i++ {
		want = append(want, "evt-"+strconv.Itoa(i))
	}
	assert.Equal(t, want, eventIDs(history))

	reqs := fake.recorded()
	require.Len(t, reqs, 2)
	assert.Equal(t, "3", reqs[1].Get("limit"))
}

// MaxEvents satisfied by the cursorless read alone means no paging at all, and
// the newest events are the ones kept.
func TestLoadSessionHistory_MaxEventsWithinReplayWindow(t *testing.T) {
	fake := &historyFake{total: 25, budget: 10}
	svc := newHistoryService(t, fake)

	history, err := do.LoadSessionHistory(context.Background(), svc, "sess_x", do.SessionHistoryOptions{MaxEvents: 4})
	require.NoError(t, err)
	assert.Equal(t, []string{"evt-21", "evt-22", "evt-23", "evt-24"}, eventIDs(history))
	assert.Len(t, fake.recorded(), 1)
}

// An empty history yields nothing and nothing to page back from.
func TestLoadSessionHistory_EmptyHistory(t *testing.T) {
	fake := &historyFake{}
	svc := newHistoryService(t, fake)

	history, err := do.LoadSessionHistory(context.Background(), svc, "sess_x", do.SessionHistoryOptions{})
	require.NoError(t, err)
	assert.Empty(t, history)
	assert.Len(t, fake.recorded(), 1)
}

// Control frames describe the transport, not the session: they are left out of
// history, and their missing event id never becomes a paging cursor.
func TestLoadSessionHistory_DropsControlFrames(t *testing.T) {
	fake := &historyFake{total: 6, budget: 3, controlFrame: true}
	svc := newHistoryService(t, fake)

	history, err := do.LoadSessionHistory(context.Background(), svc, "sess_x", do.SessionHistoryOptions{PageSize: 3})
	require.NoError(t, err)
	assert.Equal(t, []string{"evt-0", "evt-1", "evt-2", "evt-3", "evt-4", "evt-5"}, eventIDs(history))

	reqs := fake.recorded()
	require.Len(t, reqs, 2)
	assert.Equal(t, "evt-3", reqs[1].Get("before"))
}

// A server whose pages never move the cursor backwards would otherwise be
// re-requested forever; the walk stops instead of looping.
func TestLoadSessionHistory_StopsWhenCursorDoesNotAdvance(t *testing.T) {
	fake := &historyFake{total: 4, budget: 2, stuck: true}
	svc := newHistoryService(t, fake)

	history, err := do.LoadSessionHistory(context.Background(), svc, "sess_x", do.SessionHistoryOptions{PageSize: 2})
	require.NoError(t, err)
	// The repeated page is taken once, then the walk stops instead of taking
	// it again.
	assert.Equal(t, []string{"evt-0", "evt-2", "evt-3"}, eventIDs(history))
	assert.Len(t, fake.recorded(), 3)
}

// A page with no events has nothing to continue from, whatever it claims about
// older history.
func TestLoadSessionHistory_StopsOnEmptyPage(t *testing.T) {
	fake := &historyFake{total: 4, budget: 2, emptyPage: true}
	svc := newHistoryService(t, fake)

	history, err := do.LoadSessionHistory(context.Background(), svc, "sess_x", do.SessionHistoryOptions{PageSize: 2})
	require.NoError(t, err)
	assert.Equal(t, []string{"evt-2", "evt-3"}, eventIDs(history))
	assert.Len(t, fake.recorded(), 2)
}

// A failing page fails the whole walk: a partial history silently presented as
// complete would be worse than an error.
func TestLoadSessionHistory_PageErrorFailsTheWalk(t *testing.T) {
	fake := &historyFake{total: 25, budget: 10, pageStatus: http.StatusInternalServerError}
	svc := newHistoryService(t, fake)

	history, err := do.LoadSessionHistory(context.Background(), svc, "sess_x", do.SessionHistoryOptions{PageSize: 10})
	require.Error(t, err)
	assert.Nil(t, history)
}

// LoadSessionHistoryPage reads exactly one page and reports whether older
// events remain behind it.
func TestLoadSessionHistoryPage(t *testing.T) {
	fake := &historyFake{total: 25}
	svc := newHistoryService(t, fake)

	page, hasMore, err := do.LoadSessionHistoryPage(context.Background(), svc, "sess_x", "evt-10", 4)
	require.NoError(t, err)
	assert.Equal(t, []string{"evt-6", "evt-7", "evt-8", "evt-9"}, eventIDs(page))
	assert.True(t, hasMore)

	page, hasMore, err = do.LoadSessionHistoryPage(context.Background(), svc, "sess_x", "evt-4", 10)
	require.NoError(t, err)
	assert.Equal(t, []string{"evt-0", "evt-1", "evt-2", "evt-3"}, eventIDs(page))
	assert.False(t, hasMore)

	reqs := fake.recorded()
	require.Len(t, reqs, 2)
	assert.Equal(t, "4", reqs[0].Get("limit"))
	assert.Equal(t, "true", reqs[0].Get("replay_only"))
}

// A page needs a cursor; without one there is no page to ask for.
func TestLoadSessionHistoryPage_RequiresBefore(t *testing.T) {
	fake := &historyFake{total: 3}
	svc := newHistoryService(t, fake)

	_, _, err := do.LoadSessionHistoryPage(context.Background(), svc, "sess_x", "", 10)
	require.Error(t, err)
	assert.Empty(t, fake.recorded())
}

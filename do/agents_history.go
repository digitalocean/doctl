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

package do

import (
	"context"
	"errors"

	"github.com/digitalocean/godo"
)

// DefaultHistoryPageSize is the events-per-request size used when
// SessionHistoryOptions leaves PageSize unset. Matches the server's own
// default page size for a history request.
const DefaultHistoryPageSize = 200

// hostedAgentEventKindStreamState is the connection-health control frame the
// stream carries. It reports transport state, not session activity, and is
// never part of a session's durable history.
//
// Declared here rather than imported for the same reason commands/agents.go
// declares its own copy: godo does not export a constant for it on the pins
// doctl builds against. Wire value matches the published godo API.
const hostedAgentEventKindStreamState = godo.HostedAgentEventKind("stream.state")

// SessionHistoryOptions tunes a LoadSessionHistory walk.
type SessionHistoryOptions struct {
	// PageSize is the per-request event limit for the backward walk.
	// Defaults to DefaultHistoryPageSize.
	PageSize int

	// MaxEvents stops the walk once this many events have been collected,
	// keeping the newest ones. Zero walks back to the beginning of history.
	MaxEvents int
}

// LoadSessionHistory returns a session's durable events in chronological
// order.
//
// A cursorless replay only covers the newest window of a session's history
// (the server's replay budget), so reading the whole thing means walking
// backwards from that window's oldest event, one bounded page per request,
// until the server reports no older events remain. Pages arrive newest-group
// first and are stitched back into chronological order here.
//
// Control frames are dropped: they describe the transport, not the session.
func LoadSessionHistory(ctx context.Context, svc HostedAgentsService, sessionID string, opt SessionHistoryOptions) ([]godo.HostedAgentEvent, error) {
	pageSize := opt.PageSize
	if pageSize <= 0 {
		pageSize = DefaultHistoryPageSize
	}

	// No limit on the cursorless read: its size is the server's replay budget,
	// not something the caller gets to pick. Anything past MaxEvents is
	// trimmed below.
	head, cursor, _, err := readHistoryPage(ctx, svc, sessionID, "", 0)
	if err != nil {
		return nil, err
	}

	// Older pages, newest group first; concatenated in reverse at the end so
	// a long walk doesn't re-copy the whole history once per page.
	var older [][]godo.HostedAgentEvent
	total := len(head)

	// An empty cursor means the head window held nothing to page back from,
	// so there is nothing to walk.
	for cursor != "" {
		limit := pageSize
		if opt.MaxEvents > 0 {
			remaining := opt.MaxEvents - total
			if remaining <= 0 {
				break
			}
			if remaining < limit {
				limit = remaining
			}
		}

		page, oldest, hasMore, err := readHistoryPage(ctx, svc, sessionID, cursor, limit)
		if err != nil {
			return nil, err
		}
		// A page whose oldest event is the cursor itself is the same page
		// again, so it is dropped rather than appended twice.
		if len(page) > 0 && oldest != cursor {
			older = append(older, page)
			total += len(page)
		}
		// Stop on the oldest page, on a page that yielded nothing to continue
		// from, and on a cursor that failed to advance — the last two would
		// otherwise re-request the same page forever.
		if !hasMore || oldest == "" || oldest == cursor {
			break
		}
		cursor = oldest
	}

	history := make([]godo.HostedAgentEvent, 0, total)
	for i := len(older) - 1; i >= 0; i-- {
		history = append(history, older[i]...)
	}
	history = append(history, head...)

	// The last page can overshoot MaxEvents; keep the newest events.
	if opt.MaxEvents > 0 && len(history) > opt.MaxEvents {
		history = history[len(history)-opt.MaxEvents:]
	}
	return history, nil
}

// LoadSessionHistoryPage reads the single page of durable events strictly
// older than before, chronologically ordered, and reports whether older events
// remain behind it. Use the returned page's first event id as the next before.
//
// This is one step of what LoadSessionHistory automates, for callers that want
// to step through history themselves. before is required.
func LoadSessionHistoryPage(ctx context.Context, svc HostedAgentsService, sessionID, before string, limit int) ([]godo.HostedAgentEvent, bool, error) {
	if before == "" {
		return nil, false, errors.New("hosted agents: before is required to read a history page")
	}
	if limit <= 0 {
		limit = DefaultHistoryPageSize
	}
	events, _, hasMore, err := readHistoryPage(ctx, svc, sessionID, before, limit)
	if err != nil {
		return nil, false, err
	}
	return events, hasMore, nil
}

// readHistoryPage drains one replay-only stream and returns its events in
// chronological order, the id of its oldest event (the cursor for the next
// page back, empty when the page carried none), and whether older events
// remain.
//
// An empty before reads the newest window instead of a page: no cursor, no
// limit, and no has_more trailer, so hasMore is false and the caller relies
// on the returned cursor to decide whether to keep walking.
func readHistoryPage(ctx context.Context, svc HostedAgentsService, sessionID, before string, limit int) (events []godo.HostedAgentEvent, oldest string, hasMore bool, err error) {
	opt := &godo.HostedAgentSessionStreamOptions{ReplayOnly: true}
	if before != "" {
		opt.Before = before
		opt.Limit = limit
	}

	stream, err := svc.StreamSession(ctx, sessionID, opt)
	if err != nil {
		return nil, "", false, err
	}
	defer stream.Close()

	for stream.Next() {
		ev := stream.Current()
		if ev.Kind == hostedAgentEventKindStreamState {
			continue
		}
		if oldest == "" {
			oldest = ev.EventID
		}
		events = append(events, ev)
	}
	if err := stream.Err(); err != nil {
		return nil, "", false, err
	}
	// The has_more trailer arrives after the last event, so this is only
	// meaningful now that the page has been drained to its end.
	return events, oldest, stream.HasMore(), nil
}

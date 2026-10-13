// Package codex: --replay support, feeding a session's durable event history
// into the first thread this facade bootstraps.
package codex

import (
	"context"
	"log"

	"github.com/digitalocean/doctl/internal/agentproxy"
	"github.com/digitalocean/godo"
)

var _ agentproxy.AfterReply = (*Facade)(nil)

// maybeReplay marks that this connection wants session history fed into the
// thread once thread/start or thread/resume has been acknowledged. The actual
// fetch is started from AfterReply (not here) so a replayed turn/started
// cannot race onto the wire before the thread/start|resume reply that creates
// the thread it refers to. A no-op when Replay is false, when a completed
// replay has already run for this Facade (replayDone), when one is already
// in flight (replaying), or when one is already pending AfterReply.
//
// The gate is per-Facade / per-connection, not process-wide: ServeListener
// builds a fresh Facade for every accepted WebSocket, and a reconnecting
// client is a new TUI with empty scrollback that needs history again under
// --replay. Within one connection, thread/start then thread/resume must not
// re-deliver the same history a second time.
func (f *Facade) maybeReplay(ctx context.Context) {
	if !f.Replay {
		return
	}
	f.replayMu.Lock()
	defer f.replayMu.Unlock()
	if f.replayDone || f.replaying || f.replayPending {
		return
	}
	f.replayPending = true
}

// AfterReply implements agentproxy.AfterReply: once handleConn has written
// a successful thread/start or thread/resume result, start the deferred
// replaySessionHistory goroutine if maybeReplay armed one.
func (f *Facade) AfterReply(ctx context.Context, method string) {
	if method != "thread/start" && method != "thread/resume" {
		return
	}
	f.replayMu.Lock()
	if !f.replayPending || f.replayDone || f.replaying {
		f.replayMu.Unlock()
		return
	}
	f.replayPending = false
	f.replaying = true
	f.replayMu.Unlock()

	go f.replaySessionHistory(ctx)
}

// replaySessionHistory fetches this session's full durable event history via
// a one-shot replay-only StreamSession call — distinct from, and unrelated
// to, the single live StreamSession call runEventLoop owns for the whole
// connection (see the Facade.mu doc comment for why there's only ever one of
// those; a replay-only stream reads to completion and closes on its own, per
// harness-api's handleStreamSession, so it never overlaps with or displaces
// the live one). Each historical event is fed through the same translateEvent
// used for live events, so past turns arrive as an ordinary sequence of
// turn/item notifications — the codex protocol has no separate "history"
// shape to send instead.
//
// turns is local to this call, not f.turns: replay's synthesized turnStates
// are never registered in the shared map at all. Two reasons. First,
// runEventLoop's own exit path unconditionally does f.turns = nil (see its
// doc comment) — sharing the map meant a live turn finishing and its event
// loop exiting while a large replay was still mid-flight would silently wipe
// replay's in-progress turnStates out from under it, losing accumulated
// message text and leaving a turn's item/completed never sent. Second,
// nothing else ever needs to look replay's historical run ids up by id:
// finishTurn's delete(f.turns, runID) is a harmless no-op for an id that was
// never inserted, and HITL handling for a replayed request now takes ts
// directly rather than re-deriving it via f.turns (see
// autoRejectFileChangeApproval).
//
// A run still in progress at the moment history was fetched (no
// run.completed/run.failed among its events yet) is adopted into the live
// path once the replay completes — see adoptInFlightReplayTurns. This is
// what makes a proxy killed and restarted mid-turn (with --replay)
// reconstruct the TUI without desync: history rebuilds the turn up to the
// fetch point, and the live loop — attached at the replay's own cursor, so
// nothing is lost in between and nothing is double-delivered — finishes it.
//
// Marks replayDone only on reaching the natural end of the stream — not on
// any early-abort path (StreamSession failing to open, or a dead client
// mid-fetch) — so an aborted attempt is retried on the next thread/start or
// thread/resume on this same connection instead of permanently foreclosing
// --replay for the rest of the connection on one flaky fetch.
func (f *Facade) replaySessionHistory(ctx context.Context) {
	completed := false
	defer func() {
		f.replayMu.Lock()
		f.replaying = false
		if completed {
			f.replayDone = true
		}
		f.replayMu.Unlock()
	}()

	// IncludeRaw on replay too: a raw-eligible session's history renders with
	// the same fidelity as its live tail (the server may or may not retain
	// raw bytes durably — absent bytes just mean canonical fallback per
	// event, same as live).
	stream, err := f.Sessions.StreamSession(ctx, f.SessionID, &godo.HostedAgentSessionStreamOptions{
		ReplayOnly: true,
		IncludeRaw: f.rawEligible(),
	})
	if err != nil {
		log.Printf("codex facade: replay history fetch failed, will retry on next connect: %v", err)
		return
	}
	defer stream.Close()

	turns := make(map[string]*turnState)
	finished := make(map[string]bool)
	// lastEventID tracks the replay's own resume cursor: the id of the last
	// event the stream delivered, run-less control frames included — cursor
	// position is a property of the stream, not of any one turn.
	var lastEventID string
	for stream.Next() {
		ev := stream.Current()

		if ev.EventID != "" {
			lastEventID = ev.EventID
		}

		// Same guard as drainStream: control frames (stream.state) and other
		// run-less events belong to no turn — don't synthesize a phantom
		// turnState keyed "" with itemID "-msg".
		if ev.RunID == "" {
			continue
		}

		ts, ok := turns[ev.RunID]
		if !ok {
			ts = &turnState{itemID: ev.RunID + "-msg"}
			turns[ev.RunID] = ts
		}

		switch ev.Kind {
		case godo.HostedAgentEventKindRunCompleted, godo.HostedAgentEventKindRunFailed:
			finished[ev.RunID] = true
		}

		if f.translateEvent(ctx, ev, ts, true) {
			log.Printf("codex facade: replay stopped early (client disconnected), will retry on next connect")
			return
		}
	}
	if err := stream.Err(); err != nil {
		log.Printf("codex facade: replay history stream ended with error, will retry on next connect: %v", err)
		return
	}
	completed = true

	f.adoptInFlightReplayTurns(ctx, turns, finished, lastEventID)
}

// adoptInFlightReplayTurns hands any run the replay saw start but never saw
// end over to the live event loop, so a turn that was mid-flight when this
// proxy (re)started keeps streaming instead of sitting "inProgress" forever.
// The replayed turnState is registered as-is — accumulated message text,
// itemStarted, in-flight tool calls — so the continuation appends to exactly
// the state the client was just shown, and the live stream is opened at the
// replay's own last event id so the continuation starts strictly after what
// history already delivered (no gap, no double delivery). Only the durable
// suffix arrives raw-less (the server doesn't retain source_raw); the
// canonical fallback plus translateEvent's late item/started keep the
// client-visible item lifecycle consistent either way.
//
// The session's parent run (run id == SessionID) is never adopted: OHR
// establishes it as the session-wide multi-turn run, so it has no terminal
// event while the session lives — adopting it would hold the live loop open
// (and reconnecting) forever on a "turn" that can never complete.
//
// The cursor is only seeded when no live loop is running and none has left a
// cursor behind (the normal bootstrap order: thread/start's replay finishes
// before the first turn/start). If a live turn already raced ahead of the
// replay, its own loop owns the cursor; the adopted runs are still
// registered so that loop claims their future events.
func (f *Facade) adoptInFlightReplayTurns(ctx context.Context, turns map[string]*turnState, finished map[string]bool, lastEventID string) {
	adopt := make(map[string]*turnState)
	for runID, ts := range turns {
		if finished[runID] || runID == f.SessionID {
			continue
		}
		adopt[runID] = ts
	}
	if len(adopt) == 0 {
		return
	}

	f.mu.Lock()
	if f.turns == nil {
		f.turns = make(map[string]*turnState)
	}
	for runID, ts := range adopt {
		if _, exists := f.turns[runID]; !exists {
			f.turns[runID] = ts
		}
	}
	if !f.streamStarted && f.streamCursor == "" && lastEventID != "" {
		f.streamCursor = lastEventID
	}
	f.mu.Unlock()

	log.Printf("codex facade: adopted %d in-flight turn(s) from replayed history; attaching live", len(adopt))
	if err := f.ensureEventLoop(ctx); err != nil {
		// The turn stays tracked: a later turn/start's own ensureEventLoop
		// still picks it up from the same cursor.
		log.Printf("codex facade: attaching live for adopted turn(s) failed: %v", err)
	}
}

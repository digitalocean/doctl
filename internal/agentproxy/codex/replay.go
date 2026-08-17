// Package codex: --replay support, feeding a session's durable event history
// into the first thread this facade bootstraps.
package codex

import (
	"context"
	"log"

	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/doctl/internal/agentproxy"
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

// maxReplayEvents caps how much history one --replay fetch reconstructs. A
// single replay request only covers the newest window of a long transcript, so
// do.LoadSessionHistory walks backwards for the rest; this bounds that walk, so
// a session with a very long transcript can't hold up the thread bootstrap
// waiting on history no scrollback will show anyway. The newest
// maxReplayEvents events are the ones kept.
const maxReplayEvents = 5000

// replaySessionHistory fetches this session's durable event history via
// replay-only StreamSession calls — distinct from, and unrelated to, the
// single live StreamSession call runEventLoop owns for the whole connection
// (see the Facade.mu doc comment for why there's only ever one of those; a
// replay-only stream reads to completion and closes on its own, per
// harness-api's handleStreamSession, so it never overlaps with or displaces
// the live one). Each historical event is fed through the same translateEvent
// used for live events, so past turns arrive as an ordinary sequence of
// turn/item notifications — the codex protocol has no separate "history"
// shape to send instead.
//
// History is collected in full before any of it is translated: the server
// bounds one replay to its newest window, so the older events arrive from
// later backward pages and have to be reordered before they can be replayed
// as a chronological sequence of turns.
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
// run.completed/run.failed among its events yet) is left in this local map
// reporting "inProgress" and never gets completed: this facade has no way to
// attach to an already-running turn's future events, since the live loop
// only ever starts from this connection's own turn/start (see trackTurn). A
// known, narrow gap — not a bug to fix here — the harness has no "attach to
// an existing in-flight run" concept for this proxy to use.
//
// Marks replayDone only on reaching the natural end of history — not on any
// early-abort path (the history fetch failing, or a dead client mid-replay) —
// so an aborted attempt is retried on the next thread/start or thread/resume
// on this same connection instead of permanently foreclosing --replay for the
// rest of the connection on one flaky fetch.
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

	history, err := do.LoadSessionHistory(ctx, f.Sessions, f.SessionID, do.SessionHistoryOptions{
		MaxEvents: maxReplayEvents,
	})
	if err != nil {
		log.Printf("codex facade: replay history fetch failed, will retry on next connect: %v", err)
		return
	}

	turns := make(map[string]*turnState)
	for _, ev := range history {
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

		if f.translateEvent(ctx, ev, ts, true) {
			log.Printf("codex facade: replay stopped early (client disconnected), will retry on next connect")
			return
		}
	}
	completed = true
}

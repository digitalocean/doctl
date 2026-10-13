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
	"fmt"

	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
)

// headlessDrain consumes one connection's worth of events. done ends the loop,
// meaning the work being watched finished; a non-nil error aborts it.
type headlessDrain func(stream *godo.HostedAgentSessionStream) (done bool, err error)

// runHeadlessStream follows a session's event stream with no TUI and no
// keyboard input, reconnecting across transient drops until drain reports the
// work finished, ctx is cancelled, or the stream fails unrecoverably.
//
// The reconnect policy is why this is shared rather than copied into each
// headless command. maxAutoReconnectAttempts bounds CONSECUTIVE failures, and a
// connection that stayed up for healthyStreamDuration before dropping is a
// server idle timeout rather than a fault, so it resets the budget. Without
// that distinction an agent that merely thinks for a few minutes would burn
// through the budget and be reported as a lost stream.
//
// onTerminal receives the user-facing message for a stream error that ends the
// watch for good (auth, missing session, superseded connection). The loop then
// returns nil: that situation has been reported to the user and is not an
// internal failure to raise on top of it.
func runHeadlessStream(
	ctx context.Context,
	svc do.HostedAgentsService,
	sessionID string,
	cursor *eventCursor,
	onTerminal func(msg string),
	drain headlessDrain,
) error {
	backoff := initialReconnectBackoff
	failures := 0

	for {
		if ctx.Err() != nil {
			return nil
		}

		stream, err := svc.StreamSession(ctx, sessionID, &godo.HostedAgentSessionStreamOptions{
			ReplayFrom: cursor.get(),
		})
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			if msg, terminal := classifyStreamError(err); terminal {
				onTerminal(msg)
				return nil
			}
			failures++
			if failures >= maxAutoReconnectAttempts {
				return fmt.Errorf("lost the event stream for session %s after %d attempts: %w", sessionID, failures, err)
			}
			if !reconnectSleepFn(ctx, backoff) {
				return nil
			}
			backoff = nextBackoff(backoff)
			continue
		}

		connectedAt := streamClock()
		done, drainErr := drain(stream)
		streamErr := stream.Err()
		stream.Close()

		if drainErr != nil {
			return drainErr
		}
		if done || ctx.Err() != nil {
			return nil
		}

		// A stream that stayed up is a server idle timeout, not a fault, so it
		// must not consume the failure budget — an unattended run can sit quiet
		// for a long time while the agent thinks.
		if streamClock().Sub(connectedAt) >= healthyStreamDuration {
			failures = 0
			backoff = initialReconnectBackoff
		} else {
			failures++
		}
		if streamErr != nil {
			if msg, terminal := classifyStreamError(streamErr); terminal {
				onTerminal(msg)
				return nil
			}
		}
		if failures >= maxAutoReconnectAttempts {
			return fmt.Errorf("lost the event stream for session %s after %d attempts", sessionID, failures)
		}
		if !reconnectSleepFn(ctx, backoff) {
			return nil
		}
		backoff = nextBackoff(backoff)
	}
}

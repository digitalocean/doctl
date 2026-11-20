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
	"net/http"
	"strings"
	"time"

	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
)

var (
	// wakeProgressInterval is how often a non-terminal run repeats a stage's
	// line, so a log that is being tailed shows the stage is still going.
	wakeProgressInterval = 10 * time.Second
	// wakePollTimeout bounds how long doctl waits for a session to leave
	// PAUSED after being told it is already running.
	wakePollTimeout = 45 * time.Second
)

// runStage runs fn while showing label: one line updated in place on a
// terminal, otherwise a line when it starts and another every
// wakeProgressInterval. fn takes no context, so it runs in its own goroutine
// and ctx ends the wait rather than the call. When ctx wins, or fn fails after
// ctx ended, the error is ctx.Err() so callers can tell a deadline or Ctrl-C
// from a real failure. prog may be nil to stay silent.
func runStage(ctx context.Context, prog *creationProgress, label string, fn func() error) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	prog.wait(label)

	result := make(chan error, 1)
	go func() { result <- fn() }()

	ticker := time.NewTicker(wakeProgressInterval)
	defer ticker.Stop()

	for {
		select {
		case err := <-result:
			if err != nil {
				prog.stopSpin()
				if ctx.Err() != nil {
					return ctx.Err()
				}
			}
			return err
		case <-ctx.Done():
			prog.stopSpin()
			return ctx.Err()
		case <-ticker.C:
			// On a terminal this only relabels the live line.
			prog.wait(label)
		}
	}
}

// wakeSessionWithProgress resumes a paused session and shows that it is
// working, because the resume request can take a while and a bare "resuming…"
// line looks like a hang. prog may be nil to stay silent.
//
// It reports whether this call's resume succeeded. A 409 saying the session is
// already running is not that: someone else started the wake, so doctl waits
// for the session to leave PAUSED instead and reports false. ctx ends the wait,
// not the request.
func wakeSessionWithProgress(ctx context.Context, svc do.HostedAgentsService, sessionID string, prog *creationProgress) (bool, error) {
	err := runStage(ctx, prog, "waking the session", func() error {
		return svc.ResumeSession(sessionID)
	})
	switch {
	case err == nil:
		prog.ok(fmt.Sprintf("Session awake (%s)", prog.elapsed()))
		return true, nil
	case ctx.Err() != nil:
		return false, ctx.Err()
	case isAlreadyRunning(err):
		return false, waitForWakeByPolling(ctx, svc, sessionID, prog)
	}
	return false, wakeFailure(err)
}

// waitForWakeByPolling covers a resume that another client is already
// performing: watch the session until it is no longer PAUSED.
func waitForWakeByPolling(ctx context.Context, svc do.HostedAgentsService, sessionID string, prog *creationProgress) error {
	// Read once: the poll goroutine can outlive this call.
	interval := sessionReadyPollInterval
	pollCtx, cancel := context.WithTimeout(ctx, wakePollTimeout)
	defer cancel()

	err := runStage(pollCtx, prog, "waiting for the session to wake", func() error {
		for {
			sess, err := svc.GetSession(sessionID)
			if err != nil {
				return err
			}
			if sess.Status != godo.HostedAgentSessionStatusPaused {
				return nil
			}
			select {
			case <-pollCtx.Done():
				return pollCtx.Err()
			case <-time.After(interval):
			}
		}
	})
	switch {
	case err == nil:
		prog.stopSpin()
		return nil
	case ctx.Err() != nil:
		return ctx.Err()
	case pollCtx.Err() != nil:
		return wakeFailureText(fmt.Sprintf("it did not wake within %s", wakePollTimeout), 0, err)
	}
	return wakeFailure(err)
}

// isAlreadyRunning reports the 409 a resume gets when the session is awake.
// Any other conflict is a real error and keeps its normal handling.
func isAlreadyRunning(err error) bool {
	msg, status, ok := agentAPIError(err)
	return ok && status == http.StatusConflict && strings.Contains(strings.ToLower(msg), "already running")
}

// wakeFailure words a failed resume. Only server-side and transport failures
// get the "still paused; try again" wording: a 4xx (auth, not found, rate
// limit) is not fixed by retrying and keeps its own title and tips, and
// billing gates keep their own card.
func wakeFailure(err error) error {
	if prepayBlockedErr(err) || prepayUnknownErr(err) {
		return err
	}
	msg, status, ok := agentAPIError(err)
	if ok && status < http.StatusInternalServerError {
		return err
	}
	reason := strings.TrimSpace(msg)
	if !ok || reason == "" {
		reason = strings.TrimSpace(err.Error())
	}
	reason = strings.TrimRight(sanitizeAgentAPIMessage(stripGodoTransportNoise(reason)), ". ")
	return wakeFailureText(reason, status, err)
}

func wakeFailureText(reason string, status int, cause error) error {
	return &agentPrettyError{
		title:  "Couldn't wake the session",
		reason: fmt.Sprintf("the session could not be woken: %s. It is still paused; try again.", reason),
		status: status,
		cause:  cause,
	}
}

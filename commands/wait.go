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
	"fmt"
	"io"
	"time"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/internal/ui"
)

const (
	// defaultWaitTimeout bounds every `--wait`, so a wait always ends.
	defaultWaitTimeout = 30 * time.Minute

	// defaultActionWaitTimeout is longer because actions are bounded by how
	// much data moves, not control-plane startup. --wait-timeout still overrides.
	defaultActionWaitTimeout = 2 * time.Hour

	defaultWaitInterval = 5 * time.Second
)

// waitTimeoutDesc documents --wait-timeout. The single backquoted word is what
// Cobra shows as the flag's value placeholder, so there must be exactly one.
const waitTimeoutDesc = "The longest doctl waits for the operation to finish, as a `duration` such as 10m or 1h30m. " +
	"When it elapses, doctl stops waiting and exits with an error, but the operation itself keeps running."

// waitOp describes a long-running operation to the shared waiter.
type waitOp struct {
	// Heading optionally titles the wait, as in "Creating Droplet".
	Heading string

	// Activity is the progress line, as in "Creating Droplet (web-01)".
	Activity string

	// Subject reads inside a sentence: "droplet (web-01) to become active".
	Subject string

	// Success is the finished statement: "Droplet (web-01) is active".
	Success string

	// Interval overrides the poll interval. Defaults to defaultWaitInterval.
	Interval time.Duration
}

// pollFunc re-reads a resource and reports whether it has settled. Returning an
// error abandons the wait. detail is optional context shown next to the progress
// line; report it only while it says something the activity does not.
type pollFunc func() (done bool, detail string, err error)

// waiter drives a poll loop and reports its progress.
type waiter struct {
	env     ui.Env
	timeout time.Duration

	// interval overrides each operation's interval, so tests need no real sleeps.
	interval time.Duration
}

// newWaiter builds the waiter for this invocation, honoring --wait-timeout.
func newWaiter(c *CmdConfig) (waiter, error) {
	timeout, err := c.Doit.GetDuration(c.NS, doctl.ArgWaitTimeout)
	if err != nil {
		return waiter{}, err
	}

	// Zero means unspecified, not a request to give up immediately.
	if timeout <= 0 {
		timeout = defaultWaitTimeout
	}

	return waiter{env: c.UI, timeout: timeout}, nil
}

func newTestWaiter() waiter {
	return waiter{
		env:      ui.Plain(io.Discard, io.Discard),
		timeout:  defaultWaitTimeout,
		interval: time.Millisecond,
	}
}

// wait polls until the operation completes, the deadline passes, or poll fails,
// reporting progress on stderr. The first poll happens immediately.
func (w waiter) wait(op waitOp, poll pollFunc) error {
	interval := op.Interval
	if w.interval > 0 {
		interval = w.interval
	}
	if interval <= 0 {
		interval = defaultWaitInterval
	}

	timeout := w.timeout
	if timeout <= 0 {
		timeout = defaultWaitTimeout
	}

	var opts []ui.SpinnerOption
	if op.Heading != "" {
		opts = append(opts, ui.WithHeading("%s", op.Heading))
	}

	spinner := w.env.NewSpinner(op.Activity, opts...)
	spinner.Start()
	defer spinner.Stop()

	deadline := time.Now().Add(timeout)

	for {
		done, detail, err := poll()
		if err != nil {
			spinner.Fail("Failed while waiting for %s", op.Subject)
			return err
		}

		if done {
			spinner.Succeed("%s", op.Success)
			reportedOutcome = true

			return nil
		}

		// Reported on every pass, not only on change, so a plain stream keeps
		// its heartbeat.
		message := op.Activity
		if detail != "" {
			message = fmt.Sprintf("%s (%s)", message, detail)
		}
		spinner.Message(message)

		remaining := time.Until(deadline)
		if remaining <= 0 {
			spinner.Fail("Timed out waiting for %s", op.Subject)
			return &waitTimeoutError{subject: op.Subject, timeout: timeout}
		}

		// Poll once more right on the deadline rather than sleeping past it,
		// so a resource that settles just in time is still noticed.
		sleep := interval
		if remaining < sleep {
			sleep = remaining
		}

		time.Sleep(sleep)
	}
}

type waitTimeoutError struct {
	subject string
	timeout time.Duration
}

func (e *waitTimeoutError) Error() string {
	return fmt.Sprintf(
		"timed out after %s waiting for %s. The operation is still running; check on it with a get command, or allow more time with --wait-timeout",
		e.timeout, e.subject,
	)
}

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

// Every `--wait` in doctl polls the API until a resource settles, and each one
// used to do it slightly differently: some printed a trail of dots, some
// printed nothing at all, and several would poll forever if the resource never
// settled. The waiter below is the single implementation they all share, so
// that a user sees the same progress line and the same timeout behaviour no
// matter which resource they are waiting on.

const (
	// defaultWaitTimeout bounds every `--wait`. Provisioning a database
	// cluster or a Kubernetes control plane legitimately takes tens of
	// minutes, so the default is generous; the point is that a wait always
	// ends rather than leaving a script wedged on a resource that will never
	// settle.
	defaultWaitTimeout = 30 * time.Minute

	// defaultWaitInterval is how often a resource is re-read while waiting.
	// Individual operations override it to match how quickly they can
	// plausibly change.
	defaultWaitInterval = 5 * time.Second
)

// waitTimeoutDesc documents --wait-timeout. The single backquoted word is what
// Cobra shows as the flag's value placeholder, so there must be exactly one.
const waitTimeoutDesc = "The longest doctl waits for the operation to finish, as a `duration` such as 10m or 1h30m. " +
	"When it elapses, doctl stops waiting and exits with an error, but the operation itself keeps running."

// waitOp describes a long-running operation to the shared waiter.
type waitOp struct {
	// Subject names what is being waited on and the state it must reach, as
	// in "database cluster (abc-123) to become online". It is phrased to read
	// correctly inside both the progress line and the timeout error, so it
	// starts lower case and carries no terminal punctuation.
	Subject string

	// Success is the line reported once the operation completes, written as a
	// finished statement: "Database cluster (abc-123) is online".
	Success string

	// Interval overrides how often poll is called. Defaults to
	// defaultWaitInterval.
	//
	// There is deliberately no per-operation timeout: how long a user is
	// willing to wait is their decision, not the resource's, and one policy
	// for every wait is what makes --wait-timeout mean the same thing
	// everywhere.
	Interval time.Duration
}

func (op waitOp) message() string {
	return "Waiting for " + op.Subject
}

// pollFunc re-reads a resource and reports whether it has settled. Returning
// an error abandons the wait.
//
// detail is optional context about where the operation has got to, such as the
// resource's current status or a step count. It is shown alongside the
// progress line and refreshed on every poll, which is the difference between
// telling a user that doctl is still waiting and telling them why.
type pollFunc func() (done bool, detail string, err error)

// waiter drives a poll loop and reports its progress. It is built from a
// CmdConfig so that terminal capabilities and the user's --wait-timeout are
// resolved once, then passed to the resource-specific helpers, which stay
// unit-testable without a full command.
type waiter struct {
	env     ui.Env
	timeout time.Duration

	// interval, when set, overrides the poll interval each operation asks
	// for. It exists so that tests can exercise a real poll loop without
	// sleeping for the intervals a live API needs.
	interval time.Duration
}

// newWaiter builds the waiter for this invocation, honouring --wait-timeout.
func newWaiter(c *CmdConfig) (waiter, error) {
	timeout, err := c.Doit.GetDuration(c.NS, doctl.ArgWaitTimeout)
	if err != nil {
		return waiter{}, err
	}

	// A command that predates --wait-timeout, or a config that leaves it
	// unset, reads back as zero. Treat that as unspecified rather than as a
	// request to give up immediately.
	if timeout <= 0 {
		timeout = defaultWaitTimeout
	}

	return waiter{env: c.UI, timeout: timeout}, nil
}

// newTestWaiter returns a waiter that renders nothing and polls without
// pausing, for unit tests.
func newTestWaiter() waiter {
	return waiter{
		env:      ui.Plain(io.Discard, io.Discard),
		timeout:  defaultWaitTimeout,
		interval: time.Millisecond,
	}
}

// wait polls until the operation completes, the deadline passes, or poll
// fails, reporting progress on stderr throughout.
//
// The first poll happens immediately, because an operation that has already
// finished by the time doctl asks should not cost the user an interval.
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

	spinner := w.env.NewSpinner(op.message())
	spinner.Start()
	defer spinner.Stop()

	deadline := time.Now().Add(timeout)

	for {
		done, detail, err := poll()
		if err != nil {
			spinner.Fail("Gave up waiting for %s", op.Subject)
			return err
		}

		if done {
			spinner.Succeed("%s", op.Success)
			return nil
		}

		// Reported on every pass rather than only when the detail changes, so
		// that a plain stream can repeat an unmoved stage on its heartbeat
		// instead of falling silent for the length of the wait.
		message := op.message()
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
		if remaining < interval {
			interval = remaining
		}

		time.Sleep(interval)
	}
}

// waitTimeoutError reports that doctl stopped waiting. It says so in terms the
// user can act on: the operation itself is unaffected and the wait can be
// extended.
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

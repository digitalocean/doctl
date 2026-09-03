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
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/internal/ui"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// recordingWaiter returns a waiter that renders to buf so that the progress a
// user would see can be asserted on.
func recordingWaiter(buf *bytes.Buffer, timeout time.Duration) waiter {
	return waiter{
		env:      ui.Detect(nil, buf, ui.WithAnimation(false)),
		timeout:  timeout,
		interval: time.Millisecond,
	}
}

func TestWaitPollsUntilDone(t *testing.T) {
	var buf bytes.Buffer
	w := recordingWaiter(&buf, time.Minute)

	calls := 0
	err := w.wait(waitOp{
		Subject: "database (abc) to become online",
		Success: "Database (abc) is online",
	}, func() (bool, string, error) {
		calls++
		return calls == 3, "creating", nil
	})

	require.NoError(t, err)
	assert.Equal(t, 3, calls)
	assert.Contains(t, buf.String(), "Waiting for database (abc) to become online")
	assert.Contains(t, buf.String(), "Success: Database (abc) is online")
}

// TestWaitPollsImmediately guards the case that used to cost a full interval:
// an operation that has already finished should return at once.
func TestWaitPollsImmediately(t *testing.T) {
	var buf bytes.Buffer
	w := recordingWaiter(&buf, time.Minute)

	calls := 0
	start := time.Now()
	err := w.wait(waitOp{
		Subject:  "action (1) to complete",
		Success:  "Action (1) completed",
		Interval: time.Hour,
	}, func() (bool, string, error) {
		calls++
		return true, "", nil
	})

	require.NoError(t, err)
	assert.Equal(t, 1, calls)
	assert.Less(t, time.Since(start), time.Second)
}

func TestWaitReturnsPollError(t *testing.T) {
	var buf bytes.Buffer
	w := recordingWaiter(&buf, time.Minute)

	sentinel := errors.New("cluster entered status `errored`")
	err := w.wait(waitOp{
		Subject: "cluster (abc) to start running",
		Success: "Cluster (abc) is running",
	}, func() (bool, string, error) {
		return false, "", sentinel
	})

	assert.ErrorIs(t, err, sentinel)
	assert.Contains(t, buf.String(), "Failure: Gave up waiting for cluster (abc) to start running")
}

func TestWaitTimesOut(t *testing.T) {
	var buf bytes.Buffer
	w := recordingWaiter(&buf, 5*time.Millisecond)

	err := w.wait(waitOp{
		Subject: "database (abc) to become online",
		Success: "Database (abc) is online",
	}, func() (bool, string, error) {
		return false, "creating", nil
	})

	var timeout *waitTimeoutError
	require.ErrorAs(t, err, &timeout)

	// The message has to tell the user that the resource is unaffected and how
	// to wait longer, because otherwise a timeout reads like a failed create.
	assert.Contains(t, err.Error(), "timed out after 5ms waiting for database (abc) to become online")
	assert.Contains(t, err.Error(), "still running")
	assert.Contains(t, err.Error(), "--wait-timeout")
	assert.Contains(t, buf.String(), "Failure: Timed out waiting for database (abc) to become online")
}

func TestWaitShowsPollDetail(t *testing.T) {
	t.Run("a terminal carries the detail on its updating line", func(t *testing.T) {
		var buf bytes.Buffer
		w := waiter{
			env:      ui.Detect(nil, &buf, ui.WithAnimation(true)),
			timeout:  time.Minute,
			interval: time.Millisecond,
		}

		calls := 0
		err := w.wait(waitOp{
			Subject: "app (abc) deployment to complete",
			Success: "App (abc) deployment is complete",
		}, func() (bool, string, error) {
			calls++
			return calls == 3, "2 of 7 steps complete", nil
		})

		require.NoError(t, err)
		assert.Contains(t, buf.String(), "2 of 7 steps complete")
	})

	// Without the updating line, each stage has to be reported on its own
	// line: a wait that logged only its opening and closing lines would leave
	// a CI job with no way to tell a slow provision from a stuck one.
	t.Run("a plain stream gets a line per stage change", func(t *testing.T) {
		var buf bytes.Buffer
		w := recordingWaiter(&buf, time.Minute)

		calls := 0
		err := w.wait(waitOp{
			Subject: "app (abc) deployment to complete",
			Success: "App (abc) deployment is complete",
		}, func() (bool, string, error) {
			calls++
			return calls == 3, fmt.Sprintf("%d of 7 steps complete", calls), nil
		})

		require.NoError(t, err)
		assert.Contains(t, buf.String(), "1 of 7 steps complete")
		assert.Contains(t, buf.String(), "2 of 7 steps complete")
	})
}

func TestWaitNeverWritesToOut(t *testing.T) {
	var out, errOut bytes.Buffer
	w := waiter{
		env:      ui.Detect(&out, &errOut, ui.WithAnimation(true)),
		timeout:  time.Minute,
		interval: time.Millisecond,
	}

	err := w.wait(waitOp{
		Subject: "database (abc) to become online",
		Success: "Database (abc) is online",
	}, func() (bool, string, error) {
		return true, "", nil
	})

	require.NoError(t, err)
	assert.Empty(t, out.String(), "progress must not contaminate piped data")
	assert.NotEmpty(t, errOut.String())
}

func TestNewWaiter(t *testing.T) {
	tests := []struct {
		name     string
		timeout  time.Duration
		expected time.Duration
	}{
		{
			name:     "an explicit timeout is honoured",
			timeout:  90 * time.Second,
			expected: 90 * time.Second,
		},
		{
			name:     "an unset timeout falls back to the shared default",
			timeout:  0,
			expected: defaultWaitTimeout,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
				config.Doit.Set(config.NS, doctl.ArgWaitTimeout, tt.timeout)

				w, err := newWaiter(config)
				require.NoError(t, err)
				assert.Equal(t, tt.expected, w.timeout)
			})
		})
	}
}

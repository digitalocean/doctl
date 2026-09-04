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

package ui

import (
	"bytes"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/muesli/termenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fixedClock advances a spinner's notion of time by a known amount per call so
// that elapsed times appear in output deterministically.
func fixedClock(step time.Duration) func() time.Time {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	calls := 0

	return func() time.Time {
		t := base.Add(time.Duration(calls) * step)
		calls++
		return t
	}
}

func TestSpinnerWithoutAnimation(t *testing.T) {
	tests := []struct {
		name     string
		opts     []Option
		finish   func(*Spinner)
		expected string
	}{
		{
			// A redirected wait reads as a running narrative: one line per
			// stage and then the outcome, with no label standing in for the
			// glyph a terminal would have drawn.
			name:     "a completed operation closes on the outcome and its elapsed time",
			finish:   func(s *Spinner) { s.Succeed("Database is online") },
			expected: "Waiting for database (0s)\nDatabase is online (1s)\n",
		},
		{
			// The cause is reported separately by the command's own Error
			// line, so the closing line says only what doctl gave up on.
			name:     "a failed operation closes on what it was waiting for",
			finish:   func(s *Spinner) { s.Fail("Timed out waiting for database") },
			expected: "Waiting for database (0s)\nTimed out waiting for database (1s)\n",
		},
		{
			// A plain stream is already glyph-free, so the ASCII fallback has
			// nothing left to substitute.
			name:     "the ascii fallback changes nothing",
			opts:     []Option{WithASCII(true)},
			finish:   func(s *Spinner) { s.Succeed("Database is online") },
			expected: "Waiting for database (0s)\nDatabase is online (1s)\n",
		},
		{
			name:     "stopping without an outcome leaves only the opening line",
			finish:   func(s *Spinner) { s.Stop() },
			expected: "Waiting for database (0s)\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out, errOut bytes.Buffer
			env := Detect(&out, &errOut, append(tt.opts, WithAnimation(false))...)

			s := env.NewSpinner("Waiting for database")
			s.now = fixedClock(time.Second)
			s.Start()
			tt.finish(s)

			assert.Equal(t, tt.expected, errOut.String())
			assert.Empty(t, out.String(), "chrome must never reach the data stream")
		})
	}
}

// TestSpinnerWritesNoEscapesWithoutAnimation is the property that makes the
// spinner safe to leave enabled when stderr is redirected to a log or a pipe.
func TestSpinnerWritesNoEscapesWithoutAnimation(t *testing.T) {
	var out, errOut bytes.Buffer
	// A colour profile is forced on to prove that the absence of escapes comes
	// from animation being off, not merely from an undetectable buffer.
	env := Detect(&out, &errOut, WithAnimation(false), WithProfile(termenv.TrueColor))

	s := env.NewSpinner("Waiting for database")
	s.Start()
	s.Message("Waiting for database (provisioning)")
	s.Succeed("Database is online")

	assert.NotContains(t, errOut.String(), eraseLine)
	assert.NotContains(t, errOut.String(), "\r")
}

// TestSpinnerWritesNoGlyphsWithoutAnimation guards the other half of that
// property: a glyph is a terminal affordance, and a log or a pipe is read by
// grep and by people who never saw the terminal it would have been drawn on.
func TestSpinnerWritesNoGlyphsWithoutAnimation(t *testing.T) {
	var out, errOut bytes.Buffer
	env := Detect(&out, &errOut, WithAnimation(false), WithProfile(termenv.TrueColor))

	s := env.NewSpinner("Waiting for database")
	s.Start()
	s.Message("Waiting for database (provisioning)")
	s.Fail("Timed out waiting for database")

	// Only the unicode set is asserted on: the ASCII fallback is drawn from
	// characters like `o` and `+` that occur in ordinary prose, so it cannot
	// be distinguished from a message by substring search. Nor does it need
	// to be, since the fallback exists for terminals that cannot render the
	// unicode set, and a plain stream is now given no glyph from either.
	rendered := errOut.String()
	for _, glyph := range append(unicodeGlyphs.Spinner, unicodeGlyphs.Success, unicodeGlyphs.Failure, unicodeGlyphs.Pending) {
		assert.NotContains(t, rendered, glyph)
	}
}

// TestSpinnerReportsEachStageChangeWithoutAnimation covers what a CI log gets
// in place of the single updating line a terminal would show.
func TestSpinnerReportsEachStageChangeWithoutAnimation(t *testing.T) {
	var out, errOut bytes.Buffer
	env := Detect(&out, &errOut, WithAnimation(false))

	s := env.NewSpinner("Waiting for database (some-id) to become online")
	s.now = fixedClock(0)
	s.Start()
	s.Message("Waiting for database (some-id) to become online (creating)")
	// A poll that comes back with the stage unchanged must not add a line, or
	// a twenty minute provision fills the log with one line per poll. The
	// clock is frozen here, so the heartbeat never comes due.
	s.Message("Waiting for database (some-id) to become online (creating)")
	s.Message("Waiting for database (some-id) to become online (configuring)")
	s.Succeed("Database (some-id) is online")

	expected := "Waiting for database (some-id) to become online (0s)\n" +
		"Waiting for database (some-id) to become online (creating) (0s)\n" +
		"Waiting for database (some-id) to become online (configuring) (0s)\n" +
		"Database (some-id) is online (0s)\n"

	assert.Equal(t, expected, errOut.String())
	assert.Empty(t, out.String())
}

// TestSpinnerRepeatsUnchangedStageOnHeartbeat covers the other half of that
// behaviour: a stage that has not moved still has to show up periodically, or
// a slow provision looks to whoever is watching the build log like a hung job.
func TestSpinnerRepeatsUnchangedStageOnHeartbeat(t *testing.T) {
	var out, errOut bytes.Buffer
	env := Detect(&out, &errOut, WithAnimation(false))

	s := env.NewSpinner("Waiting for droplet to become active")
	s.now = fixedClock(StageHeartbeat)
	s.Start()
	s.Message("Waiting for droplet to become active (new)")
	s.Message("Waiting for droplet to become active (new)")
	s.Message("Waiting for droplet to become active (new)")

	expected := "Waiting for droplet to become active (0s)\n" +
		"Waiting for droplet to become active (new) (1m0s)\n" +
		"Waiting for droplet to become active (new) (2m0s)\n" +
		"Waiting for droplet to become active (new) (3m0s)\n"

	assert.Equal(t, expected, errOut.String())
}

// TestSpinnerMessageBeforeStartIsNotReportedTwice guards the plain path
// against reporting a stage before there is an elapsed time to report it
// against.
func TestSpinnerMessageBeforeStartIsNotReportedTwice(t *testing.T) {
	var out, errOut bytes.Buffer
	env := Detect(&out, &errOut, WithAnimation(false))

	s := env.NewSpinner("Waiting for database")
	s.now = fixedClock(0)
	s.Message("Waiting for database (creating)")
	s.Start()
	s.Stop()

	assert.Equal(t, "Waiting for database (creating) (0s)\n", errOut.String())
}

func TestSpinnerAnimates(t *testing.T) {
	var out, errOut bytes.Buffer
	env := Detect(&out, &errOut, WithAnimation(true))

	s := env.NewSpinner("Waiting for database")
	s.Start()
	s.Succeed("Database is online")

	rendered := errOut.String()

	// At least one frame is painted before the spinner can be halted, and the
	// closing line must erase it rather than appear underneath it.
	assert.Contains(t, rendered, eraseLine)
	assert.Contains(t, rendered, unicodeGlyphs.Spinner[0])
	assert.True(t, strings.HasSuffix(rendered, "✓ Database is online (0s)\n"), "got %q", rendered)
	assert.Empty(t, out.String())
}

// TestSpinnerPaintsOnlyTheSymbol pins the animated line to the design system:
// the symbol carries the colour and the message stays default, so a routine
// wait does not read as a caution. The frame uses the info slot rather than
// the warning one for the same reason, and the elapsed counter trails as
// muted chrome.
//
// This deliberately differs from the agents renderer on the beta line, which
// paints the frame and the message together in the warning colour.
func TestSpinnerPaintsOnlyTheSymbol(t *testing.T) {
	var out, errOut bytes.Buffer
	env := Detect(&out, &errOut, WithAnimation(true), WithProfile(termenv.TrueColor))

	s := env.NewSpinner("Waiting for database")
	s.Start()
	s.Succeed("Database is online")

	rendered := errOut.String()

	assert.Contains(t, rendered,
		env.SprintErr(env.NewErrStyle().Foreground(ColorInfo), unicodeGlyphs.Spinner[0])+" Waiting for database ",
		"the frame is painted in the info slot and the message left plain")

	assert.NotContains(t, rendered, env.SprintErr(env.NewErrStyle().Foreground(ColorWarning), "Waiting for database"),
		"the message must not be painted as a warning")

	closing := env.SprintErr(env.NewErrStyle().Foreground(ColorSuccess), unicodeGlyphs.Success) + " Database is online "
	assert.Contains(t, rendered, closing, "the closing symbol carries the colour and the message stays default")
}

func TestSpinnerMessageChangesTheAnimatedLine(t *testing.T) {
	var out, errOut bytes.Buffer
	env := Detect(&out, &errOut, WithAnimation(true))

	s := env.NewSpinner("Waiting for app deployment")
	s.Message("Waiting for app deployment (3 of 7 steps complete)")
	// Start after the message so the very first frame already carries it,
	// which keeps the assertion independent of the tick rate.
	s.Start()
	s.Stop()

	assert.Contains(t, errOut.String(), "3 of 7 steps complete")
}

func TestSpinnerIsIdempotentAfterFinishing(t *testing.T) {
	var out, errOut bytes.Buffer
	env := Detect(&out, &errOut, WithAnimation(false))

	s := env.NewSpinner("Waiting for database")
	s.now = fixedClock(time.Second)
	s.Start()
	s.Succeed("Database is online")

	// Callers defer Stop and then report an outcome, so the redundant Stop
	// must not add a line or erase the one that matters.
	before := errOut.String()
	s.Stop()
	s.Succeed("Database is online")
	s.Fail("Should not appear")

	assert.Equal(t, before, errOut.String())
}

func TestSpinnerStyling(t *testing.T) {
	t.Run("styles the glyph when err supports colour", func(t *testing.T) {
		var out, errOut bytes.Buffer
		env := Detect(&out, &errOut, WithAnimation(false), WithProfile(termenv.TrueColor))

		s := env.NewSpinner("Waiting for database")
		s.Start()
		s.Succeed("Database is online")

		assert.Contains(t, errOut.String(), "\x1b[")
	})

	t.Run("machine output is never styled", func(t *testing.T) {
		var out, errOut bytes.Buffer
		env := Detect(&out, &errOut, WithMachineOutput(true), WithProfile(termenv.TrueColor))

		s := env.NewSpinner("Waiting for database")
		s.Start()
		s.Succeed("Database is online")

		assert.NotContains(t, errOut.String(), "\x1b[")
	})
}

func TestSpinnerElapsed(t *testing.T) {
	env := Plain(io.Discard, io.Discard)

	s := env.NewSpinner("Waiting for database")
	assert.Zero(t, s.Elapsed(), "an unstarted spinner has not elapsed")

	s.now = fixedClock(3 * time.Second)
	s.Start()
	assert.Equal(t, 3*time.Second, s.Elapsed())
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		width    int
		expected string
	}{
		{
			name:     "a line within the width is untouched",
			line:     "waiting",
			width:    20,
			expected: "waiting",
		},
		{
			name:     "a zero width leaves the line unconstrained",
			line:     "waiting for a database to become online",
			width:    0,
			expected: "waiting for a database to become online",
		},
		{
			name:     "an overlong line is cut to the width",
			line:     "waiting for a database",
			width:    10,
			expected: "waiting f…",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncate(tt.line, tt.width, "…")
			assert.Equal(t, tt.expected, got)

			if tt.width > 0 {
				require.LessOrEqual(t, len([]rune(got)), tt.width)
			}
		})
	}
}

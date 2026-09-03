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
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// SpinnerInterval is how often an animated spinner repaints. It is fast enough
// to read as motion and slow enough that a remote terminal is not flooded.
const SpinnerInterval = 100 * time.Millisecond

// eraseLine returns the cursor to the start of the line and clears it. It is
// only ever emitted when Anim is set, which requires a terminal on Err, so a
// redirected stream never receives escape sequences.
const eraseLine = "\r\x1b[2K"

// Stage names the state a progress line reports. A redirected stream leads
// with the stage spelled out instead of a glyph, because ✓ and ✗ mean nothing
// to grep and nothing to whoever reads the build log without having seen the
// terminal it would have been drawn on.
const (
	stageSuccess = "Success"
	stageFailure = "Failure"
)

// Spinner reports the progress of a long-running operation. It renders to Err
// so that data on Out stays parseable, and it degrades in two steps: an
// animated frame on an interactive terminal, and plain text everywhere else —
// one line per stage change, with no glyphs and no escape sequences.
//
// A Spinner is safe for concurrent use, and every method is a no-op after Stop
// so that callers may defer a Stop and still report an outcome.
type Spinner struct {
	env    Env
	out    io.Writer
	glyphs Glyphs
	now    func() time.Time

	mu      sync.Mutex
	message string
	started time.Time
	// painted records that an animation frame is currently on screen and must
	// be erased before anything else is written.
	painted bool
	stopped bool
	// reported is the last message written to a plain stream, so that a stage
	// which has not moved is not reported again.
	reported string

	quit chan struct{}
	done chan struct{}
}

// NewSpinner returns a Spinner that reports message while an operation runs.
// The caller must Start it.
func (e Env) NewSpinner(message string) *Spinner {
	return &Spinner{
		env:     e,
		out:     e.ErrWriter(),
		glyphs:  e.Glyphs(),
		now:     time.Now,
		message: message,
	}
}

// Start begins reporting progress. On an interactive terminal it animates
// until the spinner is stopped; otherwise it prints the opening line once and
// leaves later stages to Message, so that a log records where the operation
// got to without accumulating a frame per tick.
func (s *Spinner) Start() {
	s.mu.Lock()
	if s.stopped || !s.started.IsZero() {
		s.mu.Unlock()
		return
	}
	s.started = s.now()
	animate := s.env.Anim

	if !animate {
		s.report(0)
		s.mu.Unlock()
		return
	}
	s.mu.Unlock()

	quit, done := make(chan struct{}), make(chan struct{})
	s.quit, s.done = quit, done

	// The channels are captured as locals because halt clears the fields, and
	// a select on a nil channel blocks forever.
	go func() {
		defer close(done)

		ticker := time.NewTicker(SpinnerInterval)
		defer ticker.Stop()

		for frame := 0; ; frame++ {
			s.paint(frame)

			select {
			case <-quit:
				return
			case <-ticker.C:
			}
		}
	}()
}

// Message replaces the text shown alongside the spinner. It lets a single
// spinner narrate an operation that moves through phases without leaving a
// line behind for each one on a terminal, while a plain stream gets one line
// per phase so that a log still shows where the operation got to.
func (s *Spinner) Message(message string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.stopped {
		return
	}

	s.message = message

	// An animated line is rewritten in place by the next frame, and a spinner
	// that has not started yet reports its message when it does.
	if !s.env.Anim && !s.started.IsZero() {
		s.report(s.now().Sub(s.started))
	}
}

// Succeed stops the spinner and reports that the operation completed, leaving
// the outcome and the elapsed time on screen.
func (s *Spinner) Succeed(format string, a ...any) {
	s.finish(stageSuccess, s.glyphs.Success, ColorSuccess, fmt.Sprintf(format, a...))
}

// Fail stops the spinner and reports that the operation did not complete.
// The error itself is reported separately by the command, so the message here
// should say what doctl was waiting for rather than restate the cause.
func (s *Spinner) Fail(format string, a ...any) {
	s.finish(stageFailure, s.glyphs.Failure, ColorError, fmt.Sprintf(format, a...))
}

// Stop halts the spinner without reporting an outcome, clearing any frame it
// left on screen. It is safe to call more than once, which makes it suitable
// for a defer that guards an early return.
func (s *Spinner) Stop() {
	s.halt()

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.stopped {
		return
	}
	s.stopped = true

	if s.painted {
		fmt.Fprint(s.out, eraseLine)
		s.painted = false
	}
}

// Elapsed reports how long the operation has been running, or how long it ran
// before the spinner was stopped.
func (s *Spinner) Elapsed() time.Duration {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.started.IsZero() {
		return 0
	}

	return s.now().Sub(s.started)
}

func (s *Spinner) finish(stage, glyph string, color lipgloss.Color, message string) {
	elapsed := s.Elapsed()

	s.halt()

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.stopped {
		return
	}
	s.stopped = true

	if s.painted {
		fmt.Fprint(s.out, eraseLine)
		s.painted = false
	}

	// An animated terminal has the room and the context for a glyph; a log or
	// a pipe gets the stage named, in the same `Label: message` shape as the
	// error and notice chrome it will sit next to.
	lead := stage + ":"
	if s.env.Anim {
		lead = glyph
	}

	lead = s.env.SprintErr(s.env.NewErrStyle().Foreground(color).Bold(true), lead)
	fmt.Fprintf(s.out, "%s %s %s\n", lead, message, s.duration(elapsed))
}

// report writes the current message to a plain stream, skipping a stage that
// has already been reported: a wait that re-reads a resource every ten seconds
// would otherwise repeat one line for the whole of a twenty minute provision.
//
// The line carries no glyph and no cursor movement, which is what makes the
// spinner safe to leave enabled when stderr is a log file or a pipe. The
// caller's message already opens with the stage it is in ("Waiting for ..."),
// so nothing is prefixed here.
//
// s.mu must be held.
func (s *Spinner) report(elapsed time.Duration) {
	if s.message == s.reported {
		return
	}
	s.reported = s.message

	fmt.Fprintf(s.out, "%s %s\n", s.message, s.duration(elapsed))
}

// paint draws one animation frame over the previous one.
func (s *Spinner) paint(frame int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.stopped {
		return
	}

	frames := s.glyphs.Spinner
	glyph := s.env.SprintErr(s.env.NewErrStyle().Foreground(ColorInfo), frames[frame%len(frames)])
	line := fmt.Sprintf("%s %s %s", glyph, s.message, s.duration(s.now().Sub(s.started)))

	fmt.Fprint(s.out, eraseLine+truncate(line, s.env.Width, s.glyphs.Ellipsis))
	s.painted = true
}

// duration renders an elapsed time as dim chrome, in whole seconds so that the
// value does not churn between frames.
func (s *Spinner) duration(d time.Duration) string {
	return s.env.SprintErr(s.env.NewErrStyle().Foreground(ColorMuted), "("+d.Round(time.Second).String()+")")
}

// halt shuts the animation goroutine down and waits for the final frame to
// land, so that nothing is painted after the closing line is written.
func (s *Spinner) halt() {
	s.mu.Lock()
	quit, done := s.quit, s.done
	s.quit = nil
	s.mu.Unlock()

	if quit == nil {
		return
	}

	close(quit)
	<-done
}

// truncate keeps an animated line within the terminal so that a long message
// does not wrap and leave orphaned rows behind as the spinner repaints.
// ansi.Truncate measures in terminal cells and preserves escape sequences, so
// a styled line cannot be cut mid-sequence and leave colour switched on.
func truncate(line string, width int, ellipsis string) string {
	if width <= 0 || ansi.StringWidth(line) <= width {
		return line
	}

	return ansi.Truncate(line, width, ellipsis)
}

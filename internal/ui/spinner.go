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

// SpinnerInterval is how often an animated spinner repaints.
const SpinnerInterval = 120 * time.Millisecond

// eraseLine returns the cursor to the start of the line and clears it. It is
// only emitted when Anim is set, so a pipe never receives escape sequences.
const eraseLine = "\r\x1b[2K"

// StageHeartbeat is how often a plain stream repeats an unchanged stage.
const StageHeartbeat = time.Minute

// Spinner reports the progress of a long-running operation. It renders to Err
// so that data on Out stays parseable: an animated frame on a terminal, one
// plain line per stage change elsewhere. Every method is a no-op after Stop.
type Spinner struct {
	env    Env
	out    io.Writer
	glyphs Glyphs
	now    func() time.Time

	heading string

	mu      sync.Mutex
	message string
	started time.Time
	// painted records that a frame is on screen and must be erased first.
	painted bool
	stopped bool
	// reported and reportedAt are the last message written to a plain stream.
	reported   string
	reportedAt time.Time

	quit chan struct{}
	done chan struct{}
}

// SpinnerOption configures a Spinner.
type SpinnerOption func(*Spinner)

// WithHeading titles the wait, on its own line above the progress line.
func WithHeading(format string, a ...any) SpinnerOption {
	return func(s *Spinner) {
		s.heading = fmt.Sprintf(format, a...)
	}
}

// NewSpinner returns a Spinner reporting message. The caller must Start it.
func (e Env) NewSpinner(message string, opts ...SpinnerOption) *Spinner {
	s := &Spinner{
		env:     e,
		out:     e.ErrWriter(),
		glyphs:  e.Glyphs(),
		now:     time.Now,
		message: message,
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}

// Start begins reporting progress. It animates on an interactive terminal, and
// otherwise prints the opening line and leaves later stages to Message.
func (s *Spinner) Start() {
	quit, done := make(chan struct{}), make(chan struct{})

	s.mu.Lock()
	if s.stopped || !s.started.IsZero() {
		s.mu.Unlock()
		return
	}
	s.started = s.now()

	if s.heading != "" {
		s.commit(s.styled(ColorInfo, true, s.heading))
	}

	if !s.env.Anim {
		s.report(s.started)
		s.mu.Unlock()
		return
	}

	// Assigned under the lock halt reads them under, so Stop cannot race Start.
	s.quit, s.done = quit, done
	s.mu.Unlock()

	// The channels are used as locals from here because halt clears the
	// fields, and a select on a nil channel blocks forever.
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

// Message replaces the text shown alongside the spinner.
func (s *Spinner) Message(message string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.stopped {
		return
	}

	s.message = message

	// An animated line is rewritten by the next frame; an unstarted spinner
	// reports when it starts.
	if !s.env.Anim && !s.started.IsZero() {
		s.report(s.now())
	}
}

// Note records a stage that has passed, leaving it on screen as work moves on.
func (s *Spinner) Note(format string, a ...any) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.stopped {
		return
	}

	text := fmt.Sprintf(format, a...)
	if !s.env.Anim {
		s.commit(s.styled(ColorInfo, false, text))
		return
	}

	glyph := s.styled(ColorInfo, false, s.glyphs.Bullet)
	s.commit(glyph + " " + s.styled(ColorInfo, false, text+s.glyphs.Ellipsis))
}

// Succeed stops the spinner and reports that the operation completed.
func (s *Spinner) Succeed(format string, a ...any) {
	s.finish(s.glyphs.Success, ColorSuccess, fmt.Sprintf(format, a...))
}

// Fail stops the spinner and reports that the operation did not complete.
func (s *Spinner) Fail(format string, a ...any) {
	s.finish(s.glyphs.Failure, ColorError, fmt.Sprintf(format, a...))
}

// Stop halts the spinner without reporting an outcome, clearing any frame it
// left on screen. It is safe to call more than once, so a defer may guard it.
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

// Elapsed reports how long the operation has run, or ran before it stopped.
func (s *Spinner) Elapsed() time.Duration {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.started.IsZero() {
		return 0
	}

	return s.now().Sub(s.started)
}

func (s *Spinner) finish(glyph string, color lipgloss.TerminalColor, message string) {
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

	// A plain stream closes on the sentence alone: no glyph, and no `Success:`
	// label, which a running narrative of one line per stage does not carry.
	if !s.env.Anim {
		fmt.Fprintf(s.out, "%s %s\n", s.styled(color, true, message), s.duration(elapsed))
		return
	}

	lead := s.env.SprintErr(s.env.NewErrStyle().Foreground(color), glyph)
	text := s.env.SprintErr(s.env.NewErrStyle().Foreground(color).Bold(true), message)

	fmt.Fprintf(s.out, "%s %s %s\n", lead, text, s.duration(elapsed))
}

// styled paints one piece of a progress line. Whether the paint lands is
// ErrStyle's decision rather than Anim's, so animation off still keeps color.
func (s *Spinner) styled(color lipgloss.TerminalColor, bold bool, text string) string {
	style := s.env.NewErrStyle().Foreground(color)
	if bold {
		style = style.Bold(true)
	}

	return s.env.SprintErr(style, text)
}

// commit writes a line that stays on screen, erasing any animation frame first.
//
// s.mu must be held.
func (s *Spinner) commit(line string) {
	if s.painted {
		fmt.Fprint(s.out, eraseLine)
		s.painted = false
	}

	fmt.Fprintln(s.out, line)
}

// report writes the current message to a plain stream, at once when the stage
// has moved and every StageHeartbeat when it has not. It carries no glyph or
// cursor movement, and takes the caller's clock reading rather than its own.
//
// s.mu must be held.
func (s *Spinner) report(now time.Time) {
	if s.message == s.reported && now.Sub(s.reportedAt) < StageHeartbeat {
		return
	}
	s.reported, s.reportedAt = s.message, now

	fmt.Fprintf(s.out, "%s %s\n", s.styled(ColorWarning, false, s.message), s.duration(now.Sub(s.started)))
}

// paint draws one animation frame over the previous one.
func (s *Spinner) paint(frame int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.stopped {
		return
	}

	frames := s.glyphs.Spinner
	style := s.env.NewErrStyle().Foreground(ColorWarning)
	glyph := s.env.SprintErr(style, frames[frame%len(frames)])
	message := s.env.SprintErr(style, s.message+s.glyphs.Ellipsis)
	line := fmt.Sprintf("%s %s %s", glyph, message, s.duration(s.now().Sub(s.started)))

	fmt.Fprint(s.out, eraseLine+truncate(line, s.env.Width, s.glyphs.Ellipsis))
	s.painted = true
}

// duration renders elapsed time as dim chrome, truncated to whole seconds.
func (s *Spinner) duration(d time.Duration) string {
	return s.env.SprintErr(s.env.NewErrStyle().Foreground(ColorMuted), "("+d.Truncate(time.Second).String()+")")
}

// halt stops the animation goroutine and waits for the final frame to land.
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

// truncate keeps an animated line within the terminal so that it does not wrap.
// ansi.Truncate preserves escape sequences, so no line is cut mid-sequence.
func truncate(line string, width int, ellipsis string) string {
	if width <= 0 || ansi.StringWidth(line) <= width {
		return line
	}

	return ansi.Truncate(line, width, ellipsis)
}

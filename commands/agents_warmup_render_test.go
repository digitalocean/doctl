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
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeScreen is a minimal ANSI terminal model — enough of CSI to replay what
// the attach display writes and then assert what the user would actually see.
// The warm-up banner bugs it guards against are only visible in the composed
// effect of cursor moves, erases and wraps, which a raw byte-stream assertion
// cannot express.
type fakeScreen struct {
	rows           [][]rune
	row, col       int
	cols           int
	savedR, savedC int
}

func newFakeScreen(rows, cols int) *fakeScreen {
	s := &fakeScreen{cols: cols, rows: make([][]rune, rows)}
	for i := range s.rows {
		s.rows[i] = blankRow(cols)
	}
	return s
}

func blankRow(cols int) []rune {
	r := make([]rune, cols)
	for i := range r {
		r[i] = ' '
	}
	return r
}

func (s *fakeScreen) newline() {
	if s.row == len(s.rows)-1 {
		s.rows = append(s.rows[1:], blankRow(s.cols))
		return
	}
	s.row++
}

func (s *fakeScreen) Write(p []byte) (int, error) {
	runes := []rune(string(p))
	for i := 0; i < len(runes); i++ {
		switch r := runes[i]; r {
		case '\r':
			s.col = 0
		case '\n':
			s.newline()
		case '\x1b':
			i += s.escape(runes[i+1:])
		default:
			if s.col >= s.cols {
				s.col = 0
				s.newline()
			}
			s.rows[s.row][s.col] = r
			s.col++
		}
	}
	return len(p), nil
}

// escape applies one escape sequence and reports how many runes it consumed
// past the leading ESC.
func (s *fakeScreen) escape(rest []rune) int {
	if len(rest) == 0 {
		return 0
	}
	switch rest[0] {
	case '7':
		s.savedR, s.savedC = s.row, s.col
		return 1
	case '8':
		s.row, s.col = s.savedR, s.savedC
		return 1
	case '[':
		// CSI: parameters, then a final letter.
		end := 1
		for end < len(rest) && (rest[end] == ';' || (rest[end] >= '0' && rest[end] <= '9')) {
			end++
		}
		if end >= len(rest) {
			return end
		}
		n, err := strconv.Atoi(string(rest[1:end]))
		if err != nil || n < 1 {
			n = 1
		}
		s.csi(rest[end], n)
		return end + 1 // params plus the final letter
	}
	return 1
}

func (s *fakeScreen) csi(final rune, n int) {
	switch final {
	case 'A':
		if s.row -= n; s.row < 0 {
			s.row = 0
		}
	case 'B':
		if s.row += n; s.row > len(s.rows)-1 {
			s.row = len(s.rows) - 1
		}
	case 'C':
		s.col += n
	case 'D':
		if s.col -= n; s.col < 0 {
			s.col = 0
		}
	case 'K':
		for c := s.col; c < s.cols; c++ {
			s.rows[s.row][c] = ' '
		}
	case 'M':
		for i := 0; i < n; i++ {
			s.rows = append(append(s.rows[:s.row:s.row], s.rows[s.row+1:]...), blankRow(s.cols))
		}
	}
	// SGR ('m') and anything else leaves the grid untouched.
}

// lines returns the non-blank screen rows, right-trimmed.
func (s *fakeScreen) lines() []string {
	var out []string
	for _, r := range s.rows {
		if l := strings.TrimRight(string(r), " "); l != "" {
			out = append(out, l)
		}
	}
	return out
}

// trimmedLines is lines() with trailing blanks removed from each row, for
// whole-screen equality assertions.
func trimmedLines(s *fakeScreen) []string {
	out := s.lines()
	for i, l := range out {
		out[i] = strings.TrimRight(l, " ")
	}
	return out
}

func (s *fakeScreen) countLines(substr string) int {
	n := 0
	for _, l := range s.lines() {
		if strings.Contains(l, substr) {
			n++
		}
	}
	return n
}

// warmupScreen wires an attachState to a fakeScreen with the warm-up banner up.
func warmupScreen(t *testing.T) (*fakeScreen, *attachState) {
	t.Helper()
	scr := newFakeScreen(12, 80)
	state := newAttachState(scr, &pendingHITL{})
	state.display.termCols = 80
	state.display.setRaw(true)
	state.display.warmupInit(spinnerFrames[0], msgAgentWarmup)
	return scr, state
}

// typeDuringWarmup mirrors handleAttachByte's warm-up path: buffer the byte,
// then repaint the whole banner (warm-up never echoes keystrokes).
func typeDuringWarmup(state *attachState, text string) {
	for _, b := range []byte(text) {
		state.mu.Lock()
		state.lineBuf = append(state.lineBuf, b)
		state.cursor++
		state.mu.Unlock()
		state.display.redraw()
	}
}

func TestWarmupBannerCaretFollowsTypedInput(t *testing.T) {
	scr, state := warmupScreen(t)
	typeDuringWarmup(state, "hello")

	require.Contains(t, scr.lines(), "> hello")
	// The caret must sit after the final "o", not back at the prompt: warm-up
	// suppresses echo, so this repaint is the only thing that moves it.
	assert.Equal(t, len("> hello"), scr.col, "caret column should follow the typed text")
	assert.Equal(t, "> hello", strings.TrimRight(string(scr.rows[scr.row]), " "),
		"caret should rest on the prompt row")
}

func TestWarmupBannerCaretHonoursMidLineEditing(t *testing.T) {
	scr, state := warmupScreen(t)
	typeDuringWarmup(state, "hello")

	state.mu.Lock()
	state.cursor = 2 // caret before the first "l"
	state.mu.Unlock()
	state.display.redraw()

	assert.Equal(t, len("> he"), scr.col)
}

func TestWarmupBannerKeepsOneSpinnerRowAcrossEvents(t *testing.T) {
	scr, state := warmupScreen(t)
	typeDuringWarmup(state, "hello")
	state.display.warmupSetPhase("sandbox ready · starting agent")
	state.display.warmupSetQueued(msgAgentWarmupQueued)

	// A plain event write mid-warm-up used to land on the prompt row and push
	// the prompt down without the banner knowing, so every later frame painted
	// a fresh spinner row.
	fmt.Fprintln(state.display, "sandbox provisioned")
	state.display.warmupSetFrame(spinnerFrames[1])
	state.display.warmupSetFrame(spinnerFrames[2])

	assert.Equal(t, 1, scr.countLines(msgAgentWarmup), "exactly one spinner row should remain")
	assert.Equal(t, 1, scr.countLines("sandbox ready · starting agent"))
	assert.Equal(t, 1, scr.countLines(msgAgentWarmupQueued))

	lines := scr.lines()
	assert.Equal(t, "sandbox provisioned", lines[0], "event text belongs above the pinned banner")
	assert.Contains(t, lines[1], msgAgentWarmup)
	assert.Equal(t, "> hello", lines[len(lines)-1], "prompt and input stay at the bottom")
}

func TestWarmupBannerCommitsSubmittedLineAboveBanner(t *testing.T) {
	scr, state := warmupScreen(t)
	typeDuringWarmup(state, "hello")
	state.display.warmupSetQueued(msgAgentWarmupQueued)

	visual := displayInputBuffer(state.lineBuf)
	require.Equal(t, "hello", readSubmittedInput(state))
	echoAttachSubmitNewline(state.display, visual)

	lines := scr.lines()
	assert.Equal(t, "> hello", lines[0], "the queued message must stay visible in scrollback")
	assert.Contains(t, lines[1], msgAgentWarmup, "banner stays pinned under the committed line")
	assert.Equal(t, 1, scr.countLines(msgAgentWarmup))
	assert.Equal(t, ">", strings.TrimRight(lines[len(lines)-1], " "), "prompt is left empty for the next message")
}

// The reported case: two messages typed while the agent boots. Both must stay
// on screen, in order, with a single banner reporting the running total.
func TestWarmupBannerShowsEveryQueuedMessage(t *testing.T) {
	scr, state := warmupScreen(t)

	queue := func(text string, total int) {
		typeDuringWarmup(state, text)
		visual := displayInputBuffer(state.lineBuf)
		require.Equal(t, text, readSubmittedInput(state))
		echoAttachSubmitNewline(state.display, visual)
		state.display.warmupSetQueued(warmupQueuedLabel(total))
	}

	queue("hello", 1)
	queue("hello again", 2)

	assert.Equal(t, []string{
		"> hello",
		"> hello again",
		spinnerFrames[0] + " " + msgAgentWarmup,
		warmupQueuedLabel(2),
		">",
	}, trimmedLines(scr))
}

func TestWarmupBannerStopLeavesNoBlankRows(t *testing.T) {
	scr, state := warmupScreen(t)
	typeDuringWarmup(state, "hello")
	state.display.warmupSetPhase("sandbox ready · starting agent")
	state.display.warmupSetQueued(msgAgentWarmupQueued)
	state.display.warmupStop()

	assert.Equal(t, []string{"> hello"}, scr.lines(),
		"clearing the banner should delete its rows, not blank them in place")
	assert.Equal(t, len("> hello"), scr.col)
}

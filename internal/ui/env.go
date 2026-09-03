/*
Copyright 2018 The Doctl Authors All rights reserved.
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

// Package ui holds the terminal capability kernel that doctl's output
// components are built on.
//
// Every presentation decision derives from the same handful of facts about the
// current process: whether each stream is a terminal, whether colour is
// permitted, whether animation is safe, how wide the terminal is, and whether
// the caller asked for machine-readable output. Env captures those facts once
// so that components take them as an argument rather than re-deriving them, or
// worse, consulting a mutable package global.
//
// Three conventions hold throughout:
//
//   - Out carries data and Err carries chrome. Spinners, progress, prompts and
//     notices belong on Err so that piping data into another program keeps
//     working while the terminal still shows progress.
//   - Styling and animation are separate capabilities. A CI job is frequently
//     colour-capable but must never receive animation frames, so Anim is never
//     inferred from Style.
//   - An Env owns the lipgloss renderer used to draw with it. Components must
//     style through Env rather than through package-level lipgloss helpers,
//     which resolve against a process-wide renderer and therefore make output
//     depend on the machine running the code.
package ui

import (
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-isatty"
	"github.com/muesli/termenv"
	"golang.org/x/term"
)

// Env describes the terminal capabilities available to a single command
// invocation. The zero value is safe: it renders plain, unstyled, unanimated
// output, which is what tests and pipelines want.
type Env struct {
	// Out is the destination for data: tables, JSON, and anything a user might
	// pipe into another program.
	Out io.Writer

	// Err is the destination for chrome: spinners, progress, prompts, and
	// diagnostics.
	Err io.Writer

	// Style reports whether ANSI styling may be written to Out.
	Style bool

	// ErrStyle reports whether ANSI styling may be written to Err. It is
	// tracked separately from Style because the two streams are routinely
	// redirected independently, as in `doctl ... | less`.
	ErrStyle bool

	// Anim reports whether animation may be written to Err. It is stricter
	// than ErrStyle: it additionally requires an interactive session and a
	// non-CI environment.
	Anim bool

	// ASCII reports whether output must avoid non-ASCII glyphs.
	ASCII bool

	// Width is the usable terminal width, or 0 when it cannot be determined
	// and output should be left unconstrained. It describes the session, so it
	// is the right width for chrome on Err.
	Width int

	// DataWidth is the width available to data on Out, or 0 when data must be
	// left unconstrained. It is tracked separately from Width because a
	// terminal attached to Err says nothing about Out: reflowing a table to a
	// width Out does not have would truncate the values a pipeline reads.
	DataWidth int

	// Machine reports whether the caller asked for machine-readable output.
	// When true, Style, ErrStyle, and Anim are all false.
	Machine bool

	renderer    *lipgloss.Renderer
	errRenderer *lipgloss.Renderer
}

type config struct {
	interactive bool
	machine     bool
	ascii       *bool
	width       *int
	profile     *termenv.Profile
	anim        *bool
}

// Option customises capability detection.
type Option func(*config)

// WithMachineOutput marks the invocation as machine-readable, which suppresses
// all styling and animation. It takes precedence over every other option.
func WithMachineOutput(v bool) Option {
	return func(c *config) { c.machine = v }
}

// WithInteractive records whether interactive behaviour was permitted, which
// corresponds to doctl's --interactive flag. Animation additionally requires a
// terminal and a non-CI environment.
func WithInteractive(v bool) Option {
	return func(c *config) { c.interactive = v }
}

// WithASCII forces the ASCII fallback on or off, overriding DOCTL_ASCII.
func WithASCII(v bool) Option {
	return func(c *config) { c.ascii = &v }
}

// WithWidth overrides the detected terminal width. A width of 0 leaves output
// unconstrained.
func WithWidth(v int) Option {
	return func(c *config) { c.width = &v }
}

// WithProfile forces the colour profile of both streams instead of detecting
// it. Passing termenv.Ascii disables styling outright. This backs a
// `--color=always|never` style flag, and it is how tests obtain deterministic
// styled output regardless of the terminal they run under.
func WithProfile(p termenv.Profile) Option {
	return func(c *config) { c.profile = &p }
}

// WithAnimation forces animation on or off rather than deriving it. Machine
// output still wins.
func WithAnimation(v bool) Option {
	return func(c *config) { c.anim = &v }
}

// Detect resolves the capabilities of out and err.
func Detect(out, err io.Writer, opts ...Option) Env {
	cfg := config{interactive: true}
	for _, opt := range opts {
		opt(&cfg)
	}

	outProfile, errProfile := profileFor(out), profileFor(err)
	if cfg.profile != nil {
		outProfile, errProfile = *cfg.profile, *cfg.profile
	}
	if cfg.machine {
		outProfile, errProfile = termenv.Ascii, termenv.Ascii
	}

	env := Env{
		Out:         out,
		Err:         err,
		Machine:     cfg.machine,
		Style:       outProfile != termenv.Ascii,
		ErrStyle:    errProfile != termenv.Ascii,
		renderer:    newRenderer(out, outProfile),
		errRenderer: newRenderer(err, errProfile),
	}

	switch {
	case cfg.machine:
		env.Anim = false
	case cfg.anim != nil:
		env.Anim = *cfg.anim
	default:
		env.Anim = cfg.interactive && isTerminal(err) && !IsCI()
	}

	if cfg.ascii != nil {
		env.ASCII = *cfg.ascii
	} else {
		env.ASCII = asciiRequested()
	}

	if cfg.width != nil {
		env.Width, env.DataWidth = *cfg.width, *cfg.width
	} else {
		env.Width = detectWidth(out, err)
		env.DataWidth = detectWidth(out)
	}

	return env
}

// Plain returns an Env with every capability disabled. It is the appropriate
// choice for tests and for any context where deterministic output matters.
func Plain(out, err io.Writer) Env {
	return Env{
		Out:         out,
		Err:         err,
		renderer:    newRenderer(out, termenv.Ascii),
		errRenderer: newRenderer(err, termenv.Ascii),
	}
}

// Renderer returns the lipgloss renderer bound to Out.
func (e Env) Renderer() *lipgloss.Renderer {
	if e.renderer != nil {
		return e.renderer
	}

	return lipgloss.DefaultRenderer()
}

// ErrRenderer returns the lipgloss renderer bound to Err.
func (e Env) ErrRenderer() *lipgloss.Renderer {
	if e.errRenderer != nil {
		return e.errRenderer
	}

	return lipgloss.DefaultRenderer()
}

// NewStyle returns an empty style bound to Out's renderer.
func (e Env) NewStyle() lipgloss.Style {
	return e.Renderer().NewStyle()
}

// NewErrStyle returns an empty style bound to Err's renderer.
func (e Env) NewErrStyle() lipgloss.Style {
	return e.ErrRenderer().NewStyle()
}

// Sprint renders s with style when styling is permitted on Out, and returns s
// unchanged otherwise. The style is rebound to this Env's renderer so that a
// style built with package-level lipgloss helpers still honours the Env.
func (e Env) Sprint(style lipgloss.Style, s string) string {
	if !e.Style {
		return s
	}

	return style.Renderer(e.Renderer()).Render(s)
}

// SprintErr renders s with style when styling is permitted on Err, and returns
// s unchanged otherwise.
func (e Env) SprintErr(style lipgloss.Style, s string) string {
	if !e.ErrStyle {
		return s
	}

	return style.Renderer(e.ErrRenderer()).Render(s)
}

// Writer returns Out, defaulting to os.Stdout when unset.
func (e Env) Writer() io.Writer {
	if e.Out != nil {
		return e.Out
	}

	return os.Stdout
}

// ErrWriter returns Err, defaulting to os.Stderr when unset.
func (e Env) ErrWriter() io.Writer {
	if e.Err != nil {
		return e.Err
	}

	return os.Stderr
}

func newRenderer(w io.Writer, p termenv.Profile) *lipgloss.Renderer {
	if w == nil {
		w = io.Discard
	}

	r := lipgloss.NewRenderer(w)
	r.SetColorProfile(p)

	return r
}

func isTerminal(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}

	return isatty.IsTerminal(f.Fd()) || isatty.IsCygwinTerminal(f.Fd())
}

// profileFor reports the colour profile of w. termenv resolves NO_COLOR,
// CLICOLOR_FORCE, and TERM for us.
func profileFor(w io.Writer) termenv.Profile {
	f, ok := w.(*os.File)
	if !ok || !isTerminal(f) {
		return termenv.Ascii
	}

	return lipgloss.NewRenderer(f).ColorProfile()
}

func detectWidth(writers ...io.Writer) int {
	attached := false

	for _, w := range writers {
		f, ok := w.(*os.File)
		if !ok || !isTerminal(f) {
			continue
		}

		attached = true

		if width, _, err := term.GetSize(int(f.Fd())); err == nil && width > 0 {
			return width
		}
	}

	// COLUMNS is consulted only when a terminal is actually attached. Shells
	// frequently export it, and a redirected stream must not be reflowed
	// because of an ambient variable: piped output stays unconstrained.
	if !attached {
		return 0
	}

	if width, err := strconv.Atoi(os.Getenv("COLUMNS")); err == nil && width > 0 {
		return width
	}

	return 0
}

func asciiRequested() bool {
	return truthy(os.Getenv("DOCTL_ASCII"))
}

// ciVariables are set by the CI providers doctl is most likely to run under.
// Their presence suppresses animation, which would otherwise fill build logs
// with thousands of discarded frames.
var ciVariables = []string{
	"CI",
	"CONTINUOUS_INTEGRATION",
	"BUILDKITE",
	"CIRCLECI",
	"CODEBUILD_BUILD_ID",
	"DRONE",
	"GITHUB_ACTIONS",
	"GITLAB_CI",
	"JENKINS_URL",
	"TEAMCITY_VERSION",
	"TF_BUILD",
	"TRAVIS",
}

// IsCI reports whether the process appears to be running in a CI environment.
func IsCI() bool {
	for _, name := range ciVariables {
		if truthy(os.Getenv(name)) {
			return true
		}
	}

	return false
}

func truthy(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "", "0", "false", "no", "off":
		return false
	default:
		return true
	}
}

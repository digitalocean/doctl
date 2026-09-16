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
// components are built on. Env captures the current process's capabilities -
// terminals, color, animation, width - so components take them as an argument
// rather than a package global. Out carries data and Err carries chrome.
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

// Env describes the capabilities available to a single command invocation.
type Env struct {
	// Out is the destination for data: tables, JSON, anything pipeable.
	Out io.Writer

	// Err is the destination for chrome: spinners, prompts, and diagnostics.
	Err io.Writer

	// Style reports whether ANSI styling may be written to Out.
	Style bool

	// ErrStyle reports whether ANSI styling may be written to Err.
	ErrStyle bool

	// Anim reports whether animation may be written to Err. It is stricter than
	// ErrStyle, additionally requiring an interactive, non-CI session.
	Anim bool

	// ASCII reports whether output must avoid non-ASCII glyphs.
	ASCII bool

	// Width is the session width, for chrome on Err, or 0 when undetermined.
	Width int

	// DataWidth is the width available to data on Out, or 0 when data must be
	// left unconstrained. It is tracked separately from Width because a
	// terminal on Err says nothing about Out: reflowing to a width Out does not
	// have would corrupt a pipe.
	DataWidth int

	// DataTTY reports whether Out is an interactive terminal outside CI.
	// Layout that is only meaningful on a screen someone is watching live -
	// cards, boxed tables, the records fallback - is gated on it rather than
	// on Style.
	DataTTY bool

	// ErrTTY reports whether Err is an interactive terminal outside CI, which
	// gates glyphs and other screen-only chrome.
	ErrTTY bool

	// Machine reports whether machine-readable output was asked for. When true,
	// Style, ErrStyle, and Anim are all false.
	Machine bool

	// Mask reports whether sensitive values (secrets, tokens, and similar)
	// should be hidden rather than printed in full. It is independent of
	// Machine: --output json still masks, since piping to a script is not
	// consent to leak a secret into a log. It is true unless the invocation
	// is running in CI or --show was passed.
	Mask bool

	renderer    *lipgloss.Renderer
	errRenderer *lipgloss.Renderer
}

type config struct {
	interactive bool
	machine     bool
	show        bool
	ascii       *bool
	width       *int
	profile     *termenv.Profile
	anim        *bool
}

// Option customises capability detection.
type Option func(*config)

// WithMachineOutput suppresses all styling and animation, overriding options.
func WithMachineOutput(v bool) Option {
	return func(c *config) { c.machine = v }
}

// WithInteractive records whether doctl's --interactive flag permitted it.
func WithInteractive(v bool) Option {
	return func(c *config) { c.interactive = v }
}

// WithShow records whether doctl's --show flag was passed, revealing values
// Mask would otherwise hide.
func WithShow(v bool) Option {
	return func(c *config) { c.show = v }
}

// WithASCII forces the ASCII fallback on or off, overriding DOCTL_ASCII.
func WithASCII(v bool) Option {
	return func(c *config) { c.ascii = &v }
}

// WithWidth overrides the detected width. A width of 0 leaves it unconstrained.
func WithWidth(v int) Option {
	return func(c *config) { c.width = &v }
}

// WithProfile forces the color profile of both streams; termenv.Ascii is off.
func WithProfile(p termenv.Profile) Option {
	return func(c *config) { c.profile = &p }
}

// WithAnimation forces animation on or off. Machine output still wins.
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

	// CI gets the plain, uncolored, uncarded output doctl always produced
	// there, even on a runner whose pty would otherwise pass every other
	// capability check: a CI log is read later, out of context, not watched
	// live, so color, cards, boxed tables, and the records fallback are all
	// screen-only chrome it never asked for. This wins over an explicit
	// WithProfile because that option exists for tests and manual overrides,
	// neither of which run with real CI env vars set.
	if IsCI() {
		outProfile, errProfile = termenv.Ascii, termenv.Ascii
	}

	tty := !cfg.machine && !IsCI()

	env := Env{
		Out:         out,
		Err:         err,
		Machine:     cfg.machine,
		Style:       outProfile != termenv.Ascii,
		ErrStyle:    errProfile != termenv.Ascii,
		DataTTY:     tty && isTerminal(out),
		ErrTTY:      tty && isTerminal(err),
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

	// CI is treated as a script that needs the real value, not a screen
	// someone is watching, so it is exempt from masking. --show overrides
	// both directions.
	env.Mask = !cfg.show && !IsCI()

	return env
}

// Plain returns an Env with every capability disabled, for deterministic output.
func Plain(out, err io.Writer) Env {
	return Env{
		Out:         out,
		Err:         err,
		renderer:    newRenderer(out, termenv.Ascii),
		errRenderer: newRenderer(err, termenv.Ascii),
	}
}

// Profile returns the color profile resolved for Err.
func (e Env) Profile() termenv.Profile {
	if !e.ErrStyle {
		return termenv.Ascii
	}

	return e.ErrRenderer().ColorProfile()
}

// DataProfile returns the color profile for Out, where a global must point.
func (e Env) DataProfile() termenv.Profile {
	if !e.Style {
		return termenv.Ascii
	}

	return e.Renderer().ColorProfile()
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

// Sprint renders s with style when styling is permitted on Out, rebinding the
// style to this Env's renderer.
func (e Env) Sprint(style lipgloss.Style, s string) string {
	if !e.Style {
		return s
	}

	return style.Renderer(e.Renderer()).Render(s)
}

// SprintErr renders s with style when styling is permitted on Err.
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

// profileFor reports the color profile of w. termenv resolves NO_COLOR here.
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

	// COLUMNS is consulted only when a terminal is attached, never for a pipe.
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

// ciVariables are set by CI providers, and their presence suppresses animation.
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

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
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/fatih/color"
	"github.com/muesli/termenv"
	"github.com/spf13/viper"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/commands/charm/template"
	"github.com/digitalocean/doctl/internal/ui"
)

// Colour policy values accepted by --color.
const (
	colorAuto   = "auto"
	colorAlways = "always"
	colorNever  = "never"
)

// colorHelpText documents --color. Detection is per stream, so a redirected
// stream losing colour while the other keeps it is expected behaviour.
const colorHelpText = "When to use ANSI color and styling [auto|always|never]. " +
	"auto styles each stream only when it is a terminal. Honors NO_COLOR."

// resolvedEnv is the terminal capability Env for this invocation, installed
// once by installOutputPolicy after flags are parsed. It is nil until then, so
// that callers running before the root pre-run hook - and tests, which never
// install a policy - fall back to detecting on demand.
//
// Writes happen only in installOutputPolicy, which cobra calls on a single
// goroutine before the command body runs, so reads need no synchronisation.
var resolvedEnv *ui.Env

// uiEnv returns the terminal capabilities governing this invocation. Every
// component that styles, animates, or measures width derives from this rather
// than consulting the terminal itself, so a single policy answers every
// presentation question.
func uiEnv() ui.Env {
	if resolvedEnv != nil {
		return *resolvedEnv
	}

	return detectUIEnv()
}

// resolveUIEnv returns the capabilities governing writes to out.
//
// The policy - what --color, --output, and --interactive asked for - is shared
// with every other caller. Capability detection is still per stream, because
// which writer a caller was handed is exactly what decides whether styling it
// is safe: a command whose output was redirected must not have escape
// sequences reflowed into the file a script is parsing.
func resolveUIEnv(out io.Writer) ui.Env {
	if out == nil || out == io.Writer(Writer) {
		return uiEnv()
	}

	return detectUIEnvFor(out)
}

func detectUIEnv() ui.Env {
	return detectUIEnvFor(Writer)
}

func detectUIEnvFor(out io.Writer) ui.Env {
	opts := []ui.Option{
		ui.WithMachineOutput(outputFormat() != "text"),
		ui.WithInteractive(Interactive),
	}
	if opt, ok := colorOption(); ok {
		opts = append(opts, opt)
	}

	return ui.Detect(out, os.Stderr, opts...)
}

// installOutputPolicy validates --color, resolves the Env once, and points the
// two process-global styling stacks at it.
//
// lipgloss and fatih/color both resolve colour from process-wide state that is
// decided before doctl knows anything about the invocation: lipgloss from a
// default renderer bound to os.Stdout, fatih/color from an init-time check of
// whether stdout is a terminal. Retargeting both here is what makes the older
// commands/charm chrome and the remaining fatih/color call sites obey --color,
// --output, and NO_COLOR without each of their call sites having to know about
// ui.Env.
//
// charm's template output is repointed for the same reason: it defaults to the
// process's own stdout, which ignores doctl's writer.
func installOutputPolicy() error {
	switch colorPolicy() {
	case colorAuto, colorAlways, colorNever:
	default:
		return fmt.Errorf("invalid --%s value %q: must be one of %s, %s, or %s",
			doctl.ArgColor, Color, colorAuto, colorAlways, colorNever)
	}

	env := detectUIEnv()
	resolvedEnv = &env

	lipgloss.SetColorProfile(env.Profile())
	color.NoColor = !env.ErrStyle
	template.Output, template.ErrOutput = env.Writer(), env.ErrWriter()

	return nil
}

// colorOption translates --color into a forced colour profile. auto returns no
// option at all, leaving per-stream detection in place.
func colorOption() (ui.Option, bool) {
	switch colorPolicy() {
	case colorNever:
		return ui.WithProfile(termenv.Ascii), true
	case colorAlways:
		return ui.WithProfile(forcedProfile()), true
	default:
		return nil, false
	}
}

func colorPolicy() string {
	policy := strings.ToLower(strings.TrimSpace(Color))
	if policy == "" {
		return colorAuto
	}

	return policy
}

// forcedProfile picks the profile to use when colour is demanded for a stream
// that would not otherwise get it, such as a pipe feeding a pager that renders
// ANSI. ANSI256 is the safe floor; a terminal advertising COLORTERM can take
// the full range.
func forcedProfile() termenv.Profile {
	if os.Getenv("COLORTERM") != "" {
		return termenv.TrueColor
	}

	return termenv.ANSI256
}

// outputFormat reports the requested output format. viper is authoritative
// because it folds in config.yaml and the bound --output flag, but it is empty
// before initConfig runs, in which case the flag default applies.
func outputFormat() string {
	if v := viper.GetString("output"); v != "" {
		return v
	}

	return Output
}

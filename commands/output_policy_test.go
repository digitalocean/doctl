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
	"io"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/fatih/color"
	"github.com/muesli/termenv"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/digitalocean/doctl/commands/charm/template"
	"github.com/digitalocean/doctl/commands/displayers"
	"github.com/digitalocean/doctl/internal/ui"
)

// withUIEnv installs env as the policy for the duration of the test, the way
// installOutputPolicy does for a real invocation. Chrome resolves through
// uiEnv(), so this is how a test points warnings, notices, and errors at a
// buffer it can assert on.
func withUIEnv(t *testing.T, env ui.Env) {
	t.Helper()

	prev := resolvedEnv
	resolvedEnv = &env
	t.Cleanup(func() { resolvedEnv = prev })
}

// withOutputPolicy restores every piece of process state installOutputPolicy
// touches, so a test can install a real policy without leaking it.
func withOutputPolicy(t *testing.T) {
	t.Helper()

	prevEnv := resolvedEnv
	prevProfile, prevNoColor := lipgloss.ColorProfile(), color.NoColor
	prevOut, prevErrOut := template.Output, template.ErrOutput

	t.Cleanup(func() {
		resolvedEnv = prevEnv
		lipgloss.SetColorProfile(prevProfile)
		color.NoColor = prevNoColor
		template.Output, template.ErrOutput = prevOut, prevErrOut
	})
}

// TestOutputPolicy covers what the installed policy resolves to. Every surface
// derives from it, so these cases stand in for the whole CLI.
//
// Styling is decided by per-stream detection, which termenv resolves from
// NO_COLOR, CLICOLOR_FORCE, and TERM. go test does not run against a terminal,
// so both cases here are the redirected one.
func TestOutputPolicy(t *testing.T) {
	tests := []struct {
		name   string
		output string
	}{
		{
			name:   "a redirected stream is left plain",
			output: "text",
		},
		{
			// Machine-readable output is parsed by programs, so it is never
			// styled whatever the stream turns out to be.
			name:   "machine output is never styled",
			output: "json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withOutputPolicy(t)
			viper.Set("output", tt.output)
			t.Cleanup(func() { viper.Set("output", "") })

			installOutputPolicy()

			env := uiEnv()
			assert.False(t, env.Style, "styling on stdout")
			assert.False(t, env.ErrStyle, "styling on stderr")
			assert.Equal(t, termenv.Ascii, env.Profile(), "resolved profile")

			// The two legacy stacks must agree with the policy, since that is
			// what makes charm chrome and the remaining fatih/color sites
			// follow it without knowing about ui.Env. lipgloss follows Out
			// because that is the stream the call sites reading it write to.
			assert.Equal(t, env.DataProfile(), lipgloss.ColorProfile(), "lipgloss profile")
			assert.True(t, color.NoColor, "fatih/color")
		})
	}
}

// TestOutputPolicyThreadsShow covers that the global --show flag reaches
// ui.Env.Mask through the same policy every other capability resolves from,
// rather than each command reading the flag for itself.
func TestOutputPolicyThreadsShow(t *testing.T) {
	tests := []struct {
		name string
		show bool
		want bool
	}{
		{name: "show unset masks", show: false, want: true},
		{name: "show set reveals", show: true, want: false},
	}

	// Mask treats CI as a script that needs the real value, so this test's
	// own CI runner must not leak in and flip the "show unset masks" case.
	ciVariables := []string{
		"CI", "CONTINUOUS_INTEGRATION", "BUILDKITE", "CIRCLECI",
		"CODEBUILD_BUILD_ID", "DRONE", "GITHUB_ACTIONS", "GITLAB_CI",
		"JENKINS_URL", "TEAMCITY_VERSION", "TF_BUILD", "TRAVIS",
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withOutputPolicy(t)
			for _, name := range ciVariables {
				t.Setenv(name, "")
			}

			prevShow := Show
			Show = tt.show
			t.Cleanup(func() { Show = prevShow })

			installOutputPolicy()

			assert.Equal(t, tt.want, uiEnv().Mask)
		})
	}
}

func TestInstallOutputPolicyRepointsTemplateOutput(t *testing.T) {
	withOutputPolicy(t)

	installOutputPolicy()

	// Charm chrome shares the reporter wrapping doctl's writer rather than
	// writing to the process's stdout, so a command that speaks only through a
	// template is not mistaken for one that said nothing.
	require.NotNil(t, stdoutSink)
	assert.Equal(t, io.Writer(stdoutSink), template.Output,
		"charm template output should go through the stdout reporter")
	assert.Equal(t, io.Writer(Writer), stdoutSink.out,
		"the reporter should wrap doctl's writer, not the process's stdout")
}

// TestResolveUIEnvDefersToTheWriter covers a caller that writes somewhere
// other than the policy's stream, such as CmdConfig with a pager or a test
// buffer. Which writer a caller was handed is what decides whether styling it
// is safe, so the decision has to be made against that writer.
func TestResolveUIEnvDefersToTheWriter(t *testing.T) {
	var buf bytes.Buffer

	withOutputPolicy(t)
	installOutputPolicy()

	env := resolveUIEnv(&buf)
	assert.False(t, env.Style, "a buffer is not a terminal")
	assert.Equal(t, io.Writer(&buf), env.Out)
	assert.Zero(t, env.DataWidth, "an unmeasurable writer stays unconstrained")
}

// TestUnstyledOutputMatchesPlain pins the contract scripts rely on: a stream
// doctl cannot see has to keep producing what those scripts parse today,
// whatever color doctl gains elsewhere.
func TestUnstyledOutputMatchesPlain(t *testing.T) {
	withOutputPolicy(t)
	installOutputPolicy()

	item := &displayers.Droplet{Droplets: testDropletList}

	var plain, redirected bytes.Buffer
	require.NoError(t, displayers.DisplayText(item, &plain, false, nil, ui.Env{}))
	require.NoError(t, displayers.DisplayText(item, &redirected, false, nil, uiEnv()))

	assert.Equal(t, plain.String(), redirected.String())
	assert.NotContains(t, redirected.String(), "\x1b[")
}

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

	"github.com/digitalocean/doctl"
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

// withColorPolicy sets --color for the duration of the test and restores every
// piece of process state installOutputPolicy touches.
func withColorPolicy(t *testing.T, policy string) {
	t.Helper()

	prevColor, prevEnv := Color, resolvedEnv
	prevViper := viper.GetString(doctl.ArgColor)
	prevProfile, prevNoColor := lipgloss.ColorProfile(), color.NoColor
	prevOut, prevErrOut := template.Output, template.ErrOutput

	// Set both: the policy is read through viper, which is what folds in
	// config.yaml and the bound flag, and falls back to the flag variable.
	Color = policy
	viper.Set(doctl.ArgColor, policy)

	t.Cleanup(func() {
		Color, resolvedEnv = prevColor, prevEnv
		viper.Set(doctl.ArgColor, prevViper)
		lipgloss.SetColorProfile(prevProfile)
		color.NoColor = prevNoColor
		template.Output, template.ErrOutput = prevOut, prevErrOut
	})
}

// TestOutputPolicy is the colour matrix. Every surface derives from the
// installed policy, so these cases stand in for the whole CLI.
func TestOutputPolicy(t *testing.T) {
	tests := []struct {
		name     string
		policy   string
		output   string
		styles   bool
		profile  termenv.Profile
		disabled bool // expected fatih/color NoColor
	}{
		{
			// go test does not run against a terminal, so this is the
			// redirected case: auto declines to style a stream it cannot see.
			name:     "auto leaves a redirected stream plain",
			policy:   colorAuto,
			output:   "text",
			styles:   false,
			profile:  termenv.Ascii,
			disabled: true,
		},
		{
			name:     "always styles even through a pipe",
			policy:   colorAlways,
			output:   "text",
			styles:   true,
			profile:  termenv.ANSI256,
			disabled: false,
		},
		{
			name:     "never wins over a capable terminal",
			policy:   colorNever,
			output:   "text",
			styles:   false,
			profile:  termenv.Ascii,
			disabled: true,
		},
		{
			// Machine-readable output is parsed by programs, so it outranks
			// an explicit request for colour.
			name:     "machine output is never styled",
			policy:   colorAlways,
			output:   "json",
			styles:   false,
			profile:  termenv.Ascii,
			disabled: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withColorPolicy(t, tt.policy)
			t.Setenv("COLORTERM", "")
			viper.Set("output", tt.output)
			t.Cleanup(func() { viper.Set("output", "") })

			require.NoError(t, installOutputPolicy())

			env := uiEnv()
			assert.Equal(t, tt.styles, env.Style, "styling on stdout")
			assert.Equal(t, tt.styles, env.ErrStyle, "styling on stderr")
			assert.Equal(t, tt.profile, env.Profile(), "resolved profile")

			// The two legacy stacks must agree with the policy, since that is
			// what makes charm chrome and the remaining fatih/color sites
			// follow it without knowing about ui.Env. lipgloss follows Out
			// because that is the stream the call sites reading it write to.
			assert.Equal(t, env.DataProfile(), lipgloss.ColorProfile(), "lipgloss profile")
			assert.Equal(t, tt.disabled, color.NoColor, "fatih/color")
		})
	}
}

func TestColorOption(t *testing.T) {
	t.Run("auto forces nothing", func(t *testing.T) {
		withColorPolicy(t, colorAuto)

		// Adding no option is what leaves NO_COLOR, CLICOLOR_FORCE, and TERM
		// to termenv's per-stream detection rather than overriding them.
		_, forced := colorOption()
		assert.False(t, forced)
	})

	t.Run("an empty value is auto", func(t *testing.T) {
		withColorPolicy(t, "")

		_, forced := colorOption()
		assert.False(t, forced)
	})

	t.Run("never forces plain", func(t *testing.T) {
		withColorPolicy(t, colorNever)

		opt, forced := colorOption()
		require.True(t, forced)
		assert.Equal(t, termenv.Ascii, envWith(opt).Profile())
	})

	t.Run("always uses the range the terminal advertises", func(t *testing.T) {
		withColorPolicy(t, colorAlways)

		t.Setenv("COLORTERM", "")
		assert.Equal(t, termenv.ANSI256, forcedProfile())

		t.Setenv("COLORTERM", "truecolor")
		assert.Equal(t, termenv.TrueColor, forcedProfile())
	})
}

// TestColorPolicyHonorsConfig covers `color: never` in config.yaml. The flag
// is bound to viper, so the value has to be read back through viper the way
// --output is: reading the flag variable alone means a config file loses to
// the flag's own default and silently does nothing.
func TestColorPolicyHonorsConfig(t *testing.T) {
	withColorPolicy(t, colorAuto)

	// What an untouched flag plus a configured value looks like.
	Color = colorAuto
	viper.Set(doctl.ArgColor, colorNever)

	assert.Equal(t, colorNever, colorPolicy())
}

func envWith(opt ui.Option) ui.Env {
	var out, errOut bytes.Buffer

	return ui.Detect(&out, &errOut, opt)
}

func TestInstallOutputPolicyRejectsUnknownColor(t *testing.T) {
	withColorPolicy(t, "sometimes")

	err := installOutputPolicy()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid --color value")
	assert.Nil(t, resolvedEnv, "a rejected policy must not be installed")
}

func TestInstallOutputPolicyRepointsTemplateOutput(t *testing.T) {
	withColorPolicy(t, colorAuto)

	require.NoError(t, installOutputPolicy())

	assert.Equal(t, io.Writer(Writer), template.Output,
		"charm template output should follow doctl's writer, not the process's stdout")
}

// TestResolveUIEnvAppliesPolicyToAnyWriter covers a caller that writes
// somewhere other than the policy's stream, such as CmdConfig with a pager or
// a test buffer. An explicit --color choice has to travel with it, while
// anything left to detection is decided by the writer it was handed.
func TestResolveUIEnvAppliesPolicyToAnyWriter(t *testing.T) {
	var buf bytes.Buffer

	t.Run("an explicit request travels", func(t *testing.T) {
		withColorPolicy(t, colorAlways)
		require.NoError(t, installOutputPolicy())

		env := resolveUIEnv(&buf)
		assert.True(t, env.Style, "--color=always means colour wherever doctl writes")
		assert.Equal(t, io.Writer(&buf), env.Out)
	})

	t.Run("auto defers to the writer", func(t *testing.T) {
		withColorPolicy(t, colorAuto)
		require.NoError(t, installOutputPolicy())

		env := resolveUIEnv(&buf)
		assert.False(t, env.Style, "a buffer is not a terminal")
		assert.Zero(t, env.DataWidth, "an unmeasurable writer stays unconstrained")
	})

	t.Run("never travels too", func(t *testing.T) {
		withColorPolicy(t, colorNever)
		require.NoError(t, installOutputPolicy())

		assert.False(t, resolveUIEnv(&buf).Style)
	})
}

// TestColorNeverMatchesUnstyledOutput pins the escape hatch: whatever colour
// doctl gains, --color=never has to keep producing what scripts parse today.
func TestColorNeverMatchesUnstyledOutput(t *testing.T) {
	withColorPolicy(t, colorNever)
	require.NoError(t, installOutputPolicy())

	item := &displayers.Droplet{Droplets: testDropletList}

	var plain, never bytes.Buffer
	require.NoError(t, displayers.DisplayText(item, &plain, false, nil, ui.Env{}))
	require.NoError(t, displayers.DisplayText(item, &never, false, nil, uiEnv()))

	assert.Equal(t, plain.String(), never.String())
	assert.NotContains(t, never.String(), "\x1b[")
}

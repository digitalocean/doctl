package ui

import (
	"bytes"
	"io"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
	"github.com/stretchr/testify/assert"
)

// clearEnv neutralises the ambient variables that capability detection reads,
// so results do not depend on the machine running the tests.
func clearEnv(t *testing.T) {
	t.Helper()

	t.Setenv("COLUMNS", "")
	t.Setenv("DOCTL_ASCII", "")
	for _, name := range ciVariables {
		t.Setenv(name, "")
	}
}

func TestPlain(t *testing.T) {
	var out, errOut bytes.Buffer

	env := Plain(&out, &errOut)

	assert.False(t, env.Style)
	assert.False(t, env.ErrStyle)
	assert.False(t, env.Anim)
	assert.False(t, env.Machine)
	assert.Zero(t, env.Width)
}

func TestDetectNonTerminal(t *testing.T) {
	clearEnv(t)

	var out, errOut bytes.Buffer
	env := Detect(&out, &errOut)

	assert.False(t, env.Style, "a buffer is never colour-capable")
	assert.False(t, env.ErrStyle)
	assert.False(t, env.Anim, "a buffer is never animatable")
	assert.Zero(t, env.Width)
}

func TestDetectMachineOutput(t *testing.T) {
	clearEnv(t)

	var out, errOut bytes.Buffer
	env := Detect(&out, &errOut,
		WithMachineOutput(true),
		// Machine output must win even when styling and animation are forced.
		WithProfile(termenv.TrueColor),
		WithAnimation(true),
	)

	assert.True(t, env.Machine)
	assert.False(t, env.Style)
	assert.False(t, env.ErrStyle)
	assert.False(t, env.Anim)
}

func TestDetectProfile(t *testing.T) {
	clearEnv(t)

	var out, errOut bytes.Buffer

	t.Run("a colour profile enables styling", func(t *testing.T) {
		env := Detect(&out, &errOut, WithProfile(termenv.TrueColor))
		assert.True(t, env.Style)
		assert.True(t, env.ErrStyle)
	})

	t.Run("an ascii profile disables styling", func(t *testing.T) {
		env := Detect(&out, &errOut, WithProfile(termenv.Ascii))
		assert.False(t, env.Style)
		assert.False(t, env.ErrStyle)
	})
}

func TestDetectAnimation(t *testing.T) {
	clearEnv(t)

	var out, errOut bytes.Buffer

	t.Run("forced on", func(t *testing.T) {
		assert.True(t, Detect(&out, &errOut, WithAnimation(true)).Anim)
	})

	t.Run("forced off", func(t *testing.T) {
		assert.False(t, Detect(&out, &errOut, WithAnimation(false)).Anim)
	})

	t.Run("non-interactive disables animation", func(t *testing.T) {
		assert.False(t, Detect(&out, &errOut, WithInteractive(false)).Anim)
	})

	t.Run("CI disables animation", func(t *testing.T) {
		t.Setenv("GITHUB_ACTIONS", "true")
		assert.False(t, Detect(&out, &errOut).Anim)
	})
}

func TestDetectWidth(t *testing.T) {
	clearEnv(t)

	var out, errOut bytes.Buffer

	t.Run("explicit width wins", func(t *testing.T) {
		assert.Equal(t, 120, Detect(&out, &errOut, WithWidth(120)).Width)
	})

	t.Run("zero width is honoured as unconstrained", func(t *testing.T) {
		assert.Zero(t, Detect(&out, &errOut, WithWidth(0)).Width)
	})

	// A redirected stream must not be reflowed because the shell happens to
	// export COLUMNS; that would truncate piped output.
	t.Run("COLUMNS is ignored without a terminal", func(t *testing.T) {
		t.Setenv("COLUMNS", "97")
		assert.Zero(t, Detect(&out, &errOut).Width)
	})

	t.Run("malformed COLUMNS is ignored", func(t *testing.T) {
		t.Setenv("COLUMNS", "not-a-number")
		assert.Zero(t, Detect(&out, &errOut).Width)
	})
}

func TestDetectASCII(t *testing.T) {
	clearEnv(t)

	var out, errOut bytes.Buffer

	t.Run("defaults to unicode", func(t *testing.T) {
		assert.False(t, Detect(&out, &errOut).ASCII)
	})

	t.Run("DOCTL_ASCII opts in", func(t *testing.T) {
		t.Setenv("DOCTL_ASCII", "1")
		assert.True(t, Detect(&out, &errOut).ASCII)
	})

	t.Run("explicit option overrides the environment", func(t *testing.T) {
		t.Setenv("DOCTL_ASCII", "1")
		assert.False(t, Detect(&out, &errOut, WithASCII(false)).ASCII)
	})
}

func TestIsCI(t *testing.T) {
	clearEnv(t)

	assert.False(t, IsCI(), "no CI variables set")

	t.Run("a set provider variable is detected", func(t *testing.T) {
		t.Setenv("GITHUB_ACTIONS", "true")
		assert.True(t, IsCI())
	})

	t.Run("an explicitly false variable is not CI", func(t *testing.T) {
		t.Setenv("CI", "false")
		assert.False(t, IsCI())
	})
}

func TestSprint(t *testing.T) {
	clearEnv(t)

	bold := lipgloss.NewStyle().Bold(true)

	t.Run("styling is applied when the profile permits it", func(t *testing.T) {
		var out, errOut bytes.Buffer
		env := Detect(&out, &errOut, WithProfile(termenv.TrueColor))

		assert.Contains(t, env.Sprint(bold, "hi"), "\x1b[")
		assert.Contains(t, env.SprintErr(bold, "hi"), "\x1b[")
	})

	t.Run("styling is skipped when it is not", func(t *testing.T) {
		var out, errOut bytes.Buffer
		env := Detect(&out, &errOut, WithProfile(termenv.Ascii))

		assert.Equal(t, "hi", env.Sprint(bold, "hi"))
		assert.Equal(t, "hi", env.SprintErr(bold, "hi"))
	})

	t.Run("streams are gated independently", func(t *testing.T) {
		env := Env{
			Style:       false,
			ErrStyle:    true,
			renderer:    newRenderer(io.Discard, termenv.TrueColor),
			errRenderer: newRenderer(io.Discard, termenv.TrueColor),
		}

		assert.Equal(t, "hi", env.Sprint(bold, "hi"))
		assert.Contains(t, env.SprintErr(bold, "hi"), "\x1b[")
	})
}

func TestGlyphs(t *testing.T) {
	t.Run("unicode by default", func(t *testing.T) {
		assert.Equal(t, "✔", Env{}.Glyphs().Success)
	})

	t.Run("ascii fallback", func(t *testing.T) {
		assert.Equal(t, "+", Env{ASCII: true}.Glyphs().Success)
	})

	t.Run("spinner frames share a display width", func(t *testing.T) {
		for _, set := range []Glyphs{unicodeGlyphs, asciiGlyphs} {
			width := lipgloss.Width(set.Spinner[0])
			for _, frame := range set.Spinner {
				assert.Equal(t, width, lipgloss.Width(frame), "frame %q jitters the line", frame)
			}
		}
	})
}

func TestWriterDefaults(t *testing.T) {
	assert.NotNil(t, Env{}.Writer())
	assert.NotNil(t, Env{}.ErrWriter())

	var out bytes.Buffer
	assert.Equal(t, &out, Env{Out: &out}.Writer())
	assert.Equal(t, &out, Env{Err: &out}.ErrWriter())
}

func TestTruthy(t *testing.T) {
	for _, v := range []string{"", "0", "false", "no", "off", " FALSE "} {
		assert.False(t, truthy(v), "expected %q to be falsey", v)
	}

	for _, v := range []string{"1", "true", "yes", "on", "anything"} {
		assert.True(t, truthy(v), "expected %q to be truthy", v)
	}
}

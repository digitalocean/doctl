package charm

import (
	"fmt"
	"io"
	"os"

	"github.com/charmbracelet/lipgloss"
)

var (
	// ErrCanceled represents a user-initiated cancellation.
	ErrCanceled = fmt.Errorf("canceled")
)

// Style is a styled component.
type Style struct {
	style  lipgloss.Style
	output io.Writer
}

// NewStyle creates a new styled component.
func NewStyle(style lipgloss.Style) Style {
	return Style{style: style.Copy()}
}

// Lipgloss returns a copy of the underlying lipgloss.Style.
func (s Style) Lipgloss() lipgloss.Style {
	return s.style.Copy()
}

// Copy returns a copy of the style.
func (s Style) Copy() Style {
	return Style{
		style:  s.style.Copy(),
		output: s.output,
	}
}

// Inherit returns a copy of the original style with the properties from another style inherited.
// This follows lipgloss's inheritance behavior so margins, padding, and underlying string values are not inherited.
func (s Style) Inherit(styles ...Style) Style {
	c := s.Copy()
	for _, style := range styles {
		c.style = c.style.Inherit(style.style)
	}
	return c
}

// Inherit returns a copy of the original style with the properties from a lipgloss.Style inherited.
// This follows lipgloss's inheritance behavior so margins, padding, and underlying string values are not inherited.
func (s Style) InheritLipgloss(styles ...lipgloss.Style) Style {
	c := s.Copy()
	for _, style := range styles {
		c.style = c.style.Inherit(style)
	}
	return c
}

// Sprintf formats the specified text with the style applied.
func (s Style) Sprintf(format string, a ...any) string {
	return s.style.Render(fmt.Sprintf(format, a...))
}

// Sprint applies the style to the specified text.
func (s Style) Sprint(str any) string {
	return s.style.Render(fmt.Sprint(str))
}

// S is shorthand for Sprint.
func (s Style) S(str any) string {
	return s.Sprint(str)
}

// Print applies the style to the specified text and prints it to the output writer.
func (s Style) Print(str any) (int, error) {
	return fmt.Fprint(s, str)
}

// Printf formats the specified text with the style applied and prints it to the output writer.
func (s Style) Printf(format string, a ...any) (int, error) {
	return fmt.Fprintf(s, format, a...)
}

// Write implements the io.Writer interface and prints to the output writer.
func (s Style) Write(b []byte) (n int, err error) {
	n = len(b)
	_, err = fmt.Fprint(s.writer(), s.Sprint(string(b)))
	return
}

// WithString returns a copy of the style with the string configured for the String() method.
func (s Style) WithString(str string) Style {
	c := s.Copy()
	c.style = c.style.SetString(str)
	return c
}

// WithOutput sets the output writer.
func (s Style) WithOutput(output io.Writer) Style {
	s.output = output
	return s
}

// String implements the fmt.Stringer interface.
func (s Style) String() string {
	return s.style.String()
}

func (s Style) writer() io.Writer {
	if s.output != nil {
		return s.output
	}
	return os.Stdout
}

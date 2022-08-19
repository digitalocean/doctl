package charm

import (
	"fmt"
	"os"

	"github.com/charmbracelet/lipgloss"
)

var (
	// ErrCancelled represents a user-initiated cancellation.
	ErrCancelled = fmt.Errorf("cancelled")
)

// Style is a styled component.
type Style struct {
	style lipgloss.Style
}

// NewStyle creates a new styled component.
func NewStyle(style lipgloss.Style) Style {
	return Style{style.Copy()}
}

// Lipgloss returns a copy of the underlying lipgloss.Style.
func (s Style) Lipgloss() lipgloss.Style {
	return s.style.Copy()
}

// Copy returns a copy of the style.
func (s Style) Copy() Style {
	return Style{style: s.style.Copy()}
}

// Inherit returns a copy of the original style with the properties from another style inherited.
// This follows lipgloss's inheritance behavior so margins, padding, and underlying string values are not inherited.
func (s Style) Inherit(o Style) Style {
	c := s.Copy()
	c.style = c.style.Inherit(o.style)
	return c
}

// NOTE is this actually needed
// // Inherit inherits properties from another style in-place.
// // This follows lipgloss's inheritance behavior so margins, padding, and underlying string values are not inherited.
// //
// // NOTE: this is an internal method that overwrites the original style.
// func (s *Style) inherit(o Style) *Style {
// 	s.style = s.style.Inherit(o.style)
// 	return s
// }

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

// Printf formats the specified text with the style applied and prints it to stdout.
func (s Style) Printf(format string, a ...any) (int, error) {
	return fmt.Fprint(os.Stdout, s.Sprintf(format, a...))
}

// Write implements the io.Writer interface and prints to stdout.
func (s Style) Write(b []byte) (n int, err error) {
	n = len(b)
	_, err = s.Printf(string(b))
	return
}

// WithString returns a copy of the style with the string configured for the String() method.
func (s Style) WithString(str string) Style {
	c := s.Copy()
	c.style = c.style.SetString(str)
	return c
}

// String implements the fmt.Stringer interface.
func (s Style) String() string {
	return s.style.String()
}

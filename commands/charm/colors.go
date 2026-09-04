package charm

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/digitalocean/doctl/internal/ui"
)

// ColorScheme describes a color scheme.
//
// The fields are TerminalColor rather than Color because internal/ui names the
// palette as slots in the terminal's own 16 colors, which lipgloss models as
// ANSIColor. Only the interface is wide enough to hold either.
type ColorScheme struct {
	Success   lipgloss.TerminalColor
	Warning   lipgloss.TerminalColor
	Error     lipgloss.TerminalColor
	Highlight lipgloss.TerminalColor
	Muted     lipgloss.TerminalColor
}

// Colors contains the default doctl color scheme.
var Colors = DefaultColorScheme()

// DefaultColorScheme returns doctl's default color scheme.
//
// internal/ui owns the palette; this is only the charm-shaped view of it, so
// interactive charm chrome and CLI error styling cannot drift apart. Whether
// these colors are emitted at all is decided by the process-wide lipgloss
// profile, which doctl points at the resolved ui.Env at startup.
func DefaultColorScheme() ColorScheme {
	return ColorScheme{
		Success:   ui.ColorSuccess,
		Warning:   ui.ColorWarning,
		Error:     ui.ColorError,
		Highlight: ui.ColorInfo,
		Muted:     ui.ColorMuted,
	}
}

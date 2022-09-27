package charm

import "github.com/charmbracelet/lipgloss"

// ColorScheme describes a color scheme.
type ColorScheme struct {
	Success   lipgloss.Color
	Warning   lipgloss.Color
	Error     lipgloss.Color
	Highlight lipgloss.Color
	Muted     lipgloss.Color
}

// Colors contains the default doctl color scheme.
var Colors = DefaultColorScheme()

// DefaultColorScheme returns doctl's default color scheme.
func DefaultColorScheme() ColorScheme {
	var (
		// TODO: adapt to light/dark color schemes.
		green  = lipgloss.Color("#04b575")
		yellow = lipgloss.Color("#ffd866")
		red    = lipgloss.Color("#ff6188")
		blue   = lipgloss.Color("#2ea0f9")
		muted  = lipgloss.Color("241")
	)

	return ColorScheme{
		Success:   green,
		Warning:   yellow,
		Error:     red,
		Highlight: blue,
		Muted:     muted,
	}
}

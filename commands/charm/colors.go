package charm

import "github.com/charmbracelet/lipgloss"

// ColorScheme describes a color scheme.
type ColorScheme struct {
	// TODO: do we actually need these explicit color names
	Green  lipgloss.Color
	Yellow lipgloss.Color
	Red    lipgloss.Color
	Blue   lipgloss.Color

	// aliases
	Success   lipgloss.Color
	Warning   lipgloss.Color
	Error     lipgloss.Color
	Highlight lipgloss.Color
}

// Colors contains the default doctl color scheme.
var Colors = DefaultColorScheme()

// DefaultColorScheme returns doctl's default color scheme.
func DefaultColorScheme() ColorScheme {
	c := ColorScheme{
		// TODO: check contrast w/ light and dark backgrounds.
		Green:  lipgloss.Color("#04b575"),
		Yellow: lipgloss.Color("#ffd866"),
		Red:    lipgloss.Color("#ff6188"),
		Blue:   lipgloss.Color("#2ea0f9"),
	}

	// aliases
	c.Success = c.Green
	c.Warning = c.Yellow
	c.Error = c.Red
	c.Highlight = c.Blue

	return c
}

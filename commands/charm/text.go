package charm

import (
	"github.com/charmbracelet/lipgloss"
)

var (
	TextSuccess   = Style{lipgloss.NewStyle().Foreground(Colors.Success)}
	TextWarning   = Style{lipgloss.NewStyle().Foreground(Colors.Warning)}
	TextError     = Style{lipgloss.NewStyle().Foreground(Colors.Error)}
	TextHighlight = Style{lipgloss.NewStyle().Foreground(Colors.Highlight)}

	TextBold      = Style{lipgloss.NewStyle().Bold(true)}
	TextUnderline = Style{lipgloss.NewStyle().Underline(true)}
)

const (
	// CheckmarkCharacter is the checkmark character.
	CheckmarkCharacter = "✓"
	// CrossmarkCharacter is the crossmark character.
	CrossmarkCharacter = "✘"
)

var (
	// Checkmark is a checkmark icon.
	Checkmark = Style{lipgloss.NewStyle().SetString(CheckmarkCharacter)}
	// CheckmarkSuccess is a green checkmark that implements fmt.Stringer.
	//
	// Example: fmt.Printf("%s success!", charm.CheckmarkSuccess)
	CheckmarkSuccess = Checkmark.Inherit(TextSuccess)

	// Crossmark is a crossmark icon.
	Crossmark = Style{lipgloss.NewStyle().SetString(CrossmarkCharacter)}
	// CrossmarkSuccess is a green crossmark that implements fmt.Stringer.
	//
	// Example: fmt.Printf("%s success!", charm.CrossmarkError)
	CrossmarkError = Crossmark.Inherit(TextError)
)

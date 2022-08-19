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
	// PromptPrefixCharacter is the > prompt character.
	PromptPrefixCharacter = "❯"
)

var (
	// Checkmark is a checkmark icon.
	Checkmark = Style{lipgloss.NewStyle().SetString(CheckmarkCharacter)}
	// CheckmarkSuccess is a green checkmark icon.
	CheckmarkSuccess = Checkmark.Inherit(TextSuccess)

	// Crossmark is a crossmark icon.
	Crossmark = Style{lipgloss.NewStyle().SetString(CrossmarkCharacter)}
	// CrossmarkSuccess is a green crossmark icon.
	CrossmarkError = Crossmark.Inherit(TextError)

	// PromptPrefix is a prompt-prefix icon.
	PromptPrefix = Style{lipgloss.NewStyle().SetString(PromptPrefixCharacter)}
	// PromptPrefixSuccess is a green prompt-prefix icon.
	PromptPrefixSuccess = PromptPrefix.Inherit(TextSuccess)
	// PromptPrefixError is a red prompt-prefix icon.
	PromptPrefixError = PromptPrefix.Inherit(TextError)
	// PromptPrefixHighlight is a highlighted prompt-prefix icon.
	PromptPrefixHighlight = PromptPrefix.Inherit(TextHighlight)
)

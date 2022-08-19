package charm

import (
	"github.com/charmbracelet/lipgloss"
)

var (
	TextSuccess   = Style{lipgloss.NewStyle().Foreground(Colors.Success)}
	TextWarning   = Style{lipgloss.NewStyle().Foreground(Colors.Warning)}
	TextError     = Style{lipgloss.NewStyle().Foreground(Colors.Error)}
	TextHighlight = Style{lipgloss.NewStyle().Foreground(Colors.Highlight)}
	TextMuted     = Style{lipgloss.NewStyle().Foreground(Colors.Muted)}

	TextBold      = Style{lipgloss.NewStyle().Bold(true)}
	TextUnderline = Style{lipgloss.NewStyle().Underline(true)}
)

var (
	// Checkmark is a checkmark icon.
	Checkmark = NewStyledText("✔")
	// Crossmark is a crossmark icon.
	Crossmark = NewStyledText("✘")
	// Astreisk is a astreisk icon.
	Astreisk = NewStyledText("✱")
	// PromptPrefix is a prompt-prefix icon.
	PromptPrefix = NewStyledText("❯")
	// PointerUpCharacter is an up pointer icon.
	PointerUp = NewStyledText("▴")
	// PointerRightCharacter is a right pointer icon.
	PointerRight = NewStyledText("▸")
	// PointerDownCharacter is a down pointer icon.
	PointerDown = NewStyledText("▾")
	// PointerLeftCharacter is a left pointer icon.
	PointerLeft = NewStyledText("◂")
)

type StyledText struct {
	style Style
}

// NewStyledText builds a new styled text component.
func NewStyledText(s string) StyledText {
	return StyledText{
		style: NewStyle(lipgloss.NewStyle().SetString(s)),
	}
}

func (t StyledText) String() string {
	return t.style.String()
}

func (t StyledText) Inherit(styles ...Style) StyledText {
	return StyledText{
		style: t.style.Copy().Inherit(styles...),
	}
}

func (t StyledText) Success() StyledText {
	return t.Inherit(TextSuccess)
}

func (t StyledText) Warning() StyledText {
	return t.Inherit(TextWarning)
}

func (t StyledText) Error() StyledText {
	return t.Inherit(TextError)
}

func (t StyledText) Highlight() StyledText {
	return t.Inherit(TextHighlight)
}

func (t StyledText) Muted() StyledText {
	return t.Inherit(TextMuted)
}

package text

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/digitalocean/doctl/commands/charm"
)

var (
	Success   = charm.NewStyle(lipgloss.NewStyle().Foreground(charm.Colors.Success))
	Warning   = charm.NewStyle(lipgloss.NewStyle().Foreground(charm.Colors.Warning))
	Error     = charm.NewStyle(lipgloss.NewStyle().Foreground(charm.Colors.Error))
	Highlight = charm.NewStyle(lipgloss.NewStyle().Foreground(charm.Colors.Highlight))
	Muted     = charm.NewStyle(lipgloss.NewStyle().Foreground(charm.Colors.Muted))

	Bold      = charm.NewStyle(lipgloss.NewStyle().Bold(true))
	Underline = charm.NewStyle(lipgloss.NewStyle().Underline(true))
)

var (
	// Checkmark is a checkmark icon.
	Checkmark = NewStyled("✔")
	// Crossmark is a crossmark icon.
	Crossmark = NewStyled("✘")
	// Astreisk is a astreisk icon.
	Astreisk = NewStyled("✱")
	// PromptPrefix is a prompt-prefix icon.
	PromptPrefix = NewStyled("❯")
	// PointerUpCharacter is an up pointer icon.
	PointerUp = NewStyled("▴")
	// PointerRightCharacter is a right pointer icon.
	PointerRight = NewStyled("▸")
	// PointerDownCharacter is a down pointer icon.
	PointerDown = NewStyled("▾")
	// PointerLeftCharacter is a left pointer icon.
	PointerLeft = NewStyled("◂")
)

type Styled struct {
	style charm.Style
}

// NewStyled builds a new styled text component.
func NewStyled(s string) Styled {
	return Styled{
		style: charm.NewStyle(lipgloss.NewStyle().SetString(s)),
	}
}

func (t Styled) String() string {
	return t.style.String()
}

func (t Styled) Inherit(styles ...charm.Style) Styled {
	return Styled{
		style: t.style.Copy().Inherit(styles...),
	}
}

func (t Styled) Success() Styled {
	return t.Inherit(Success)
}

func (t Styled) Warning() Styled {
	return t.Inherit(Warning)
}

func (t Styled) Error() Styled {
	return t.Inherit(Error)
}

func (t Styled) Highlight() Styled {
	return t.Inherit(Highlight)
}

func (t Styled) Muted() Styled {
	return t.Inherit(Muted)
}

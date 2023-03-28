package textbox

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/digitalocean/doctl/commands/charm"
)

// TextBox is a text box
type TextBox struct {
	charm.Style
}

func New() *TextBox {
	return &TextBox{
		Style: charm.NewStyle(
			lipgloss.NewStyle().
				Padding(1, 2).
				BorderStyle(lipgloss.RoundedBorder()).
				Margin(1),
		),
	}
}

func (t TextBox) Success() TextBox {
	t.Style = t.Style.InheritLipgloss(lipgloss.NewStyle().BorderForeground(charm.Colors.Success))
	return t
}

func (t TextBox) Error() TextBox {
	t.Style = t.Style.InheritLipgloss(lipgloss.NewStyle().BorderForeground(charm.Colors.Error))
	return t
}

func (t TextBox) Warning() TextBox {
	t.Style = t.Style.InheritLipgloss(lipgloss.NewStyle().BorderForeground(charm.Colors.Warning))
	return t
}

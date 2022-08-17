package charm

import (
	"github.com/charmbracelet/lipgloss"
)

// TextBox
type TextBox struct {
	Style
}

func NewTextBox() TextBox {
	return TextBox{
		Style: Style{
			lipgloss.NewStyle().
				Padding(1, 2).
				BorderStyle(lipgloss.RoundedBorder()).
				Margin(1),
		},
	}
}

func (t TextBox) Success() TextBox {
	t.style.BorderForeground(Colors.Success)
	return t
}

func (t TextBox) Error() TextBox {
	t.style.BorderForeground(Colors.Error)
	return t
}

package charm

import (
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/paginator"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	// ListDefaultStyle is the default list style.
	ListDefaultStyle = Style{lipgloss.NewStyle().Margin(1, 1)}
)

// List is a component used to select an item from a list.
type List struct {
	Fullscreen bool
	Style      Style

	items    []list.Item
	selected list.Item

	model list.Model
}

// listModel implements the bubbletea.Model interface.
type listModel List

// NewList creates a new list.
func NewList(items []list.Item) *List {
	delegate := list.NewDefaultDelegate()
	// TODO: accept an optional sample item for the height
	delegate.SetHeight(3)

	model := list.New(items, delegate, 0, 0)
	model.Paginator.Type = paginator.Arabic
	model.Paginator.ArabicFormat = "page %d of %d"

	return &List{
		model: model,
		Style: ListDefaultStyle,
	}
}

// Model returns the underlying list.Model.
func (l *List) Model() *list.Model {
	return &l.model
}

func (l *List) teaOptions() (opts []tea.ProgramOption) {
	if l.Fullscreen {
		opts = append(opts, tea.WithAltScreen())
	}
	return opts
}

// Select displays the list and prompts the user to select an item.
func (l *List) Select() (list.Item, error) {
	p := tea.NewProgram((*listModel)(l), l.teaOptions()...)
	if err := p.Start(); err != nil {
		return nil, err
	}
	return l.selected, nil
}

// Init implements bubbletea.Model.
func (l *listModel) Init() tea.Cmd {
	return nil
}

// Update implements bubbletea.Model.
func (l *listModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch keypress := msg.String(); keypress {
		case "ctrl+c", "esc":
			return l, tea.Quit

		case "enter":
			l.selected = l.model.SelectedItem()
			return l, tea.Quit
		}
	case tea.WindowSizeMsg:
		w, h := l.Style.style.GetFrameSize()
		w = msg.Width - w
		if l.Fullscreen {
			h = msg.Height - h
		} else {
			// TODO: finish this
			h = 10
		}
		l.model.SetSize(w, h)
	}

	var cmd tea.Cmd
	l.model, cmd = l.model.Update(msg)
	return l, cmd
}

// Update implements bubbletea.Model.
func (l *listModel) View() string {
	return l.Style.style.Render(l.model.View())
}

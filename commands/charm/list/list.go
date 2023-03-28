package list

import (
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/paginator"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/digitalocean/doctl/commands/charm"
)

var (
	// ListDefaultStyle is the default list style.
	ListDefaultStyle = charm.NewStyle(lipgloss.NewStyle().Margin(1, 1))
)

type Item list.Item

// List is a component used to select an item from a list.
type List struct {
	fullscreen bool
	style      charm.Style

	items    []Item
	selected Item

	model list.Model
}

// listModel implements the bubbletea.Model interface.
type listModel List

type Option func(*List)

// New creates a new list.
func New(items []Item, opts ...Option) *List {
	delegate := list.NewDefaultDelegate()
	// TODO: accept an optional sample item for the height
	delegate.SetHeight(3)

	teaItems := make([]list.Item, len(items))
	for i, item := range items {
		teaItems[i] = list.Item(item)
	}
	model := list.New(teaItems, delegate, 0, 0)
	model.Paginator.Type = paginator.Arabic
	model.Paginator.ArabicFormat = "page %d of %d"

	l := &List{
		model:      model,
		style:      ListDefaultStyle,
		fullscreen: true,
	}

	for _, opt := range opts {
		opt(l)
	}

	return l
}

func WithStyle(s charm.Style) Option {
	return func(l *List) {
		l.style = s
	}
}

// TODO: export once this is fully implemented.
func withFullscreen(v bool) Option {
	return func(l *List) {
		l.fullscreen = v
	}
}

// Model returns the underlying list.Model.
func (l *List) Model() *list.Model {
	return &l.model
}

func (l *List) teaOptions() (opts []tea.ProgramOption) {
	if l.fullscreen {
		opts = append(opts, tea.WithAltScreen())
	}
	return opts
}

// Select displays the list and prompts the user to select an item.
func (l *List) Select() (Item, error) {
	p := tea.NewProgram((*listModel)(l), l.teaOptions()...)
	if err := p.Start(); err != nil {
		return nil, err
	}
	if l.selected == nil {
		return nil, fmt.Errorf("canceled")
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
		w, h := l.style.Lipgloss().GetFrameSize()
		w = msg.Width - w
		if l.fullscreen {
			h = msg.Height - h
		} else {
			// TODO: what should we set this to
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
	return l.style.Lipgloss().Render(l.model.View())
}

// Items converts a slice of items into a []Item type slice.
func Items[T interface{ Item }](items []T) []Item {
	l := make([]Item, len(items))
	for i, item := range items {
		l[i] = Item(item)
	}
	return l
}

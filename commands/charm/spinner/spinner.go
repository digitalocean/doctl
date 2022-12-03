package spinner

import (
	"fmt"
	"os"

	s "github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

type SpinningLoader struct {
	model   s.Model
	prog    *tea.Program
	cancel  bool
	message string
}

type Option func(*SpinningLoader)

// New creates a new spinning loader.
func New(message string, opts ...Option) SpinningLoader {
	sm := s.New()
	sm.Spinner = s.Dot

	l := SpinningLoader{
		model:   sm,
		message: message,
	}

	for _, opt := range opts {
		opt(&l)
	}
	return l
}

// New creates a new spinner.
func (sl *SpinningLoader) Start() error {
	p := tea.NewProgram((*SpinningLoader)(sl))
	sl.prog = p

	if err := p.Start(); err != nil {
		return err
	}

	if sl.cancel {
		os.Exit(1)
	}
	return nil
}

func (sl *SpinningLoader) Stop() {
	if sl.prog != nil {
		sl.prog.Kill()
	}
}

// Init implements bubbletea.Model.
func (sl *SpinningLoader) Init() tea.Cmd {
	return sl.model.Tick
}

// Update implements bubbletea.Model.
func (sl *SpinningLoader) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch keypress := msg.String(); keypress {
		case tea.KeyCtrlC.String():
			sl.cancel = true
			return sl, tea.Quit
		}

	case s.TickMsg:
		var cmd tea.Cmd
		sl.model, cmd = sl.model.Update(msg)
		return sl, cmd
	}

	return sl, nil
}

// View implements bubbletea.Model.
func (sl *SpinningLoader) View() string {
	return fmt.Sprintf("%s %s", sl.model.View(), sl.message)
}

// Model returns the underlying SpinningLoader.model
func (sl SpinningLoader) Model() *s.Model {
	return &sl.model
}

func WithSpinner(s s.Spinner) Option {
	return func(sl *SpinningLoader) {
		sl.model.Spinner = s
	}
}

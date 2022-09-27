package selection

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/erikgeiser/promptkit"
	"github.com/muesli/termenv"
)

// Model implements the bubbletea.Model for a selection prompt.
type Model[T any] struct {
	*Selection[T]

	// Err holds errors that may occur during the execution of
	// the selection prompt.
	Err error

	// MaxWidth limits the width of the view using the Selection's WrapMode.
	MaxWidth int

	filterInput textinput.Model
	// currently displayed choices, after filtering and pagination
	currentChoices []*Choice[T]
	// number of available choices after filtering
	availableChoices int
	// index of current selection in currentChoices slice
	currentIdx        int
	scrollOffset      int
	width             int
	height            int
	tmpl              *template.Template
	resultTmpl        *template.Template
	requestedPageSize int

	quitting bool
}

// ensure that the Model interface is implemented.
var _ tea.Model = &Model[any]{}

// NewModel returns a new selection prompt model for the
// provided choices.
func NewModel[T any](selection *Selection[T]) *Model[T] {
	return &Model[T]{Selection: selection}
}

// Init initializes the selection prompt model.
func (m *Model[T]) Init() tea.Cmd {
	m.reindexChoices()

	if len(m.choices) == 0 {
		m.Err = fmt.Errorf("no choices provided")

		return tea.Quit
	}

	if m.ColorProfile == 0 {
		m.ColorProfile = termenv.ColorProfile()
	}

	if m.Template == "" {
		m.Err = fmt.Errorf("empty template")

		return tea.Quit
	}

	m.tmpl, m.Err = m.initTemplate()
	if m.Err != nil {
		return tea.Quit
	}

	m.resultTmpl, m.Err = m.initResultTemplate()
	if m.Err != nil {
		return tea.Quit
	}

	m.filterInput = m.initFilterInput()

	m.currentChoices, m.availableChoices = m.filteredAndPagedChoices()

	m.requestedPageSize = m.PageSize

	return textinput.Blink
}

func (m *Model[T]) initTemplate() (*template.Template, error) {
	tmpl := template.New("view")
	tmpl.Funcs(termenv.TemplateFuncs(m.ColorProfile))
	tmpl.Funcs(m.ExtendedTemplateFuncs)
	tmpl.Funcs(promptkit.UtilFuncMap())
	tmpl.Funcs(template.FuncMap{
		"IsScrollDownHintPosition": func(idx int) bool {
			return m.canScrollDown() && (idx == len(m.currentChoices)-1)
		},
		"IsScrollUpHintPosition": func(idx int) bool {
			return m.canScrollUp() && idx == 0 && m.scrollOffset > 0
		},
		"Selected": func(c *Choice[T]) string {
			if m.SelectedChoiceStyle == nil {
				return c.String
			}

			return m.SelectedChoiceStyle(c)
		},
		"Unselected": func(c *Choice[T]) string {
			if m.UnselectedChoiceStyle == nil {
				return c.String
			}

			return m.UnselectedChoiceStyle(c)
		},
	})

	return tmpl.Parse(m.Template)
}

func (m *Model[T]) initResultTemplate() (*template.Template, error) {
	if m.ResultTemplate == "" {
		return nil, nil // nolint:nilnil
	}

	tmpl := template.New("result")
	tmpl.Funcs(termenv.TemplateFuncs(m.ColorProfile))
	tmpl.Funcs(m.ExtendedTemplateFuncs)
	tmpl.Funcs(promptkit.UtilFuncMap())
	tmpl.Funcs(template.FuncMap{
		"Final": func(c *Choice[T]) string {
			if m.FinalChoiceStyle == nil {
				return c.String
			}

			return m.FinalChoiceStyle(c)
		},
	})

	return tmpl.Parse(m.ResultTemplate)
}

func (m *Model[T]) initFilterInput() textinput.Model {
	filterInput := textinput.NewModel()
	filterInput.Prompt = ""
	filterInput.TextStyle = m.FilterInputTextStyle
	filterInput.BackgroundStyle = m.FilterInputBackgroundStyle
	filterInput.PlaceholderStyle = m.FilterInputPlaceholderStyle
	filterInput.CursorStyle = m.FilterInputCursorStyle
	filterInput.Placeholder = m.FilterPlaceholder
	filterInput.Width = 80
	filterInput.Focus()

	return filterInput
}

func (m *Model[T]) ValueAsChoice() (*Choice[T], error) {
	if m.Err != nil {
		return nil, m.Err
	}

	if len(m.currentChoices) == 0 {
		return nil, fmt.Errorf("no choices")
	}

	if m.currentIdx < 0 || m.currentIdx >= len(m.currentChoices) {
		return nil, fmt.Errorf("choice index out of bounds")
	}

	return m.currentChoices[m.currentIdx], nil
}

// Value returns the choice that is currently selected or the final
// choice after the prompt has concluded.
func (m *Model[T]) Value() (T, error) {
	choice, err := m.ValueAsChoice()
	if err != nil {
		var zeroValue T

		return zeroValue, err
	}

	return choice.Value, nil
}

// Update updates the model based on the received message.
func (m *Model[T]) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.Err != nil {
		return m, tea.Quit
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case keyMatches(msg, m.KeyMap.Abort):
			m.Err = promptkit.ErrAborted
			m.quitting = true

			return m, tea.Quit
		case keyMatches(msg, m.KeyMap.Select):
			if len(m.currentChoices) == 0 {
				return m, nil
			}

			m.quitting = true

			return m, tea.Quit
		case keyMatches(msg, m.KeyMap.ClearFilter):
			m.filterInput.Reset()
			m.currentChoices, m.availableChoices = m.filteredAndPagedChoices()
		case keyMatches(msg, m.KeyMap.Down):
			m.cursorDown()
		case keyMatches(msg, m.KeyMap.Up):
			m.cursorUp()
		case keyMatches(msg, m.KeyMap.ScrollDown):
			m.scrollDown()
		case keyMatches(msg, m.KeyMap.ScrollUp):
			m.scrollUp()
		default:
			return m.updateFilter(msg)
		}

		return m, nil
	case tea.WindowSizeMsg:
		m.width = zeroAwareMin(msg.Width, m.MaxWidth)

		if m.height != msg.Height {
			m.height = msg.Height
			m.forceUpdatePageSizeForHeight()
		}

		return m, nil
	case error:
		m.Err = msg

		return m, tea.Quit
	}

	var cmd tea.Cmd

	return m, cmd
}

func (m *Model[T]) forceUpdatePageSizeForHeight() {
	if m.requestedPageSize == 0 {
		m.PageSize = len(m.choices)
	} else {
		m.PageSize = min(len(m.choices), m.requestedPageSize)
	}

	for m.PageSize > 1 {
		m.currentIdx = 0
		m.scrollOffset = 0
		m.currentChoices, m.availableChoices = m.filteredAndPagedChoices()

		if len(strings.Split(m.View(), "\n")) <= m.height {
			break
		}

		m.PageSize--
	}
}

func (m *Model[T]) updateFilter(msg tea.Msg) (*Model[T], tea.Cmd) {
	if m.Filter == nil {
		return m, nil
	}

	previousFilter := m.filterInput.Value()

	var cmd tea.Cmd
	m.filterInput, cmd = m.filterInput.Update(msg)

	if m.filterInput.Value() != previousFilter {
		m.currentIdx = 0
		m.scrollOffset = 0
		m.currentChoices, m.availableChoices = m.filteredAndPagedChoices()
	}

	return m, cmd
}

// View renders the selection prompt.
func (m *Model[T]) View() string {
	viewBuffer := &bytes.Buffer{}

	if m.quitting {
		view, err := m.resultView()
		if err != nil {
			m.Err = err

			return ""
		}

		return m.wrap(view)
	}

	// avoid panics if Quit is sent during Init
	if m.tmpl == nil {
		return ""
	}

	err := m.tmpl.Execute(viewBuffer, map[string]interface{}{
		"Prompt":        m.Prompt,
		"IsFiltered":    m.Filter != nil,
		"FilterPrompt":  m.FilterPrompt,
		"FilterInput":   m.filterInput.View(),
		"Choices":       m.currentChoices,
		"NChoices":      len(m.currentChoices),
		"SelectedIndex": m.currentIdx,
		"PageSize":      m.PageSize,
		"IsPaged":       m.PageSize > 0 && len(m.currentChoices) > m.PageSize,
		"AllChoices":    m.choices,
		"NAllChoices":   len(m.choices),
		"TerminalWidth": m.width,
	})
	if err != nil {
		m.Err = err

		return "Template Error: " + err.Error()
	}

	return m.wrap(viewBuffer.String())
}

func (m *Model[T]) resultView() (string, error) {
	viewBuffer := &bytes.Buffer{}

	if m.ResultTemplate == "" {
		return "", nil
	}

	if m.resultTmpl == nil {
		return "", fmt.Errorf("rendering confirmation without loaded template")
	}

	choice, err := m.ValueAsChoice()
	if err != nil {
		return "", err
	}

	err = m.resultTmpl.Execute(viewBuffer, map[string]interface{}{
		"FinalChoice":   choice,
		"Prompt":        m.Prompt,
		"AllChoices":    m.choices,
		"NAllChoices":   len(m.choices),
		"TerminalWidth": m.width,
	})
	if err != nil {
		return "", fmt.Errorf("execute confirmation template: %w", err)
	}

	return viewBuffer.String(), nil
}

func (m *Model[T]) wrap(text string) string {
	if m.WrapMode == nil {
		return text
	}

	return m.WrapMode(text, m.width)
}

func (m *Model[T]) filteredAndPagedChoices() ([]*Choice[T], int) {
	choices := []*Choice[T]{}

	var available, ignored int

	for _, choice := range m.choices {
		if m.Filter != nil && !m.Filter(m.filterInput.Value(), choice) {
			continue
		}

		available++

		if m.PageSize > 0 && (len(choices) >= m.PageSize || ignored < m.scrollOffset) {
			ignored++

			continue
		}

		choices = append(choices, choice)
	}

	return choices, available
}

func (m *Model[T]) canScrollDown() bool {
	if m.PageSize <= 0 || m.availableChoices <= m.PageSize {
		return false
	}

	if m.scrollOffset+m.PageSize >= len(m.choices) {
		return false
	}

	return true
}

func (m *Model[T]) canScrollUp() bool {
	return m.scrollOffset > 0
}

func (m *Model[T]) cursorDown() {
	if m.currentIdx == len(m.currentChoices)-1 {
		if m.canScrollDown() {
			m.scrollDown()
		} else if m.LoopCursor {
			m.scrollToTop()

			return
		}
	}

	m.currentIdx = min(len(m.currentChoices)-1, m.currentIdx+1)
}

func (m *Model[T]) cursorUp() {
	if m.currentIdx == 0 {
		if m.canScrollUp() {
			m.scrollUp()
		} else if m.LoopCursor {
			m.scrollToBottom()

			return
		}
	}

	m.currentIdx = max(0, m.currentIdx-1)
}

func (m *Model[T]) scrollDown() {
	if m.PageSize <= 0 || m.scrollOffset+m.PageSize >= m.availableChoices {
		return
	}

	m.currentIdx = max(0, m.currentIdx-1)
	m.scrollOffset++
	m.currentChoices, m.availableChoices = m.filteredAndPagedChoices()
}

func (m *Model[T]) scrollToBottom() {
	m.currentIdx = len(m.currentChoices) - 1

	if m.PageSize <= 0 || m.availableChoices < m.PageSize {
		return
	}

	m.scrollOffset = m.availableChoices - m.PageSize
	m.currentChoices, m.availableChoices = m.filteredAndPagedChoices()
}

func (m *Model[T]) scrollUp() {
	if m.PageSize <= 0 || m.scrollOffset <= 0 {
		return
	}

	m.currentIdx = min(len(m.currentChoices)-1, m.currentIdx+1)
	m.scrollOffset--
	m.currentChoices, m.availableChoices = m.filteredAndPagedChoices()
}

func (m *Model[T]) scrollToTop() {
	m.currentIdx = 0

	if m.PageSize <= 0 || m.availableChoices < m.PageSize {
		return
	}

	m.scrollOffset = 0
	m.currentChoices, m.availableChoices = m.filteredAndPagedChoices()
}

func (m *Model[T]) reindexChoices() {
	for i, choice := range m.choices {
		choice.idx = i
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}

	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}

	return b
}

func zeroAwareMin(a int, b int) int {
	switch {
	case a == 0:
		return b
	case b == 0:
		return a
	default:
		return min(a, b)
	}
}

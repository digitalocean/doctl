/*
Package selection implements a selection prompt that allows users to to select
one of the pre-defined choices. It also offers customizable appreance and key
map as well as optional support for pagination, filtering.
*/
package selection

import (
	"fmt"
	"io"
	"os"
	"strings"
	"text/template"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/erikgeiser/promptkit"
	"github.com/muesli/termenv"
)

const (
	// DefaultTemplate defines the default appearance of the selection and can
	// be copied as a starting point for a custom template.
	DefaultTemplate = `
{{- if .Prompt -}}
  {{ Bold .Prompt }}
{{ end -}}
{{ if .IsFiltered }}
  {{- print .FilterPrompt " " .FilterInput }}
{{ end }}

{{- range  $i, $choice := .Choices }}
  {{- if IsScrollUpHintPosition $i }}
    {{- "⇡ " -}}
  {{- else if IsScrollDownHintPosition $i -}}
    {{- "⇣ " -}}
  {{- else -}}
    {{- "  " -}}
  {{- end -}}

  {{- if eq $.SelectedIndex $i }}
   {{- print (Foreground "32" (Bold "▸ ")) (Selected $choice) "\n" }}
  {{- else }}
    {{- print "  " (Unselected $choice) "\n" }}
  {{- end }}
{{- end}}`

	// DefaultResultTemplate defines the default appearance with which the
	// finale result of the selection is presented.
	DefaultResultTemplate = `
	{{- print .Prompt " " (Final .FinalChoice) "\n" -}}
	`

	// DefaultFilterPrompt is the default prompt for the filter input when
	// filtering is enabled.
	DefaultFilterPrompt = "Filter:"

	// DefaultFilterPlaceholder is printed by default when no filter text was
	// entered yet.
	DefaultFilterPlaceholder = "Type to filter choices"

	accentColor = termenv.ANSI256Color(32)
)

// DefaultSelectedChoiceStyle is the default style for selected choices.
func DefaultSelectedChoiceStyle[T any](c *Choice[T]) string {
	return termenv.String(c.String).Foreground(accentColor).Bold().String()
}

// DefaultFinalChoiceStyle is the default style for final choices.
func DefaultFinalChoiceStyle[T any](c *Choice[T]) string {
	return termenv.String(c.String).Foreground(accentColor).String()
}

// Selection represents a configurable selection prompt.
type Selection[T any] struct {
	// choices represent all selectable choices of the selection. Slices of
	// arbitrary types can be converted to a slice of choices using the helper
	// selection.choices.
	choices []*Choice[T]

	// Prompt holds the the prompt text or question that is to be answered by
	// one of the choices.
	Prompt string

	// FilterPrompt is the prompt for the filter if filtering is enabled.
	FilterPrompt string

	// Filter is a function that decides whether a given choice should be
	// displayed based on the text entered by the user into the filter input
	// field. If Filter is nil, filtering will be disabled. By default the
	// filter FilterContainsCaseInsensitive is used.
	Filter func(filterText string, choice *Choice[T]) bool

	// FilterPlaceholder holds the text that is displayed in the filter input
	// field when no text was entered by the user yet. If empty, the
	// DefaultFilterPlaceholder is used. If Filter is nil, filtering is disabled
	// and FilterPlaceholder does nothing.
	FilterPlaceholder string

	// PageSize is the number of choices that are displayed at once. If PageSize
	// is smaller than the number of choices, pagination is enabled. If PageSize
	// is 0, pagenation is disabled. Regardless of the value of PageSize,
	// pagination is always enabled when the prompt does not fit the terminal.
	PageSize int

	// LoopCursor enables the cursor to loop around to the first choice when
	// navigating down from the last choice and the other way around.
	LoopCursor bool

	// Template holds the display template. A custom template can be used to
	// completely customize the appearance of the selection prompt. If empty,
	// the DefaultTemplate is used. The following variables and functions are
	// available:
	//
	//  * Prompt string: The configured prompt.
	//  * IsFiltered bool: Whether or not filtering is enabled.
	//  * FilterPrompt string: The configured filter prompt.
	//  * FilterInput string: The view of the filter input model.
	//  * Choices []*Choice: The choices on the current page.
	//  * NChoices int: The number of choices on the current page.
	//  * SelectedIndex int: The index that is currently selected.
	//  * PageSize int: The configured page size.
	//  * IsPaged bool: Whether pagination is currently active.
	//  * AllChoices []*Choice: All configured choices.
	//  * NAllChoices int: The number of configured choices.
	//  * TerminalWidth int: The width of the terminal.
	//  * Selected(*Choice) string: The configured SelectedChoiceStyle.
	//  * Unselected(*Choice) string: The configured UnselectedChoiceStyle.
	//  * IsScrollDownHintPosition(idx int) bool: Returns whether
	//    the scroll down hint shoud be displayed at the given index.
	//  * IsScrollUpHintPosition(idx int) bool: Returns whether the
	//    scroll up hint shoud be displayed at the given index).
	//  * promptkit.UtilFuncMap: Handy helper functions.
	//  * termenv TemplateFuncs (see https://github.com/muesli/termenv).
	//  * The functions specified in ExtendedTemplateFuncs.
	Template string

	// ResultTemplate is rendered as soon as a choice has been selected.
	// It is intended to permanently indicate the result of the prompt when the
	// selection itself has disappeared. This template is only rendered in the
	// Run() method and NOT when the selection prompt is used as a model. The
	// following variables and functions are available:
	//
	//  * FinalChoice: The choice that was selected by the user.
	//  * Prompt string: The configured prompt.
	//  * AllChoices []*Choice: All configured choices.
	//  * NAllChoices int: The number of configured choices.
	//  * TerminalWidth int: The width of the terminal.
	//  * Final(*Choice) string: The configured FinalChoiceStyle.
	//  * promptkit.UtilFuncMap: Handy helper functions.
	//  * termenv TemplateFuncs (see https://github.com/muesli/termenv).
	//  * The functions specified in ExtendedTemplateFuncs.
	ResultTemplate string

	// ExtendedTemplateFuncs can be used to add additional functions to the
	// evaluation scope of the templates.
	ExtendedTemplateFuncs template.FuncMap

	// Styles of the filter input field. These will be applied as inline styles.
	//
	// For an introduction to styling with Lip Gloss see:
	// https://github.com/charmbracelet/lipgloss
	FilterInputTextStyle        lipgloss.Style
	FilterInputBackgroundStyle  lipgloss.Style
	FilterInputPlaceholderStyle lipgloss.Style
	FilterInputCursorStyle      lipgloss.Style

	// SelectedChoice style allows to customize the appearance of the currently
	// selected choice. By default DefaultSelectedChoiceStyle is used. If it is
	// nil, no style will be applied and the plain string representation of the
	// choice will be used. This style will be available as the template
	// function Selected. Custom templates may or may not use this function.
	SelectedChoiceStyle func(*Choice[T]) string

	// UnselectedChoiceStyle style allows to customize the appearance of the
	// currently unselected choice. By default it is nil, such that no style
	// will be applied and the plain string representation of the choice will be
	// used. This style will be available as the template function Unselected.
	// Custom templates may or may not use this function.
	UnselectedChoiceStyle func(*Choice[T]) string

	// FinalChoiceStyle style allows to customize the appearance of the choice
	// that was ultimately chosen. By default DefaultFinalChoiceStyle is used.
	// If it is nil, no style will be applied and the plain string
	// representation of the choice will be used. This style will be available
	// as the template function Final. Custom templates may or may not use this
	// function.
	FinalChoiceStyle func(*Choice[T]) string

	// KeyMap determines with which keys the selection prompt is controlled. By
	// default, DefaultKeyMap is used.
	KeyMap *KeyMap

	// WrapMode decides which way the prompt view is wrapped if it does not fit
	// the terminal. It can be a WrapMode provided by promptkit or a custom
	// function. By default it is promptkit.WordWrap. It can also be nil which
	// disables wrapping and likely causes output glitches.
	WrapMode promptkit.WrapMode

	// Output is the output writer, by default os.Stdout is used.
	Output io.Writer
	// Input is the input reader, by default, os.Stdin is used.
	Input io.Reader

	// ColorProfile determines how colors are rendered. By default, the terminal
	// is queried.
	ColorProfile termenv.Profile
}

// New creates a new selection prompt. See the Selection properties for more
// documentation.
func New[T any](prompt string, choices []T) *Selection[T] {
	return &Selection[T]{
		choices:                     asChoices(choices),
		Prompt:                      prompt,
		FilterPrompt:                DefaultFilterPrompt,
		Template:                    DefaultTemplate,
		ResultTemplate:              DefaultResultTemplate,
		Filter:                      FilterContainsCaseInsensitive[T],
		FilterInputPlaceholderStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("240")),
		SelectedChoiceStyle:         DefaultSelectedChoiceStyle[T],
		FinalChoiceStyle:            DefaultFinalChoiceStyle[T],
		KeyMap:                      NewDefaultKeyMap(),
		FilterPlaceholder:           DefaultFilterPlaceholder,
		ExtendedTemplateFuncs:       template.FuncMap{},
		WrapMode:                    promptkit.Truncate,
		Output:                      os.Stdout,
		Input:                       os.Stdin,
	}
}

// RunPrompt executes the selection prompt.
func (s *Selection[T]) RunPrompt() (T, error) {
	var zeroValue T

	err := validateKeyMap(s.KeyMap)
	if err != nil {
		return zeroValue, fmt.Errorf("insufficient key map: %w", err)
	}

	m := NewModel(s)

	p := tea.NewProgram(m, tea.WithOutput(s.Output), tea.WithInput(s.Input))
	if err := p.Start(); err != nil {
		return zeroValue, fmt.Errorf("running prompt: %w", err)
	}

	return m.Value()
}

// FilterContainsCaseInsensitive returns true if the string representation of
// the choice contains the filter string without regard for capitalization.
func FilterContainsCaseInsensitive[T any](filter string, choice *Choice[T]) bool {
	return strings.Contains(strings.ToLower(choice.String), strings.ToLower(filter))
}

// FilterContainsCaseSensitive returns true if the string representation of the
// choice contains the filter string respecting capitalization.
func FilterContainsCaseSensitive[T any](filter string, choice *Choice[T]) bool {
	return strings.Contains(choice.String, filter)
}

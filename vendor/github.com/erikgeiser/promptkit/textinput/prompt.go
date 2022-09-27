/*
Package textinput implements prompt for a string input that can also be used for
secret strings such as passwords. It also offers customizable appreance as well
as optional support for input validation and a customizable key map.
*/
package textinput

import (
	"fmt"
	"io"
	"os"
	"text/template"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/erikgeiser/promptkit"
	"github.com/muesli/termenv"
)

const (
	// DefaultTemplate defines the default appearance of the text input and can
	// be copied as a starting point for a custom template.
	DefaultTemplate = `
	{{- Bold .Prompt }} {{ .Input -}}
	{{- if .ValidationError }} {{ Foreground "1" (Bold "✘") }}
	{{- else }} {{ Foreground "2" (Bold "✔") }}
	{{- end -}}
	`

	// DefaultResultTemplate defines the default appearance with which the
	// finale result of the prompt is presented.
	DefaultResultTemplate = `
	{{- print .Prompt " " (Foreground "32"  (Mask .FinalValue)) "\n" -}}
	`

	// DefaultMask specified the character with which the input is masked by
	// default if Hidden is true.
	DefaultMask = '●'
)

// ErrInputValidation is a generic input validation error. For more detailed
// diagnosis, feel free to return any custom error instead.
var ErrInputValidation = fmt.Errorf("validation error")

// TextInput represents a configurable selection prompt.
type TextInput struct {
	// Prompt holds the the prompt text or question that is printed above the
	// choices in the default template (if not empty).
	Prompt string

	// Placeholder holds the text that is displayed in the input field when the
	// input data is empty, e.g. when no text was entered yet.
	Placeholder string

	// InitialValue is similar to Placeholder, however, the actual input data is
	// set to InitialValue such that as if it was entered by the user. This can
	// be used to provide an editable default value.
	InitialValue string

	// Validate is a function that validates whether the current input data is
	// valid. If it is not, the data cannot be submitted. By default, Validate
	// ensures that the input data is not empty. If Validate is set to nil, no
	// validation is performed.
	Validate func(string) error

	// AutoComplete is a function that suggests multiple candidates for
	// auto-completion based on a given input. If it returns only a single
	// candidate, this candidate is auto-completed. If it returns multiple
	// candidates, these candidates may be displayed in custom templates using
	// the variables AutoCompleteTriggered, AutoCompleteIndecisive as well as
	// the function AutoCompleteSuggestions. If AutoComplete is nil, no
	// auto-completion is performed.
	AutoComplete func(string) []string

	// Hidden specified whether or not the input data is considered secret and
	// should be masked. This is useful for password prompts.
	Hidden bool

	// HideMask specified the character with which the input data should be
	// masked when Hidden is set to true.
	HideMask rune

	// CharLimit is the maximum amount of characters this input element will
	// accept. If 0 or less, there's no limit.
	CharLimit int

	// InputWidth is the maximum number of characters that can be displayed at
	// once. It essentially treats the text field like a horizontally scrolling
	// viewport. If 0 or less this setting is ignored.
	InputWidth int

	// Template holds the display template. A custom template can be used to
	// completely customize the appearance of the text input. If empty,
	// the DefaultTemplate is used. The following variables and functions are
	// available:
	//
	//  * Prompt string: The configured prompt.
	//  * InitialValue string: The configured initial value of the input.
	//  * Placeholder string: The configured placeholder of the input.
	//  * Input string: The actual input field.
	//  * ValidationError error: The error value returned by Validate.
	//    to the configured Validate function.
	//  * TerminalWidth int: The width of the terminal.
	//  * promptkit.UtilFuncMap: Handy helper functions.
	//  * termenv TemplateFuncs (see https://github.com/muesli/termenv).
	//  * The functions specified in ExtendedTemplateFuncs.
	Template string

	// ResultTemplate is rendered as soon as a input has been confirmed.
	// It is intended to permanently indicate the result of the prompt when the
	// input itself has disappeared. This template is only rendered in the Run()
	// method and NOT when the text input is used as a model. The following
	// variables and functions are available:
	//
	//  * FinalChoice: The choice that was selected by the user.
	//  * Prompt string: The configured prompt.
	//  * InitialValue string: The configured initial value of the input.
	//  * Placeholder string: The configured placeholder of the input.
	//  * TerminalWidth int: The width of the terminal.
	//  * AutoCompleteTriggered bool: An indication that auto-complete was
	//    just triggered by the user. It resets after further input.
	//  * AutoCompleteIndecisive bool: An indication that auto-complete was
	//    just triggered by the user with an indecisive results. It resets
	//    after further input.
	//  * AutoCompleteSuggestions() []string: A function that returns the
	//    auto-complete suggestions for the current input.
	//  * Mask(string) string: A function that replaces all characters of
	//    a string with the character specified in HideMask if Hidden is
	//    true and returns the input string if Hidden is false.
	//  * promptkit.UtilFuncMap: Handy helper functions.
	//  * termenv TemplateFuncs (see https://github.com/muesli/termenv).
	//  * The functions specified in ExtendedTemplateFuncs.
	ResultTemplate string

	// ExtendedTemplateFuncs can be used to add additional functions to the
	// evaluation scope of the templates.
	ExtendedTemplateFuncs template.FuncMap

	// Styles of the actual input field. These will be applied as inline styles.
	//
	// For an introduction to styling with Lip Gloss see:
	// https://github.com/charmbracelet/lipgloss
	InputTextStyle        lipgloss.Style
	InputBackgroundStyle  lipgloss.Style
	InputPlaceholderStyle lipgloss.Style
	InputCursorStyle      lipgloss.Style

	// KeyMap determines with which keys the text input is controlled. By
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

// New creates a new text input. See the TextInput properties for more
// documentation.
func New(prompt string) *TextInput {
	return &TextInput{
		Prompt:                prompt,
		Template:              DefaultTemplate,
		ResultTemplate:        DefaultResultTemplate,
		KeyMap:                NewDefaultKeyMap(),
		InputPlaceholderStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("240")),
		Validate:              ValidateNotEmpty,
		HideMask:              DefaultMask,
		ExtendedTemplateFuncs: template.FuncMap{},
		WrapMode:              promptkit.Truncate,
		Output:                os.Stdout,
		Input:                 os.Stdin,
	}
}

// RunPrompt executes the text input prompt.
func (t *TextInput) RunPrompt() (string, error) {
	err := validateKeyMap(t.KeyMap)
	if err != nil {
		return "", fmt.Errorf("insufficient key map: %w", err)
	}

	m := NewModel(t)

	p := tea.NewProgram(m, tea.WithOutput(t.Output), tea.WithInput(t.Input))
	if err := p.Start(); err != nil {
		return "", fmt.Errorf("running prompt: %w", err)
	}

	return m.Value()
}

// ValidateNotEmpty is a validation function that ensures that the input is not
// empty.
func ValidateNotEmpty(s string) error {
	if len(s) == 0 {
		return ErrInputValidation
	}

	return nil
}

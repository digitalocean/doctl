package input

import (
	"errors"
	"fmt"

	"github.com/digitalocean/doctl/commands/charm"
	"github.com/digitalocean/doctl/commands/charm/template"
	"github.com/erikgeiser/promptkit"
	"github.com/erikgeiser/promptkit/textinput"
)

type Input struct {
	text         string
	placeholder  string
	initialValue string
	hidden       bool
	required     bool
	validator    Validator
}

type Validator func(input string) error

var ErrRequired = fmt.Errorf("required")

type Option func(*Input)

func New(text string, opts ...Option) *Input {
	i := &Input{
		text: text,
	}
	for _, opt := range opts {
		opt(i)
	}
	return i
}

func WithPlaceholder(s string) Option {
	return func(i *Input) {
		i.placeholder = s
	}
}

func WithRequired() Option {
	return func(i *Input) {
		i.required = true
	}
}

func WithHidden() Option {
	return func(i *Input) {
		i.hidden = true
	}
}

func WithValidator(v Validator) Option {
	return func(i *Input) {
		i.validator = v
	}
}

func WithInitialValue(s string) Option {
	return func(i *Input) {
		i.initialValue = s
	}
}

var templateFuncs template.FuncMap

func init() {
	templateFuncs = template.Funcs(charm.Colors)
	templateFuncs["ErrRequired"] = func() error { return ErrRequired }
}

func (i *Input) Prompt() (string, error) {
	in := textinput.New(i.text)
	in.Placeholder = i.placeholder
	in.InitialValue = i.initialValue
	in.Hidden = i.hidden

	validator := i.validator
	if i.required {
		validator = func(input string) error {
			if input == "" {
				return ErrRequired
			}
			if i.validator != nil {
				return i.validator(input)
			}
			return nil
		}
	}
	in.Validate = validator
	in.ExtendedTemplateFuncs = templateFuncs

	in.Template = `
		{{- highlight promptPrefix }} {{ bold .Prompt }} {{ .Input -}}
		{{- with .ValidationError -}}
			{{- if eq ErrRequired . -}}
				{{- error (printf " %s required" asterisk) -}}
			{{- else -}}
				{{- error (printf " %s %v" crossmark .) -}}
			{{- end -}}
		{{- else }}
		{{- success (printf " %s" checkmark) -}}
		{{- end -}}
		{{- nl}}{{- muted "   ctrl-c to cancel" -}}
	`

	in.ResultTemplate = `
		{{- success promptPrefix }} {{ bold .Prompt }} {{ Mask .FinalValue -}}{{nl -}}
	`

	res, err := in.RunPrompt()
	if err != nil {
		if errors.Is(err, promptkit.ErrAborted) {
			template.Print(`
			{{- error promptPrefix }} {{ bold . }} {{ error "canceled" }}{{nl -}}
		`, i.text)
		}
		return "", err
	}
	return res, nil
}

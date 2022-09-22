package confirm

import (
	"github.com/digitalocean/doctl/commands/charm"
	"github.com/digitalocean/doctl/commands/charm/template"
	"github.com/erikgeiser/promptkit/confirmation"
)

// Choice describes a prompt choice.
type Choice string

const (
	// Yes represents the "yes" choice.
	Yes Choice = "yes"
	// No represents the "no" choice.
	No Choice = "no"
	// Undecided is returned if the user was unable to make a choice.
	Undecided Choice = "undecided"
)

func toChoice(v *bool) Choice {
	if v == nil {
		return Undecided
	}

	if *v {
		return Yes
	} else {
		return No
	}
}

func fromChoice(v Choice) confirmation.Value {
	switch v {
	case Yes:
		return confirmation.Yes
	case No:
		return confirmation.No
	default:
		return confirmation.Undecided
	}
}

// Prompt describes a confirmation prompt.
type Prompt struct {
	text          string
	choice        Choice
	persistPrompt PersistPrompt
}

// Option configures an option on a prompt.
type Option func(*Prompt)

// WithDefaultChoice sets the default choice on the prompt.
func WithDefaultChoice(c Choice) Option {
	return func(p *Prompt) {
		p.choice = c
	}
}

// PersistPrompt describes the behavior of the prompt after a choice is made.
type PersistPrompt int

const (
	// PersistPromptAlways always persists the prompt on the screen regardless of the choice.
	PersistPromptAlways PersistPrompt = iota
	// PersistPromptIfYes only persists the prompt on the screen if the choice is Yes.
	PersistPromptIfYes
	// PersistPromptIfNo only persists the prompt on the screen if the choice is No.
	PersistPromptIfNo
)

// WithPersistPrompt configures the prompt persistance behavior.
func WithPersistPrompt(v PersistPrompt) Option {
	return func(p *Prompt) {
		p.persistPrompt = v
	}
}

// New creates a new prompt.
func New(text string, opts ...Option) *Prompt {
	p := &Prompt{
		text: text,
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

var promptTemplate = `
{{- highlight promptPrefix }} {{ .Prompt -}}
{{ if .YesSelected -}}
	{{- print (bold (print " " pointerRight "yes ")) " no" -}}
{{- else if .NoSelected -}}
	{{- print "  yes " (bold (print pointerRight "no")) -}}
{{- else -}}
	{{- "  yes  no" -}}
{{- end -}}
`

var resultTemplate = `
{{- if RenderResult .FinalValue -}}
{{- if .FinalValue -}}{{success promptPrefix}}{{else}}{{error promptPrefix}}{{end}}
{{- print " " .Prompt " " -}}
	{{- if .FinalValue -}}
		{{- success "yes" -}}
	{{- else -}}
		{{- error "no" -}}
	{{- end }}
{{- end -}}
`

// Prompt renders the prompt on the screen.
func (p *Prompt) Prompt() (Choice, error) {
	input := confirmation.New(p.text, fromChoice(p.choice))
	tfs := template.Funcs(charm.Colors)
	tfs["RenderResult"] = func(choice bool) bool {
		switch p.persistPrompt {
		case PersistPromptAlways:
			return true
		case PersistPromptIfNo:
			// the prompt should only be persisted if the choice is `no`
			// render only if choice == false
			return !choice
		case PersistPromptIfYes:
			// the prompt should only be persisted if the choice is `yes`
			// render only if choice == true
			return choice
		default:
			return true
		}
	}
	input.ExtendedTemplateFuncs = tfs
	input.Template = promptTemplate
	input.ResultTemplate = resultTemplate

	v, err := input.RunPrompt()
	if err != nil {
		if err.Error() == "no decision was made" {
			return Undecided, err
		}
		return "", err
	}
	return toChoice(&v), nil
}

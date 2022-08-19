package confirm

import (
	"github.com/digitalocean/doctl/commands/charm"
	"github.com/erikgeiser/promptkit/confirmation"
)

type Choice string

const (
	Yes       Choice = "yes"
	No        Choice = "no"
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

type Prompt struct {
	text   string
	choice Choice
}

type Option func(*Prompt)

func WithDefaultChoice(c Choice) Option {
	return func(p *Prompt) {
		p.choice = c
	}
}

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
{{- if.FinalValue -}}{{success promptPrefix}}{{else}}{{error promptPrefix}}{{end}}
{{- print " " .Prompt " " -}}
{{- if .FinalValue -}}
	{{- success "yes" -}}
{{- else -}}
	{{- error "no" -}}
{{- end }}
`

func (p *Prompt) Prompt() (Choice, error) {
	input := confirmation.New(p.text, fromChoice(p.choice))
	input.ExtendedTemplateFuncs = charm.TemplateFuncs(charm.Colors)
	input.Template = promptTemplate
	input.ResultTemplate = resultTemplate

	v, err := input.RunPrompt()
	if err != nil {
		if err.Error() == "no decision was made" {
			return Undecided, err
		}
		return "", nil
	}
	return toChoice(&v), nil
}

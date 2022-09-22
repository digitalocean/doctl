package selection

import "github.com/erikgeiser/promptkit/selection"

type Selection struct {
	options   []string
	prompt    string
	filtering bool
}

type Option func(*Selection)

func New(options []string, opts ...Option) *Selection {
	s := &Selection{
		options:   options,
		filtering: true,
		prompt:    "selection:",
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

func WithFiltering(v bool) Option {
	return func(s *Selection) {
		s.filtering = v
	}
}

func WithPrompt(prompt string) Option {
	return func(s *Selection) {
		s.prompt = prompt
	}
}

func (s *Selection) Select() (string, error) {
	sp := selection.New(s.prompt, s.options)
	if !s.filtering {
		sp.Filter = nil
	}
	return sp.RunPrompt()
}

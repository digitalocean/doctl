package selection

import "github.com/erikgeiser/promptkit/selection"

type Selection struct {
	options   []string
	prompt    string
	filtering bool
	pageSize  int
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

// WithPageSize scrolls the list instead of printing every option, which keeps
// a long list (a service catalog, say) from pushing the question itself off
// screen. Zero, the default, prints everything.
func WithPageSize(n int) Option {
	return func(s *Selection) {
		s.pageSize = n
	}
}

func (s *Selection) Select() (string, error) {
	sp := selection.New(s.prompt, s.options)
	if !s.filtering {
		sp.Filter = nil
	}
	if s.pageSize > 0 {
		sp.PageSize = s.pageSize
	}
	return sp.RunPrompt()
}

package doit

import "golang.org/x/oauth2"

// TokenSource holds an oauth token.
type TokenSource struct {
	AccessToken string
}

// Runner is an interface that Runs things.
type Runner interface {
	Run() error
}

// MockRunner is an implemenation of Runner for mocking.
type MockRunner struct {
	Err error
}

var _ Runner = &MockRunner{}

// Run mock runs things.
func (tr *MockRunner) Run() error {
	return tr.Err
}

// Token returns an oauth token.
func (t *TokenSource) Token() (*oauth2.Token, error) {
	return &oauth2.Token{
		AccessToken: t.AccessToken,
	}, nil
}

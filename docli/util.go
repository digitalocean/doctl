package docli

import (
	"bytes"
	"encoding/json"
	"io"

	"github.com/codegangsta/cli"
	"github.com/digitalocean/godo"
	"golang.org/x/oauth2"
)

var DefaultClientSource ClientSource = &LiveClientSource{}

type TokenSource struct {
	AccessToken string
}

func (t *TokenSource) Token() (*oauth2.Token, error) {
	return &oauth2.Token{
		AccessToken: t.AccessToken,
	}, nil
}

func WriteJSON(item interface{}, w io.Writer) error {
	b, err := json.Marshal(item)
	if err != nil {
		return err
	}

	var out bytes.Buffer
	json.Indent(&out, b, "", "  ")
	_, err = out.WriteTo(w)
	return err

}

func ToJSON(item interface{}) (string, error) {
	b, err := json.MarshalIndent(item, "", "  ")
	if err != nil {
		return "", err
	}

	return string(b), nil
}

type ClientSource interface {
	NewClient(token string) *godo.Client
}

type LiveClientSource struct{}

func (cs *LiveClientSource) NewClient(token string) *godo.Client {
	tokenSource := &TokenSource{
		AccessToken: token,
	}

	oauthClient := oauth2.NewClient(oauth2.NoContext, tokenSource)
	return godo.NewClient(oauthClient)
}

func NewClient(c *cli.Context, cs ClientSource) *godo.Client {
	if cs == nil {
		cs = &LiveClientSource{}
	}

	pat := c.GlobalString("token")
	return cs.NewClient(pat)
}

func WithinTest(cs ClientSource, fn func()) {
	ogSource := DefaultClientSource
	DefaultClientSource = cs

	defer func() {
		DefaultClientSource = ogSource
	}()

	fn()
}

package docli

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"io"

	"github.com/codegangsta/cli"
	"github.com/digitalocean/godo"
	"golang.org/x/oauth2"
)

var DefaultClientSource ClientSource = &LiveClientSource{}

type TokenSource struct {
	AccessToken string
}

type TestClientSource struct {
	Client *godo.Client
}

func (cs *TestClientSource) NewClient(_ string) *godo.Client {
	return cs.Client
}

func (t *TokenSource) Token() (*oauth2.Token, error) {
	return &oauth2.Token{
		AccessToken: t.AccessToken,
	}, nil
}

func LoadOpts(c *cli.Context) *Opts {
	return &Opts{
		Debug: c.GlobalBool("debug"),
	}
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

func WithinTest(cs ClientSource, fs *flag.FlagSet, fn func(*cli.Context)) {
	ogSource := DefaultClientSource
	DefaultClientSource = cs

	defer func() {
		DefaultClientSource = ogSource
	}()

	var b bytes.Buffer
	app := cli.NewApp()
	app.Writer = bufio.NewWriter(&b)

	globalSet := flag.NewFlagSet("global test", 0)
	globalSet.String("token", "token", "token")

	globalCtx := cli.NewContext(app, globalSet, nil)

	if fs == nil {
		fs = flag.NewFlagSet("local test", 0)
	}

	c := cli.NewContext(app, fs, globalCtx)

	fn(c)
}

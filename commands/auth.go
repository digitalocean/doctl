package commands

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"

	"code.google.com/p/go-uuid/uuid"
	"github.com/bryanl/doit"
	"github.com/bryanl/doit-server"
	"github.com/bryanl/webbrowser"
	"github.com/gorilla/websocket"
	"github.com/spf13/cobra"
)

var (
	doitServerURL = "http://doit-server.apps.pifft.com"
)

// UnknownSchemeError signifies an unknown HTTP scheme.
type UnknownSchemeError struct {
	Scheme string
}

var _ error = &UnknownSchemeError{}

func (use *UnknownSchemeError) Error() string {
	return "unknown scheme: " + use.Scheme
}

// Auth creates auth commands for doit.
func Auth() *cobra.Command {
	cmdAuth := &cobra.Command{
		Use:   "auth",
		Short: "auth commands",
		Long:  "auth is used to access auth commands",
	}

	cmdAuthLogin := cmdBuilder(RunAuthLogin, "login", "login to DigitalOcean account", writer)
	cmdAuth.AddCommand(cmdAuthLogin)

	return cmdAuth
}

// RunAuthLogin runs auth login. It communicates with doit-server to perform auth.
func RunAuthLogin(ns string, config doit.Config, out io.Writer, args []string) error {
	ac, err := retrieveAuthCredentials(doitServerURL)
	if err != nil {
		return err
	}

	u, err := createAuthURL(doitServerURL, ac)
	if err != nil {
		return err
	}

	webbrowser.Open(u, webbrowser.NewTab, true)

	tr, err := monitorAuth(doitServerURL, ac)
	if err != nil {
		return err
	}

	cf := doit.NewConfigFile()
	err = cf.Set("access-token", tr.AccessToken)
	if err != nil {
		return err
	}

	fmt.Println("updated access token")

	return nil
}

func retrieveAuthCredentials(serverURL string) (*doitserver.AuthCredentials, error) {
	u, err := url.Parse(serverURL)
	if err != nil {
		return nil, err
	}

	u.Path = "/token"
	v := u.Query()
	v.Set("id", uuid.New())
	u.RawQuery = v.Encode()

	r, err := http.Get(u.String())
	if err != nil {
		return nil, err
	}
	defer r.Body.Close()
	if err != nil {
		return nil, err
	}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}

	if r.StatusCode != 200 {
		return nil, errors.New("it's broke, Jim")
	}

	var m doitserver.AuthCredentials
	err = json.Unmarshal(body, &m)
	if err != nil {
		return nil, err
	}

	return &m, nil
}

func createAuthURL(serverURL string, ac *doitserver.AuthCredentials) (string, error) {
	authURL, err := url.Parse(serverURL)
	if err != nil {
		return "", err
	}

	authURL.Path = "/auth/digitalocean"

	q := authURL.Query()
	q.Set("id", ac.ID)
	q.Set("cs", ac.CS)
	authURL.RawQuery = q.Encode()

	return authURL.String(), nil

}

func monitorAuth(serverURL string, ac *doitserver.AuthCredentials) (*doitserver.TokenResponse, error) {
	u, err := url.Parse(serverURL)
	if err != nil {
		return nil, err
	}

	switch u.Scheme {
	case "http":
		u.Scheme = "ws"
	case "https":
		u.Scheme = "wss"
	default:
		return nil, &UnknownSchemeError{Scheme: u.Scheme}
	}

	u.Path = "/status"

	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		return nil, err
	}

	err = conn.WriteJSON(ac)
	if err != nil {
		return nil, err
	}

	var tr doitserver.TokenResponse

	err = conn.ReadJSON(&tr)
	if err != nil {
		return nil, err
	}

	return &tr, nil

}

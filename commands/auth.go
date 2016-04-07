/*
Copyright 2016 The Doctl Authors All rights reserved.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package commands

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"runtime"

	"golang.org/x/crypto/ssh/terminal"

	"github.com/bryanl/doit-server"
	"github.com/bryanl/webbrowser"
	"github.com/digitalocean/doctl"
	"github.com/gorilla/websocket"
	"github.com/satori/go.uuid"
	"github.com/spf13/cobra"
)

// ErrUnknownTerminal signies an unknown terminal. It is returned when doit
// can't ascertain the current terminal type with requesting an auth token.
var ErrUnknownTerminal = errors.New("unknown terminal")

// UnknownSchemeError signifies an unknown HTTP scheme.
type UnknownSchemeError struct {
	Scheme string
}

var _ error = &UnknownSchemeError{}

func (use *UnknownSchemeError) Error() string {
	return "unknown scheme: " + use.Scheme
}

// Auth creates auth commands for doit.
func Auth() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:   "auth",
			Short: "auth commands",
			Long:  "auth is used to access auth commands",
		},
	}

	CmdBuilder(cmd, RunAuthLogin, "login", "login to DigitalOcean account", Writer, docCategories("account"))

	return cmd
}

// RunAuthLogin runs auth login. It communicates with doit-server to perform auth.
func RunAuthLogin(c *CmdConfig) error {
	dsa := newDoitServerAuth()

	ac, err := dsa.retrieveAuthCredentials()
	if err != nil {
		return err
	}

	token, err := dsa.initAuth(ac)
	if err != nil {
		return err
	}

	cf, err := doit.NewConfigFile()
	if err != nil {
		return err
	}

	err = cf.Set("access-token", token)
	if err != nil {
		return err
	}

	fmt.Println("updated access token")

	return nil
}

type doitServerAuth struct {
	url         string
	browserOpen func(u string) error
	isCLI       func() bool
	monitorAuth func(u string, ac *doitserver.AuthCredentials) (*doitserver.TokenResponse, error)
}

func newDoitServerAuth() *doitServerAuth {
	return &doitServerAuth{
		url: "http://doit-server.apps.pifft.com",
		browserOpen: func(u string) error {
			return webbrowser.Open(u, webbrowser.NewTab, true)
		},
		isCLI: func() bool {
			return (runtime.GOOS == "linux" && os.Getenv("DISPLAY") == "") || os.Getenv("CLIAUTH") != ""
		},
		monitorAuth: monitorAuthWS,
	}
}

func (dsa *doitServerAuth) initAuth(ac *doitserver.AuthCredentials) (string, error) {
	if dsa.isCLI() {
		return dsa.initAuthCLI(ac)
	}

	return dsa.initAuthBrowser(ac)
}

func (dsa *doitServerAuth) initAuthCLI(ac *doitserver.AuthCredentials) (string, error) {
	if !terminal.IsTerminal(int(os.Stdout.Fd())) {
		return "", ErrUnknownTerminal
	}

	u, err := dsa.createAuthURL(ac, keyPair{k: "cliauth", v: "1"})
	if err != nil {
		return "", err
	}

	fmt.Printf("Visit the following URL in your browser: %s\n", u)

	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter token: ")
	return reader.ReadString('\n')
}

func (dsa *doitServerAuth) initAuthBrowser(ac *doitserver.AuthCredentials) (string, error) {
	u, err := dsa.createAuthURL(ac)
	if err != nil {
		return "", err
	}

	err = dsa.browserOpen(u)
	if err != nil {
		return "", err
	}

	tr, err := dsa.monitorAuth(dsa.url, ac)
	if err != nil {
		return "", err
	}

	return tr.AccessToken, nil
}

func (dsa *doitServerAuth) retrieveAuthCredentials() (*doitserver.AuthCredentials, error) {
	u, err := url.Parse(dsa.url)
	if err != nil {
		return nil, err
	}

	u.Path = "/token"
	v := u.Query()
	v.Set("id", uuid.NewV4().String())
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

type keyPair struct {
	k, v string
}

func (dsa *doitServerAuth) createAuthURL(ac *doitserver.AuthCredentials, kps ...keyPair) (string, error) {
	authURL, err := url.Parse(dsa.url)
	if err != nil {
		return "", err
	}

	authURL.Path = "/auth/digitalocean"

	q := authURL.Query()
	q.Set("id", ac.ID)
	q.Set("cs", ac.CS)

	for _, kp := range kps {
		q.Set(kp.k, kp.v)
	}

	authURL.RawQuery = q.Encode()

	return authURL.String(), nil

}

func monitorAuthWS(serverURL string, ac *doitserver.AuthCredentials) (*doitserver.TokenResponse, error) {
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

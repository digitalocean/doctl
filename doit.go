package doit

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"github.com/bryanl/doit/Godeps/_workspace/src/github.com/Sirupsen/logrus"
	"github.com/bryanl/doit/Godeps/_workspace/src/github.com/digitalocean/godo"
	"github.com/bryanl/doit/Godeps/_workspace/src/github.com/docker/docker/pkg/term"
	"github.com/bryanl/doit/Godeps/_workspace/src/github.com/spf13/viper"
	"github.com/bryanl/doit/Godeps/_workspace/src/golang.org/x/crypto/ssh"
	"github.com/bryanl/doit/Godeps/_workspace/src/golang.org/x/crypto/ssh/terminal"
	"github.com/bryanl/doit/Godeps/_workspace/src/golang.org/x/oauth2"
)

var (
	// DoitConfig holds the app's current configuration.
	DoitConfig Config = &LiveConfig{}
)

// Config is an interface that represent doit's config.
type Config interface {
	GetGodoClient() *godo.Client
	SSH(user, host, keyPath string, port int) Runner
	Set(ns, key string, val interface{})
	GetString(ns, key string) string
	GetBool(ns, key string) bool
	GetInt(ns, key string) int
	GetStringSlice(ns, key string) []string
}

// LiveConfig is an implementation of Config for live values.
type LiveConfig struct{}

var _ Config = &LiveConfig{}

// GetGodoClient returns a GodoClient.
func (c *LiveConfig) GetGodoClient() *godo.Client {
	token := viper.GetString("access-token")
	tokenSource := &TokenSource{AccessToken: token}
	oauthClient := oauth2.NewClient(oauth2.NoContext, tokenSource)
	return godo.NewClient(oauthClient)
}

func sshConnect(user string, host string, method ssh.AuthMethod) error {
	sshc := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{method},
	}
	conn, err := ssh.Dial("tcp", host, sshc)
	if err != nil {
		return err
	}
	defer conn.Close()

	session, err := conn.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	session.Stdout = os.Stdout
	session.Stderr = os.Stderr
	session.Stdin = os.Stdin

	var (
		termWidth, termHeight int
	)
	fd := os.Stdin.Fd()

	oldState, err := term.MakeRaw(fd)
	if err != nil {
		return err
	}
	defer term.RestoreTerminal(fd, oldState)

	winsize, err := term.GetWinsize(fd)
	if err != nil {
		termWidth = 80
		termHeight = 24
	} else {
		termWidth = int(winsize.Width)
		termHeight = int(winsize.Height)
	}

	modes := ssh.TerminalModes{
		ssh.ECHO: 1,
	}

	if err := session.RequestPty("xterm", termHeight, termWidth, modes); err != nil {
		return err
	}
	if err == nil {
		err = session.Shell()
	}
	if err != nil {
		return err
	}

	err = session.Wait()
	if _, ok := err.(*ssh.ExitError); ok {
		return nil
	}
	if err == io.EOF {
		return nil
	}
	return err
}

type sshRunner struct {
	user    string
	host    string
	keyPath string
	port    int
}

var _ Runner = &sshRunner{}

func (r *sshRunner) Run() error {
	logrus.WithFields(logrus.Fields{
		"user": r.user,
		"host": r.host,
	}).Info("ssh")

	sshHost := fmt.Sprintf("%s:%d", r.host, r.port)

	// Key Auth
	key, err := ioutil.ReadFile(r.keyPath)
	if err != nil {
		return err
	}
	privateKey, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return err
	}
	if err := sshConnect(r.user, sshHost, ssh.PublicKeys(privateKey)); err != nil {
		// Password Auth if Key Auth Fails
		fd := os.Stdin.Fd()
		state, err := terminal.MakeRaw(int(fd))
		if err != nil {
			return err
		}
		defer terminal.Restore(int(fd), state)
		t := terminal.NewTerminal(os.Stdout, ">")
		password, err := t.ReadPassword("Password: ")
		if err != nil {
			return err
		}
		if err := sshConnect(r.user, sshHost, ssh.Password(string(password))); err != nil {
			return err
		}
	}
	return err

}

// SSH creates a ssh connection to a host.
func (c *LiveConfig) SSH(user, host, keyPath string, port int) Runner {
	return &sshRunner{
		user:    user,
		host:    host,
		keyPath: keyPath,
		port:    port,
	}

}

// Set sets a config key.
func (c *LiveConfig) Set(ns, key string, val interface{}) {
	nskey := fmt.Sprintf("%s-%s", ns, key)
	viper.Set(nskey, val)
}

// GetString returns a config value as a string.
func (c *LiveConfig) GetString(ns, key string) string {
	if ns == NSRoot {
		return viper.GetString(key)
	}

	nskey := fmt.Sprintf("%s-%s", ns, key)
	return viper.GetString(nskey)
}

// GetBool returns a config value as a bool.
func (c *LiveConfig) GetBool(ns, key string) bool {
	if ns == NSRoot {
		return viper.GetBool(key)
	}

	nskey := fmt.Sprintf("%s-%s", ns, key)
	return viper.GetBool(nskey)
}

// GetInt returns a config value as an int.
func (c *LiveConfig) GetInt(ns, key string) int {
	if ns == NSRoot {
		return viper.GetInt(key)
	}

	nskey := fmt.Sprintf("%s-%s", ns, key)
	return viper.GetInt(nskey)
}

// GetStringSlice returns a config value as a string slice.
func (c *LiveConfig) GetStringSlice(ns, key string) []string {
	if ns == NSRoot {
		return viper.GetStringSlice(key)
	}

	nskey := fmt.Sprintf("%s-%s", ns, key)
	return viper.GetStringSlice(nskey)
}

package doit

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"github.com/Sirupsen/logrus"
	"github.com/digitalocean/godo"
	"github.com/docker/docker/pkg/term"
	"github.com/spf13/viper"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/terminal"
	"golang.org/x/oauth2"
)

var (
	// DoitConfig holds the app's current configuration.
	DoitConfig Config = &LiveConfig{}
)

// Config is an interface that represent doit's config.
type Config interface {
	GetGodoClient() *godo.Client
	SSH(user, host, keyPath string, port int) error
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

func sshConnect(user string, host string, method ssh.AuthMethod) (err error) {
	sshc := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{method},
	}
	conn, err := ssh.Dial("tcp", host, sshc)
	if err != nil {
		return err
	}

	session, err := conn.NewSession()
	if err != nil {
		return err
	}

	session.Stdout = os.Stdout
	session.Stderr = os.Stderr
	session.Stdin = os.Stdin

	defer session.Close()

	modes := ssh.TerminalModes{
		ssh.ECHO: 1,
	}

	var (
		termWidth, termHeight int
	)
	fd := os.Stdin.Fd()
	if term.IsTerminal(fd) {
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
	}

	if err := session.RequestPty("xterm", termWidth, termHeight, modes); err != nil {
		session.Close()
		return err
	}
	if err == nil {
		err = session.Shell()
	}
	if err != nil {
		return err
	}

	err = session.Wait()
	if err != nil && err != io.EOF {
		// Ignore the error if it's an ExitError with an empty message,
		// this occurs when you do CTRL+c and then run exit cmd which isn't an
		// actual error.
		waitMsg, ok := err.(*ssh.ExitError)
		if ok && waitMsg.Msg() == "" {
			return nil
		}

		return err
	}
	return err
}

// SSH creates a ssh connection to a host.
func (c *LiveConfig) SSH(user, host, keyPath string, port int) (err error) {
	logrus.WithFields(logrus.Fields{
		"user": user,
		"host": host,
	}).Info("ssh")

	sshHost := fmt.Sprintf("%s:%d", host, port)

	// Key Auth
	key, err := ioutil.ReadFile(keyPath)
	if err != nil {
		return err
	}
	privateKey, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return err
	}
	if err := sshConnect(user, sshHost, ssh.PublicKeys(privateKey)); err != nil {
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
		if err := sshConnect(user, sshHost, ssh.Password(string(password))); err != nil {
			return err
		}
	}
	return err
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

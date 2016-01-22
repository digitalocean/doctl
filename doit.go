package doit

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"github.com/bryanl/doit/pkg/term"
	"github.com/digitalocean/godo"
	jww "github.com/spf13/jwalterweatherman"
	"github.com/spf13/viper"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/terminal"
	"golang.org/x/oauth2"
)

const (
	// NSRoot is a configuration key that signifies this value is at the root.
	NSRoot = "doit"
)

var (
	// DoitConfig holds the app's current configuration.
	DoitConfig Config = &LiveConfig{}

	// DoitVersion is doit's version.
	DoitVersion = Version{
		Major: 0,
		Minor: 7,
		Patch: 0,
		Name:  "Maroon Marion",
		Label: "dev",
	}

	// Build is doit's build tag.
	Build string
)

func init() {
	jww.SetStdoutThreshold(jww.LevelError)
}

// Version is the version info for doit.
type Version struct {
	Major, Minor, Patch int
	Name, Build, Label  string
}

func (v Version) String() string {
	var buffer bytes.Buffer
	buffer.WriteString(fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch))
	if v.Label != "" {
		buffer.WriteString("-" + v.Label)
	}

	return buffer.String()
}

// Complete is the complete version for doit.
func (v Version) Complete() string {
	var buffer bytes.Buffer
	buffer.WriteString(fmt.Sprintf("doit version %s", v.String()))

	buffer.WriteString(fmt.Sprintf(" %q", v.Name))

	if v.Build != "" {
		buffer.WriteString(fmt.Sprintf("\nGit commit hash: %s", v.Build))
	}

	return buffer.String()
}

// Config is an interface that represent doit's config.
type Config interface {
	GetGodoClient() *godo.Client
	SSH(user, host, keyPath string, port int) Runner
	Set(ns, key string, val interface{})
	GetString(ns, key string) (string, error)
	GetBool(ns, key string) (bool, error)
	GetInt(ns, key string) (int, error)
	GetStringSlice(ns, key string) ([]string, error)
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
	defer func() {
		_ = conn.Close()
	}()

	session, err := conn.NewSession()
	if err != nil {
		return err
	}
	defer func() {
		_ = session.Close()
	}()

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
	defer func() {
		_ = term.RestoreTerminal(fd, oldState)
	}()

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
		defer func() {
			_ = terminal.Restore(int(fd), state)
		}()
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
func (c *LiveConfig) GetString(ns, key string) (string, error) {
	if ns == NSRoot {
		return viper.GetString(key), nil
	}

	nskey := fmt.Sprintf("%s.%s", ns, key)

	if _, ok := viper.AllSettings()[fmt.Sprintf("%s.required", nskey)]; ok {
		if viper.GetString(nskey) == "" {
			return "", NewMissingArgsErr(nskey)
		}
	}
	return viper.GetString(nskey), nil
}

// GetBool returns a config value as a bool.
func (c *LiveConfig) GetBool(ns, key string) (bool, error) {
	if ns == NSRoot {
		return viper.GetBool(key), nil
	}

	nskey := fmt.Sprintf("%s.%s", ns, key)

	return viper.GetBool(nskey), nil
}

// GetInt returns a config value as an int.
func (c *LiveConfig) GetInt(ns, key string) (int, error) {
	if ns == NSRoot {
		return viper.GetInt(key), nil
	}

	nskey := fmt.Sprintf("%s.%s", ns, key)

	if _, ok := viper.AllSettings()[fmt.Sprintf("%s.required", nskey)]; ok {
		if viper.GetInt(nskey) < 0 {
			return 0, NewMissingArgsErr(nskey)
		}
	}

	return viper.GetInt(nskey), nil
}

// GetStringSlice returns a config value as a string slice.
func (c *LiveConfig) GetStringSlice(ns, key string) ([]string, error) {
	if ns == NSRoot {
		return viper.GetStringSlice(key), nil
	}

	nskey := fmt.Sprintf("%s.%s", ns, key)

	if _, ok := viper.AllSettings()[fmt.Sprintf("%s.required", nskey)]; ok {
		if viper.GetStringSlice(nskey) == nil {
			return nil, NewMissingArgsErr(nskey)
		}
	}

	return viper.GetStringSlice(nskey), nil
}

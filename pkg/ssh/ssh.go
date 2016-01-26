package ssh

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"github.com/bryanl/doit/pkg/runner"
	"github.com/bryanl/doit/pkg/term"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/terminal"
)

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

// Runner runs ssh commands.
type Runner struct {
	User    string
	Host    string
	KeyPath string
	Port    int
}

var _ runner.Runner = &Runner{}

// Run ssh.
func (r *Runner) Run() error {
	sshHost := fmt.Sprintf("%s:%d", r.Host, r.Port)

	// Key Auth
	key, err := ioutil.ReadFile(r.KeyPath)
	if err != nil {
		return err
	}
	privateKey, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return err
	}

	if err := sshConnect(r.User, sshHost, ssh.PublicKeys(privateKey)); err != nil {
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
		if err := sshConnect(r.User, sshHost, ssh.Password(string(password))); err != nil {
			return err
		}
	}
	return err

}

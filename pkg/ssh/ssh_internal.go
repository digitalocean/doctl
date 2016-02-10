package ssh

import (
	"fmt"
	"io/ioutil"
	"os"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/terminal"
)

func runInternalSSH(r *Runner) error {
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

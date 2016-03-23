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

package ssh

import (
	"io"
	"os"
	"runtime"

	"github.com/digitalocean/doctl/pkg/runner"
	"github.com/digitalocean/doctl/pkg/term"
	"golang.org/x/crypto/ssh"
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
	if runtime.GOOS == "windows" {
		return runInternalSSH(r)
	}

	return runExternalSSH(r)
}

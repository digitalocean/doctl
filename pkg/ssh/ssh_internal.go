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
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"golang.org/x/crypto/ssh/terminal"
)

type passwordProvider func(string) (string, error)

func askForPassword(prompt string) (string, error) {
	fd := os.Stdin.Fd()
	state, err := terminal.MakeRaw(int(fd))
	if err != nil {
		return "", err
	}
	defer func() {
		_ = terminal.Restore(int(fd), state)
	}()

	t := terminal.NewTerminal(os.Stdin, ">")
	fmt.Printf(prompt)
	password, err := t.ReadPassword("")
	if err != nil {
		return "", err
	}

	return password, nil
}

func sshConnect(user string, host string, method ssh.AuthMethod, a agent.Agent) error {
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

	if a != nil {
		if err := agent.RequestAgentForwarding(session); err != nil {
			return err
		}

		if err := agent.ForwardToAgent(conn, a); err != nil {
			return err
		}
	}

	fd := int(os.Stdin.Fd())

	oldState, err := terminal.MakeRaw(fd)
	if err != nil {
		return err
	}
	defer func() {
		_ = terminal.Restore(fd, oldState)
	}()

	termWidth, termHeight, err := terminal.GetSize(fd)
	if err != nil {
		termWidth = 80
		termHeight = 24
	}

	modes := ssh.TerminalModes{
		ssh.ECHO: 1,
	}

	if err := session.RequestPty("xterm", termHeight, termWidth, modes); err != nil {
		return err
	}

	if err := session.Shell(); err != nil {
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

func parsePrivateKey(path string, pwdProvider passwordProvider) (interface{}, error) {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// Convert key to PEM
	pemBlock, _ := pem.Decode(b)
	if pemBlock == nil {
		return nil, err
	}

	var k interface{}
	if x509.IsEncryptedPEMBlock(pemBlock) {
		prompt := fmt.Sprintf("Enter passphrase for key '%s': ", path)
		pwd, err := pwdProvider(prompt)
		if err != nil {
			return nil, err
		}
		b, err := x509.DecryptPEMBlock(pemBlock, []byte(pwd))
		if err != nil {
			return nil, err
		}
		k, err = x509.ParsePKCS1PrivateKey(b)
		if err != nil {
			return nil, err
		}
	} else {
		k, err = ssh.ParseRawPrivateKey(b)
		if err != nil {
			return nil, err
		}
	}
	return k, nil
}

func agentWithKey(k interface{}) (agent.Agent, error) {
	a := agent.NewKeyring()
	ak := agent.AddedKey{
		PrivateKey:   k,
		LifetimeSecs: 0,
	}
	if err := a.Add(ak); err != nil {
		return nil, err
	}
	return a, nil
}

func runInternalSSH(r *Runner) error {
	k, err := parsePrivateKey(r.KeyPath, askForPassword)
	if err != nil {
		return err
	}

	s, err := ssh.NewSignerFromKey(k)
	if err != nil {
		return err
	}

	var a agent.Agent
	if r.AgentForwarding {
		a, err = agentWithKey(k)
		if err != nil {
			return err
		}
	}

	sshHost := fmt.Sprintf("%s:%d", r.Host, r.Port)
	if err := sshConnect(r.User, sshHost, ssh.PublicKeys(s), a); err != nil {
		prompt := fmt.Sprintf("%s@%s's password: ", r.User, r.Host)
		password, err := askForPassword(prompt)
		if err != nil {
			return err
		}
		if err := sshConnect(r.User, sshHost, ssh.Password(string(password)), a); err != nil {
			return err
		}
	}
	return err
}

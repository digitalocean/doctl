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

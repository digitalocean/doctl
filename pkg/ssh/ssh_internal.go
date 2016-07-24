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
	"io/ioutil"
	"os"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/terminal"
)

func signerFromEncryptedKey(p *pem.Block, pwd []byte) (ssh.Signer, error) {
	b, err := x509.DecryptPEMBlock(p, pwd)
	if err != nil {
		return nil, err
	}

	k, err := x509.ParsePKCS1PrivateKey(b)
	if err != nil {
		return nil, err
	}

	s, err := ssh.NewSignerFromKey(k)
	if err != nil {
		return nil, err
	}

	return s, nil
}

func signerFromKey(b []byte) (ssh.Signer, error) {
	s, err := ssh.ParsePrivateKey(b)
	if err != nil {
		return nil, err
	}

	return s, nil
}

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

func runInternalSSH(r *Runner) error {
	sshHost := fmt.Sprintf("%s:%d", r.Host, r.Port)

	// Key Auth
	key, err := ioutil.ReadFile(r.KeyPath)
	if err != nil {
		return err
	}

	// Convert key to PEM
	pemBlock, _ := pem.Decode(key)
	if pemBlock == nil {
		return err
	}

	var signer ssh.Signer
	if x509.IsEncryptedPEMBlock(pemBlock) {
		var pwd string
		prompt := fmt.Sprintf("Enter passphrase for key '%s': ", r.KeyPath)
		if pwd, err = askForPassword(prompt); err != nil {
			return err
		}
		if signer, err = signerFromEncryptedKey(pemBlock, []byte(pwd)); err != nil {
			return err
		}
	} else {
		if signer, err = signerFromKey(key); err != nil {
			return err
		}
	}

	if err := sshConnect(r.User, sshHost, ssh.PublicKeys(signer)); err != nil {
		prompt := fmt.Sprintf("%s@%s's password: ", r.User, r.Host)
		password, err := askForPassword(prompt)
		if err != nil {
			return err
		}
		if err := sshConnect(r.User, sshHost, ssh.Password(string(password))); err != nil {
			return err
		}
	}
	return err
}

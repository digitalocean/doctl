/*
Copyright 2018 The Doctl Authors All rights reserved.
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
	"os"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/commands/displayers"
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"
)

// SSHKeys creates the ssh key commands hierarchy.
func SSHKeys() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:     "ssh-key",
			Aliases: []string{"k"},
			Short:   "Display commands to manage SSH keys on your account",
			Long: `The sub-commands of ` + "`" + `doctl compute ssh-key` + "`" + ` manage the SSH keys on your account.

DigitalOcean allows you to add SSH public keys to the interface so that you can embed your public key into a Droplet at the time of creation. Only the public key is required to take advantage of this functionality. Note that this command does not add, delete, or otherwise modify any SSH keys that may be on existing Droplets.`,
		},
	}

	cmdKeyList := CmdBuilder(cmd, RunKeyList, "list", "List all SSH keys on your account", `Retrieves a list of SSH keys associated with your account and their details, such as their IDs, fingerprints, public keys, and names.`, Writer,
		aliasOpt("ls"), displayerType(&displayers.Key{}))
	cmdKeyList.Example = `The following example lists all SSH keys on your account and use the ` + "`" + `--format` + "`" + ` flag to return only the ID and name of each key: doctl compute ssh-key list --format ID,Name`

	cmdKeyGet := CmdBuilder(cmd, RunKeyGet, "get <key-id|key-fingerprint>", "Retrieve information about an SSH key on your account", `Retrieves the ID, fingerprint, public key, and name of a specific SSH key on your account.`, Writer,
		aliasOpt("g"), displayerType(&displayers.Key{}))
	cmdKeyGet.Example = `The following example retrieves information about the SSH key with the ID ` + "`" + `386734086` + "`" + `: doctl compute ssh-key get 386734086`

	cmdSSHKeysCreate := CmdBuilder(cmd, RunKeyCreate, "create <key-name>", "Adds a new SSH key on your account", `Adds a new SSH key to your account. 

Before adding a key to your account, you must create a public and private key pair on your local machine using your preferred SSH client. Once you have created the key pair, you can add the public key to your DigitalOcean account so that you can embed your public key into a Droplet at the time of creation.

Specify a `+"`"+`<key-name>`+"`"+` for the key, and set the `+"`"+`--public-key`+"`"+` flag to a string with the contents of the key.

Adding a key to your account does not automatically add it to any Droplets. To add SSH keys to Droplets at Droplet creation time, using the `+"`"+`--ssh-keys <ssh-key-id>`+"`"+` flag with the `+"`"+`doctl compute droplet create`+"`"+` command.`, Writer,
		aliasOpt("c"), displayerType(&displayers.Key{}))
	AddStringFlag(cmdSSHKeysCreate, doctl.ArgKeyPublicKey, "", "", "The content's of the public key", requiredOpt())
	cmdSSHKeysCreate.Example = `The following example adds a new SSH key to your account with the name ` + "`" + `example-key` + "`" + `: doctl compute ssh-key create example-key --public-key="ssh-rsa AAAAB3NzaC1yc2EAAAABIwAAAQEAklOUpkDHrfHY17SbrmTIpNLTGK9Tjom/BWDSU
	GPl+nafzlHDTYW7hdI4yZ5ew18JH4JW9jbhUFrviQzM7xlELEVf4h9lFX5QVkbPppSwg0cda3
	Pbv7kOdJ/MTyBlWXFCR+HAo3FXRitBqxiX1nKhXpHAZsMciLq8V6RjsNAQwdsdMFvSlVK/7XA
	t3FaoJoAsncM1Q9x5+3V0Ww68/eIFmb1zuUFljQJKprrX88XypNDvjYNby6vw/Pb0rwert/En
	mZ+AW4OZPnTPI89ZPmVMLuayrD2cE86Z/il8b+gw3r3+1nKatmIkjn2so1d01QraTlMqVSsbx
	NrRFi9wrf+M7Q== user@mylaptop.local"`

	cmdSSHKeysImport := CmdBuilder(cmd, RunKeyImport, "import <key-name>", "Imports an SSH key from your computer to your account", `Imports a new SSH key to your account using a local public key file.

Adding a key to your account does not automatically add it to any Droplets. To add SSH keys to Droplets at Droplet creation time, using the `+"`"+`--ssh-keys <ssh-key-id>`+"`"+` flag with the `+"`"+`doctl compute droplet create`+"`"+` command.`, Writer,
		aliasOpt("i"), displayerType(&displayers.Key{}))
	AddStringFlag(cmdSSHKeysImport, doctl.ArgKeyPublicKeyFile, "", "", "A path to a public key file, such as `path/to/public-key.pub", requiredOpt())
	cmdSSHKeysImport.Example = `The following example imports a new SSH key into your account with the name ` + "`" + `example-key` + "`" + `: doctl compute ssh-key import example-key --public-key-file example-key.pub`

	cmdRunKeyDelete := CmdBuilder(cmd, RunKeyDelete, "delete <key-id|key-fingerprint>", "Permanently delete an SSH key from your account", `Permanently deletes an SSH key from your account.

This does not delete an SSH key from any Droplets and you can re-add the key to your account at anytime if you still have a copy of the public and private keys.`, Writer,
		aliasOpt("d", "rm"))
	AddBoolFlag(cmdRunKeyDelete, doctl.ArgForce, doctl.ArgShortForce, false, "Deletes the key without a confirmation prompt")
	cmdRunKeyDelete.Example = `The following example deletes the SSH key with the ID ` + "`" + `386734086` + "`" + `: doctl compute ssh-key delete 386734086`

	cmdSSHKeysUpdate := CmdBuilder(cmd, RunKeyUpdate, "update <key-id|key-fingerprint>", "Update an SSH key's name", `Updates the name of an SSH key.`, Writer,
		aliasOpt("u"), displayerType(&displayers.Key{}))
	AddStringFlag(cmdSSHKeysUpdate, doctl.ArgKeyName, "", "", "Key name", requiredOpt())
	cmdSSHKeysUpdate.Example = `The following example updates the name of the SSH key with the ID ` + "`" + `386734086` + "`" + ` to ` + "`" + `new-key-name` + "`" + `: doctl compute ssh-key update 386734086 --key-name new-key-name`

	return cmd
}

// RunKeyList lists keys.
func RunKeyList(c *CmdConfig) error {
	ks := c.Keys()

	list, err := ks.List()
	if err != nil {
		return err
	}

	item := &displayers.Key{Keys: list}
	return c.Display(item)
}

// RunKeyGet retrieves a key.
func RunKeyGet(c *CmdConfig) error {
	ks := c.Keys()

	err := ensureOneArg(c)
	if err != nil {
		return err
	}

	rawKey := c.Args[0]
	k, err := ks.Get(rawKey)

	if err != nil {
		return err
	}

	item := &displayers.KeyGet{Keys: do.SSHKeys{*k}}
	return c.Display(item)
}

// RunKeyCreate uploads a SSH key.
func RunKeyCreate(c *CmdConfig) error {
	ks := c.Keys()

	err := ensureOneArg(c)
	if err != nil {
		return err
	}

	name := c.Args[0]

	publicKey, err := c.Doit.GetString(c.NS, doctl.ArgKeyPublicKey)
	if err != nil {
		return err
	}

	kcr := &godo.KeyCreateRequest{
		Name:      name,
		PublicKey: publicKey,
	}

	r, err := ks.Create(kcr)
	if err != nil {
		return err
	}

	item := &displayers.Key{Keys: do.SSHKeys{*r}}
	return c.Display(item)
}

// RunKeyImport imports a key from a file
func RunKeyImport(c *CmdConfig) error {
	ks := c.Keys()

	err := ensureOneArg(c)
	if err != nil {
		return err
	}

	keyPath, err := c.Doit.GetString(c.NS, doctl.ArgKeyPublicKeyFile)
	if err != nil {
		return err
	}

	keyName := c.Args[0]

	keyFile, err := os.ReadFile(keyPath)
	if err != nil {
		return err
	}

	_, comment, _, _, err := ssh.ParseAuthorizedKey(keyFile)
	if err != nil {
		return err
	}

	if len(keyName) < 1 {
		keyName = comment
	}

	kcr := &godo.KeyCreateRequest{
		Name:      keyName,
		PublicKey: string(keyFile),
	}

	r, err := ks.Create(kcr)
	if err != nil {
		return err
	}

	item := &displayers.Key{Keys: do.SSHKeys{*r}}
	return c.Display(item)
}

// RunKeyDelete deletes a key.
func RunKeyDelete(c *CmdConfig) error {
	ks := c.Keys()

	err := ensureOneArg(c)
	if err != nil {
		return err
	}

	force, err := c.Doit.GetBool(c.NS, doctl.ArgForce)
	if err != nil {
		return nil
	}

	if force || AskForConfirmDelete("SSH key", 1) == nil {
		rawKey := c.Args[0]
		return ks.Delete(rawKey)
	}

	return errOperationAborted
}

// RunKeyUpdate updates a key.
func RunKeyUpdate(c *CmdConfig) error {
	ks := c.Keys()

	err := ensureOneArg(c)
	if err != nil {
		return err
	}

	rawKey := c.Args[0]

	name, err := c.Doit.GetString(c.NS, doctl.ArgKeyName)
	if err != nil {
		return err
	}

	req := &godo.KeyUpdateRequest{
		Name: name,
	}

	k, err := ks.Update(rawKey, req)
	if err != nil {
		return err
	}

	item := &displayers.Key{Keys: do.SSHKeys{*k}}
	return c.Display(item)
}

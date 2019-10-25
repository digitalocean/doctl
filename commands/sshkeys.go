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
	"fmt"
	"io/ioutil"

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
			Short:   "Provides commands that manage SSH keys on your account",
			Long:    `The sub-commands of 'doctl compute ssh-key' manage the SSH keys on your account.

DigitalOcean allows you to add SSH public keys to the interface so that you can embed your public key into a Droplet at the time of creation. Only the public key is required to take advantage of this functionality. Note that this command does not add, delete, or otherwise modify any ssh keys that may be on existing Droplets.`,
		},
	}

	CmdBuilderWithDocs(cmd, RunKeyList, "list", "List all SSH keys on your account",`Use this command to list the id, fingerprint, public_key, and name of all SSH keys on your account.`, Writer,
		aliasOpt("ls"), displayerType(&displayers.Key{}))

	CmdBuilderWithDocs(cmd, RunKeyGet, "get <key-id|key-fingerprint>", "Retrive information about an SSH key on your account",`Use this command to get the id, fingerprint, public_key, and name of a specific SSH key on your account.`, Writer,
		aliasOpt("g"), displayerType(&displayers.Key{}))

	cmdSSHKeysCreate := CmdBuilderWithDocs(cmd, RunKeyCreate, "create <key-name>", "Create a new SSH key on your account",`Use this command to add a new SSH key to your account.

Set the "name" attribute to the name you wish to use and the "public_key" attribute to a string of the full public key you are adding. 
Note that this command will not add an ssh key to any existing Droplets.`, Writer,
		aliasOpt("c"), displayerType(&displayers.Key{}))
	AddStringFlag(cmdSSHKeysCreate, doctl.ArgKeyPublicKey, "", "", "Key contents", requiredOpt())

	cmdSSHKeysImport := CmdBuilderWithDocs(cmd, RunKeyImport, "import <key-name>", "Import an SSH key from your computer to your account",`Use this command to add a new SSH key to your account, using a local public key file. 

Note that this command will not add an ssh key to any existing Droplets`, Writer,
		aliasOpt("i"), displayerType(&displayers.Key{}))
	AddStringFlag(cmdSSHKeysImport, doctl.ArgKeyPublicKeyFile, "", "", "Public key file", requiredOpt())

	cmdRunKeyDelete := CmdBuilderWithDocs(cmd, RunKeyDelete, "delete <key-id|key-fingerprint>", "Permanently delete an SSH key from your account",`Use this command to permanently delete an ssh key from your account. 
	
Note that this does not delete an ssh key from any Droplets.`, Writer,
		aliasOpt("d"))
	AddBoolFlag(cmdRunKeyDelete, doctl.ArgForce, doctl.ArgShortForce, false, "Force ssh key delete")

	cmdSSHKeysUpdate := CmdBuilderWithDocs(cmd, RunKeyUpdate, "update <key-id|key-fingerprint>", "Update an SSH key's name",`Use this command to update the name of an ssh key on your account.`, Writer,
		aliasOpt("u"), displayerType(&displayers.Key{}))
	AddStringFlag(cmdSSHKeysUpdate, doctl.ArgKeyName, "", "", "Key name", requiredOpt())

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

	if len(c.Args) != 1 {
		return doctl.NewMissingArgsErr(c.NS)
	}

	rawKey := c.Args[0]
	k, err := ks.Get(rawKey)

	if err != nil {
		return err
	}

	item := &displayers.Key{Keys: do.SSHKeys{*k}}
	return c.Display(item)
}

// RunKeyCreate uploads a SSH key.
func RunKeyCreate(c *CmdConfig) error {
	ks := c.Keys()

	if len(c.Args) != 1 {
		return doctl.NewMissingArgsErr(c.NS)
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

	if len(c.Args) != 1 {
		return doctl.NewMissingArgsErr(c.NS)
	}

	keyPath, err := c.Doit.GetString(c.NS, doctl.ArgKeyPublicKeyFile)
	if err != nil {
		return err
	}

	keyName := c.Args[0]

	keyFile, err := ioutil.ReadFile(keyPath)
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

	if len(c.Args) != 1 {
		return doctl.NewMissingArgsErr(c.NS)
	}

	force, err := c.Doit.GetBool(c.NS, doctl.ArgForce)
	if err != nil {
		return nil
	}

	if force || AskForConfirm("delete ssh key") == nil {
		rawKey := c.Args[0]
		return ks.Delete(rawKey)
	}

	return fmt.Errorf("operation aborted")
}

// RunKeyUpdate updates a key.
func RunKeyUpdate(c *CmdConfig) error {
	ks := c.Keys()

	if len(c.Args) != 1 {
		return doctl.NewMissingArgsErr(c.NS)
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

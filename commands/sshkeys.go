package commands

import (
	"io/ioutil"

	"github.com/bryanl/doit"
	"github.com/bryanl/doit/do"
	"github.com/digitalocean/godo"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"
)

// SSHKeys creates the ssh key commands heirarchy.
func SSHKeys() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:     "ssh-key",
			Aliases: []string{"k"},
			Short:   "sshkey commands",
			Long:    "sshkey is used to access ssh key commands",
		},
	}

	CmdBuilder(cmd, RunKeyList, "list", "list ssh keys", Writer,
		aliasOpt("ls"), displayerType(&key{}), docCategories("sshkeys"))

	CmdBuilder(cmd, RunKeyGet, "get <key-id|key-fingerprint>", "get ssh key", Writer,
		aliasOpt("g"), displayerType(&key{}), docCategories("sshkeys"))

	cmdSSHKeysCreate := CmdBuilder(cmd, RunKeyCreate, "create <key-name>", "create ssh key", Writer,
		aliasOpt("c"), displayerType(&key{}), docCategories("sshkeys"))
	AddStringFlag(cmdSSHKeysCreate, doit.ArgKeyPublicKey, "", "Key contents", requiredOpt())

	cmdSSHKeysImport := CmdBuilder(cmd, RunKeyImport, "import <key-name>", "import ssh key", Writer,
		aliasOpt("i"), displayerType(&key{}), docCategories("sshkeys"))
	AddStringFlag(cmdSSHKeysImport, doit.ArgKeyPublicKeyFile, "", "Public key file", requiredOpt())

	CmdBuilder(cmd, RunKeyDelete, "delete <key-id|key-fingerprint>", "delete ssh key", Writer,
		aliasOpt("d"), docCategories("sshkeys"))

	cmdSSHKeysUpdate := CmdBuilder(cmd, RunKeyUpdate, "update <key-id|key-fingerprint>", "update ssh key", Writer,
		aliasOpt("u"), displayerType(&key{}), docCategories("sshkeys"))
	AddStringFlag(cmdSSHKeysUpdate, doit.ArgKeyName, "", "Key name", requiredOpt())

	return cmd
}

// RunKeyList lists keys.
func RunKeyList(c *CmdConfig) error {
	ks := c.Keys()

	list, err := ks.List()
	if err != nil {
		return err
	}

	item := &key{keys: list}
	return c.Display(item)
}

// RunKeyGet retrieves a key.
func RunKeyGet(c *CmdConfig) error {
	ks := c.Keys()

	if len(c.Args) != 1 {
		return doit.NewMissingArgsErr(c.NS)
	}

	rawKey := c.Args[0]
	k, err := ks.Get(rawKey)

	if err != nil {
		return err
	}

	item := &key{keys: do.SSHKeys{*k}}
	return c.Display(item)
}

// RunKeyCreate uploads a SSH key.
func RunKeyCreate(c *CmdConfig) error {
	ks := c.Keys()

	if len(c.Args) != 1 {
		return doit.NewMissingArgsErr(c.NS)
	}

	name := c.Args[0]

	publicKey, err := c.Doit.GetString(c.NS, doit.ArgKeyPublicKey)
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

	item := &key{keys: do.SSHKeys{*r}}
	return c.Display(item)
}

// RunKeyImport imports a key from a file
func RunKeyImport(c *CmdConfig) error {
	ks := c.Keys()

	if len(c.Args) != 1 {
		return doit.NewMissingArgsErr(c.NS)
	}

	keyPath, err := c.Doit.GetString(c.NS, doit.ArgKeyPublicKeyFile)
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

	item := &key{keys: do.SSHKeys{*r}}
	return c.Display(item)
}

// RunKeyDelete deletes a key.
func RunKeyDelete(c *CmdConfig) error {
	ks := c.Keys()

	if len(c.Args) != 1 {
		return doit.NewMissingArgsErr(c.NS)
	}

	rawKey := c.Args[0]
	return ks.Delete(rawKey)
}

// RunKeyUpdate updates a key.
func RunKeyUpdate(c *CmdConfig) error {
	ks := c.Keys()

	if len(c.Args) != 1 {
		return doit.NewMissingArgsErr(c.NS)
	}

	rawKey := c.Args[0]

	name, err := c.Doit.GetString(c.NS, doit.ArgKeyName)
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

	item := &key{keys: do.SSHKeys{*k}}
	return c.Display(item)
}

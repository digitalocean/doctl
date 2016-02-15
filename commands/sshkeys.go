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
func SSHKeys() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "ssh-key",
		Aliases: []string{"k"},
		Short:   "sshkey commands",
		Long:    "sshkey is used to access ssh key commands",
	}

	cmdBuilder2(cmd, RunKeyList, "list", "list ssh keys", writer,
		aliasOpt("ls"), displayerType(&key{}))

	cmdBuilder2(cmd, RunKeyGet, "get <key-id|key-fingerprint>", "get ssh key", writer,
		aliasOpt("g"), displayerType(&key{}))

	cmdSSHKeysCreate := cmdBuilder2(cmd, RunKeyCreate, "create <key-name>", "create ssh key", writer,
		aliasOpt("c"), displayerType(&key{}))
	addStringFlag(cmdSSHKeysCreate, doit.ArgKeyPublicKey, "", "Key contents", requiredOpt())

	cmdSSHKeysImport := cmdBuilder2(cmd, RunKeyImport, "import <key-name>", "import ssh key", writer,
		aliasOpt("i"), displayerType(&key{}))
	addStringFlag(cmdSSHKeysImport, doit.ArgKeyPublicKeyFile, "", "Public key file", requiredOpt())

	cmdBuilder2(cmd, RunKeyDelete, "delete <key-id|key-fingerprint>", "delete ssh key", writer, aliasOpt("d"))

	cmdSSHKeysUpdate := cmdBuilder2(cmd, RunKeyUpdate, "update <key-id|key-fingerprint>", "update ssh key", writer,
		aliasOpt("u"), displayerType(&key{}))
	addStringFlag(cmdSSHKeysUpdate, doit.ArgKeyName, "", "Key name", requiredOpt())

	return cmd
}

// RunKeyList lists keys.
func RunKeyList(c *cmdConfig) error {
	ks := c.keysService()

	list, err := ks.List()
	if err != nil {
		return err
	}

	item := &key{keys: list}
	return c.display(item)
}

// RunKeyGet retrieves a key.
func RunKeyGet(c *cmdConfig) error {
	ks := c.keysService()

	if len(c.args) != 1 {
		return doit.NewMissingArgsErr(c.ns)
	}

	rawKey := c.args[0]
	k, err := ks.Get(rawKey)

	if err != nil {
		return err
	}

	item := &key{keys: do.SSHKeys{*k}}
	return c.display(item)
}

// RunKeyCreate uploads a SSH key.
func RunKeyCreate(c *cmdConfig) error {
	ks := c.keysService()

	if len(c.args) != 1 {
		return doit.NewMissingArgsErr(c.ns)
	}

	name := c.args[0]

	publicKey, err := c.doitConfig.GetString(c.ns, doit.ArgKeyPublicKey)
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
	return c.display(item)
}

// RunKeyImport imports a key from a file
func RunKeyImport(c *cmdConfig) error {
	ks := c.keysService()

	if len(c.args) != 1 {
		return doit.NewMissingArgsErr(c.ns)
	}

	keyPath, err := c.doitConfig.GetString(c.ns, doit.ArgKeyPublicKeyFile)
	if err != nil {
		return err
	}

	keyName := c.args[0]

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
	return c.display(item)
}

// RunKeyDelete deletes a key.
func RunKeyDelete(c *cmdConfig) error {
	ks := c.keysService()

	if len(c.args) != 1 {
		return doit.NewMissingArgsErr(c.ns)
	}

	rawKey := c.args[0]
	return ks.Delete(rawKey)
}

// RunKeyUpdate updates a key.
func RunKeyUpdate(c *cmdConfig) error {
	ks := c.keysService()

	if len(c.args) != 1 {
		return doit.NewMissingArgsErr(c.ns)
	}

	rawKey := c.args[0]

	name, err := c.doitConfig.GetString(c.ns, doit.ArgKeyName)
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
	return c.display(item)
}

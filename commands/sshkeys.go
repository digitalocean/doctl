package commands

import (
	"io"
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

	cmdBuilder(cmd, RunKeyList, "list", "list ssh keys", writer,
		aliasOpt("ls"), displayerType(&key{}))

	cmdBuilder(cmd, RunKeyGet, "get <key-id|key-fingerprint>", "get ssh key", writer,
		aliasOpt("g"), displayerType(&key{}))

	cmdSSHKeysCreate := cmdBuilder(cmd, RunKeyCreate, "create <key-name>", "create ssh key", writer,
		aliasOpt("c"), displayerType(&key{}))
	addStringFlag(cmdSSHKeysCreate, doit.ArgKeyPublicKey, "", "Key contents", requiredOpt())

	cmdSSHKeysImport := cmdBuilder(cmd, RunKeyImport, "import <key-name>", "import ssh key", writer,
		aliasOpt("i"), displayerType(&key{}))
	addStringFlag(cmdSSHKeysImport, doit.ArgKeyPublicKeyFile, "", "Public key file", requiredOpt())

	cmdBuilder(cmd, RunKeyDelete, "delete <key-id|key-fingerprint>", "delete ssh key", writer, aliasOpt("d"))

	cmdSSHKeysUpdate := cmdBuilder(cmd, RunKeyUpdate, "update <key-id|key-fingerprint>", "update ssh key", writer,
		aliasOpt("u"), displayerType(&key{}))
	addStringFlag(cmdSSHKeysUpdate, doit.ArgKeyName, "", "Key name", requiredOpt())

	return cmd
}

// RunKeyList lists keys.
func RunKeyList(ns string, config doit.Config, out io.Writer, args []string) error {
	client := config.GetGodoClient()
	ks := do.NewKeysService(client)

	list, err := ks.List()
	if err != nil {
		return err
	}

	item := &key{keys: []godo.Key{}}
	for _, k := range list {
		item.keys = append(item.keys, *k.Key)
	}

	dc := &outputConfig{
		ns:     ns,
		config: config,
		item:   item,
		out:    out,
	}

	return displayOutput(dc)
}

// RunKeyGet retrieves a key.
func RunKeyGet(ns string, config doit.Config, out io.Writer, args []string) error {
	client := config.GetGodoClient()
	ks := do.NewKeysService(client)

	if len(args) != 1 {
		return doit.NewMissingArgsErr(ns)
	}

	rawKey := args[0]
	k, err := ks.Get(rawKey)

	if err != nil {
		return err
	}

	dc := &outputConfig{
		ns:     ns,
		config: config,
		item:   &key{keys: keys{*k.Key}},
		out:    out,
	}

	return displayOutput(dc)
}

// RunKeyCreate uploads a SSH key.
func RunKeyCreate(ns string, config doit.Config, out io.Writer, args []string) error {
	client := config.GetGodoClient()
	ks := do.NewKeysService(client)

	if len(args) != 1 {
		return doit.NewMissingArgsErr(ns)
	}

	name := args[0]

	publicKey, err := config.GetString(ns, doit.ArgKeyPublicKey)
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

	dc := &outputConfig{
		ns:     ns,
		config: config,
		item:   &key{keys: keys{*r.Key}},
		out:    out,
	}

	return displayOutput(dc)
}

// RunKeyImport imports a key from a file
func RunKeyImport(ns string, config doit.Config, out io.Writer, args []string) error {
	client := config.GetGodoClient()
	ks := do.NewKeysService(client)

	if len(args) != 1 {
		return doit.NewMissingArgsErr(ns)
	}

	keyPath, err := config.GetString(ns, doit.ArgKeyPublicKeyFile)
	if err != nil {
		return err
	}

	keyName := args[0]

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

	dc := &outputConfig{
		ns:     ns,
		config: config,
		item:   &key{keys: keys{*r.Key}},
		out:    out,
	}

	return displayOutput(dc)
}

// RunKeyDelete deletes a key.
func RunKeyDelete(ns string, config doit.Config, out io.Writer, args []string) error {
	client := config.GetGodoClient()
	ks := do.NewKeysService(client)

	if len(args) != 1 {
		return doit.NewMissingArgsErr(ns)
	}

	rawKey := args[0]
	return ks.Delete(rawKey)
}

// RunKeyUpdate updates a key.
func RunKeyUpdate(ns string, config doit.Config, out io.Writer, args []string) error {
	client := config.GetGodoClient()
	ks := do.NewKeysService(client)

	if len(args) != 1 {
		return doit.NewMissingArgsErr(ns)
	}

	rawKey := args[0]

	name, err := config.GetString(ns, doit.ArgKeyName)
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

	dc := &outputConfig{
		ns:     ns,
		config: config,
		item:   &key{keys: keys{*k.Key}},
		out:    out,
	}

	return displayOutput(dc)
}

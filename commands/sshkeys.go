package commands

import (
	"fmt"
	"io"
	"io/ioutil"
	"strconv"

	"github.com/bryanl/doit"
	"github.com/bryanl/doit/Godeps/_workspace/src/github.com/Sirupsen/logrus"
	"github.com/bryanl/doit/Godeps/_workspace/src/github.com/digitalocean/godo"
	"github.com/bryanl/doit/Godeps/_workspace/src/github.com/spf13/cobra"
	"github.com/bryanl/doit/Godeps/_workspace/src/golang.org/x/crypto/ssh"
)

// SSHKeys creates the ssh key commands heirarchy.
func SSHKeys() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "sshkey",
		Aliases: []string{"k"},
		Short:   "sshkey commands",
		Long:    "sshkey is used to access ssh key commands",
	}

	cmdSSHKeysList := cmdBuilder(RunKeyList, "list", "list ssh keys", writer, "ls")
	cmd.AddCommand(cmdSSHKeysList)

	cmdSSHKeysGet := cmdBuilder(RunKeyGet, "get", "get ssh key", writer, "g")
	cmd.AddCommand(cmdSSHKeysGet)
	addStringFlag(cmdSSHKeysGet, doit.ArgKey, "", "Key ID or fingerprint")

	cmdSSHKeysCreate := cmdBuilder(RunKeyCreate, "create", "create ssh key", writer, "c")
	cmd.AddCommand(cmdSSHKeysCreate)
	addStringFlag(cmdSSHKeysCreate, doit.ArgKeyName, "", "Key name")
	addStringFlag(cmdSSHKeysCreate, doit.ArgKeyPublicKey, "", "Key contents")

	cmdSSHKeysImport := cmdBuilder(RunKeyImport, "import", "import ssh key", writer, "i")
	cmd.AddCommand(cmdSSHKeysImport)
	addStringFlag(cmdSSHKeysImport, doit.ArgKeyName, "", "Key name")
	addStringFlag(cmdSSHKeysImport, doit.ArgKeyPublicKeyFile, "", "Public key file")

	cmdSSHKeysDelete := cmdBuilder(RunKeyDelete, "delete", "delete ssh key", writer, "d")
	cmd.AddCommand(cmdSSHKeysDelete)
	addStringFlag(cmdSSHKeysDelete, doit.ArgKey, "", "Key ID or fingerprint")

	cmdSSHKeysUpdate := cmdBuilder(RunKeyUpdate, "update", "update ssh key", writer, "u")
	cmd.AddCommand(cmdSSHKeysUpdate)
	addStringFlag(cmdSSHKeysUpdate, doit.ArgKey, "", "Key ID or fingerprint")
	addStringFlag(cmdSSHKeysUpdate, doit.ArgKeyName, "", "Key name")

	return cmd
}

// RunKeyList lists keys.
func RunKeyList(ns string, config doit.Config, out io.Writer) error {
	client := config.GetGodoClient()

	f := func(opt *godo.ListOptions) ([]interface{}, *godo.Response, error) {
		list, resp, err := client.Keys.List(opt)
		if err != nil {
			return nil, nil, err
		}

		si := make([]interface{}, len(list))
		for i := range list {
			si[i] = list[i]
		}

		return si, resp, err
	}

	si, err := doit.PaginateResp(f)
	if err != nil {
		return err
	}

	list := make([]godo.Key, len(si))
	for i := range si {
		list[i] = si[i].(godo.Key)
	}

	return doit.DisplayOutput(list, out)
}

// RunKeyGet retrieves a key.
func RunKeyGet(ns string, config doit.Config, out io.Writer) error {
	client := config.GetGodoClient()
	rawKey, err := config.GetString(ns, doit.ArgKey)
	if err != nil {
		return err
	}

	var key *godo.Key
	if i, aerr := strconv.Atoi(rawKey); aerr == nil {
		key, _, err = client.Keys.GetByID(i)
	} else {
		if len(rawKey) > 0 {
			key, _, err = client.Keys.GetByFingerprint(rawKey)
		} else {
			err = fmt.Errorf("missing key id or fingerprint")
		}
	}

	if err != nil {
		return err
	}

	return doit.DisplayOutput(key, out)
}

// RunKeyCreate uploads a SSH key.
func RunKeyCreate(ns string, config doit.Config, out io.Writer) error {
	client := config.GetGodoClient()

	name, err := config.GetString(ns, doit.ArgKeyName)
	if err != nil {
		return err
	}

	publicKey, err := config.GetString(ns, doit.ArgKeyPublicKey)
	if err != nil {
		return err
	}

	kcr := &godo.KeyCreateRequest{
		Name:      name,
		PublicKey: publicKey,
	}

	r, _, err := client.Keys.Create(kcr)
	if err != nil {
		logrus.WithField("err", err).Fatal("could not create key")
	}

	return doit.DisplayOutput(r, out)
}

// RunKeyImport imports a key from a file
func RunKeyImport(ns string, config doit.Config, out io.Writer) error {
	client := config.GetGodoClient()

	keyPath, err := config.GetString(ns, doit.ArgKeyPublicKeyFile)
	if err != nil {
		return err
	}

	keyName, err := config.GetString(ns, doit.ArgKeyName)
	if err != nil {
		return err
	}

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

	r, _, err := client.Keys.Create(kcr)
	if err != nil {
		return err
	}

	return doit.DisplayOutput(r, out)
}

// RunKeyDelete deletes a key.
func RunKeyDelete(ns string, config doit.Config, out io.Writer) error {
	client := config.GetGodoClient()
	rawKey, err := config.GetString(ns, doit.ArgKey)
	if err != nil {
		return err
	}

	if i, aerr := strconv.Atoi(rawKey); aerr == nil {
		_, err = client.Keys.DeleteByID(i)
	} else {
		_, err = client.Keys.DeleteByFingerprint(rawKey)
	}

	return err
}

// RunKeyUpdate updates a key.
func RunKeyUpdate(ns string, config doit.Config, out io.Writer) error {
	client := config.GetGodoClient()
	rawKey, err := config.GetString(ns, doit.ArgKey)
	if err != nil {
		return err
	}

	name, err := config.GetString(ns, doit.ArgKeyName)
	if err != nil {
		return err
	}

	req := &godo.KeyUpdateRequest{
		Name: name,
	}

	var key *godo.Key
	if i, aerr := strconv.Atoi(rawKey); aerr == nil {
		key, _, err = client.Keys.UpdateByID(i, req)
	} else {
		key, _, err = client.Keys.UpdateByFingerprint(rawKey, req)
	}

	if err != nil {
		return err
	}

	return doit.DisplayOutput(key, out)
}

package commands

import (
	"fmt"
	"io"
	"io/ioutil"
	"strconv"

	"github.com/Sirupsen/logrus"
	"github.com/bryanl/doit"
	"github.com/digitalocean/godo"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"
)

// SSHKeys creates the ssh key commands heirarchy.
func SSHKeys() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sshkey",
		Short: "sshkey commands",
		Long:  "sshkey is used to access ssh key commands",
	}

	cmdSSHKeysList := cmdBuilder(RunKeyList, "list", "list ssh keys", writer)
	cmd.AddCommand(cmdSSHKeysList)

	cmdSSHKeysGet := cmdBuilder(RunKeyGet, "get", "get ssh key", writer)
	cmd.AddCommand(cmdSSHKeysGet)
	addStringFlag(cmdSSHKeysGet, doit.ArgKey, "", "Key ID or fingerprint")

	cmdSSHKeysCreate := cmdBuilder(RunKeyCreate, "create", "create ssh key", writer)
	cmd.AddCommand(cmdSSHKeysCreate)
	addStringFlag(cmdSSHKeysCreate, doit.ArgKeyName, "", "Key name")
	addStringFlag(cmdSSHKeysCreate, doit.ArgKeyPublicKey, "", "Key contents")

	cmdSSHKeysImport := cmdBuilder(RunKeyImport, "import", "import ssh key", writer)
	cmd.AddCommand(cmdSSHKeysImport)
	addStringFlag(cmdSSHKeysImport, doit.ArgKeyName, "", "Key name")
	addStringFlag(cmdSSHKeysImport, doit.ArgKeyPublicKeyFile, "", "Public key file")

	cmdSSHKeysDelete := cmdBuilder(RunKeyDelete, "delete", "delete ssh key", writer)
	cmd.AddCommand(cmdSSHKeysDelete)
	addStringFlag(cmdSSHKeysDelete, doit.ArgKey, "", "Key ID or fingerprint")

	cmdSSHKeysUpdate := cmdBuilder(RunKeyUpdate, "update", "update ssh key", writer)
	cmd.AddCommand(cmdSSHKeysUpdate)
	addStringFlag(cmdSSHKeysUpdate, doit.ArgKey, "", "Key ID or fingerprint")
	addStringFlag(cmdSSHKeysUpdate, doit.ArgKeyName, "", "Key name")

	return cmd
}

// RunKeyList lists keys.
func RunKeyList(ns string, out io.Writer) error {
	client := doit.DoitConfig.GetGodoClient()

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
func RunKeyGet(ns string, out io.Writer) error {
	client := doit.DoitConfig.GetGodoClient()
	rawKey := doit.DoitConfig.GetString(ns, doit.ArgKey)

	var err error
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
func RunKeyCreate(ns string, out io.Writer) error {
	client := doit.DoitConfig.GetGodoClient()

	kcr := &godo.KeyCreateRequest{
		Name:      doit.DoitConfig.GetString(ns, doit.ArgKeyName),
		PublicKey: doit.DoitConfig.GetString(ns, doit.ArgKeyPublicKey),
	}

	r, _, err := client.Keys.Create(kcr)
	if err != nil {
		logrus.WithField("err", err).Fatal("could not create key")
	}

	return doit.DisplayOutput(r, out)
}

// RunKeyImport imports a key from a file
func RunKeyImport(ns string, out io.Writer) error {
	client := doit.DoitConfig.GetGodoClient()

	keyPath := doit.DoitConfig.GetString(ns, doit.ArgKeyPublicKeyFile)
	keyName := doit.DoitConfig.GetString(ns, doit.ArgKeyName)

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
func RunKeyDelete(ns string, out io.Writer) error {
	client := doit.DoitConfig.GetGodoClient()
	rawKey := doit.DoitConfig.GetString(ns, doit.ArgKey)

	var err error
	if i, aerr := strconv.Atoi(rawKey); aerr == nil {
		_, err = client.Keys.DeleteByID(i)
	} else {
		_, err = client.Keys.DeleteByFingerprint(rawKey)
	}

	return err
}

// RunKeyUpdate updates a key.
func RunKeyUpdate(ns string, out io.Writer) error {
	client := doit.DoitConfig.GetGodoClient()
	rawKey := doit.DoitConfig.GetString(ns, doit.ArgKey)

	req := &godo.KeyUpdateRequest{
		Name: doit.DoitConfig.GetString(ns, doit.ArgKeyName),
	}

	var err error
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

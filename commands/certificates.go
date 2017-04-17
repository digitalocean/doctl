/*
Copyright 2017 The Doctl Authors All rights reserved.
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
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"

	"github.com/spf13/cobra"
)

// Certificate creates the certificate command.
func Certificate() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:   "certificate",
			Short: "certificate commands",
			Long:  "certificate is used to access certificate commands",
		},
	}

	CmdBuilder(cmd, RunCertificateGet, "get <id>", "get certificate", Writer, aliasOpt("g"))

	cmdCertificateCreate := CmdBuilder(cmd, RunCertificateCreate, "create", "create new certificate", Writer, aliasOpt("c"))
	AddStringFlag(cmdCertificateCreate, doctl.ArgCertificateName, "", "", "certificate name", requiredOpt())
	AddStringFlag(cmdCertificateCreate, doctl.ArgPrivateKeyPath, "", "", "path to a private key for the certificate", requiredOpt())
	AddStringFlag(cmdCertificateCreate, doctl.ArgLeafCertificatePath, "", "", "path to a certificate leaf", requiredOpt())
	AddStringFlag(cmdCertificateCreate, doctl.ArgCertificateChainPath, "", "", "path to a certificate chain")

	CmdBuilder(cmd, RunCertificateList, "list", "list certificates", Writer, aliasOpt("ls"))

	cmdCertificateDelete := CmdBuilder(cmd, RunCertificateDelete, "delete <id>", "delete certificate", Writer, aliasOpt("d", "rm"))
	AddBoolFlag(cmdCertificateDelete, doctl.ArgDeleteForce, doctl.ArgShortDeleteForce, false, "Force certificate delete")

	return cmd
}

// RunCertificateGet retrieves an existing certificate by its identifier.
func RunCertificateGet(c *CmdConfig) error {
	if len(c.Args) != 1 {
		return doctl.NewMissingArgsErr(c.NS)
	}
	cID := c.Args[0]

	cs := c.Certificates()
	cer, err := cs.Get(cID)
	if err != nil {
		return err
	}

	item := &certificate{certificates: do.Certificates{*cer}}
	return c.Display(item)
}

// RunCertificateCreate creates a certificate.
func RunCertificateCreate(c *CmdConfig) error {
	name, err := c.Doit.GetString(c.NS, doctl.ArgCertificateName)
	if err != nil {
		return err
	}

	pkPath, err := c.Doit.GetString(c.NS, doctl.ArgPrivateKeyPath)
	if err != nil {
		return err
	}

	pc, err := readInputFromFile(pkPath)
	if err != nil {
		return err
	}

	lcPath, err := c.Doit.GetString(c.NS, doctl.ArgLeafCertificatePath)
	if err != nil {
		return err
	}

	lc, err := readInputFromFile(lcPath)
	if err != nil {
		return err
	}

	r := &godo.CertificateRequest{
		Name:            name,
		PrivateKey:      pc,
		LeafCertificate: lc,
	}

	ccPath, err := c.Doit.GetString(c.NS, doctl.ArgCertificateChainPath)
	if err != nil {
		return err
	}

	if len(ccPath) > 0 {
		cc, err := readInputFromFile(ccPath)
		if err != nil {
			return err
		}

		r.CertificateChain = cc
	}

	cs := c.Certificates()
	cer, err := cs.Create(r)
	if err != nil {
		return err
	}

	item := &certificate{certificates: do.Certificates{*cer}}
	return c.Display(item)
}

// RunCertificateList lists certificates.
func RunCertificateList(c *CmdConfig) error {
	cs := c.Certificates()
	list, err := cs.List()
	if err != nil {
		return err
	}

	item := &certificate{certificates: list}
	return c.Display(item)
}

// RunCertificateDelete deletes a certificate by its identifier.
func RunCertificateDelete(c *CmdConfig) error {
	if len(c.Args) != 1 {
		return doctl.NewMissingArgsErr(c.NS)
	}
	cID := c.Args[0]

	force, err := c.Doit.GetBool(c.NS, doctl.ArgDeleteForce)
	if err != nil {
		return err
	}

	if force || AskForConfirm("delete this certificate") == nil {
		cs := c.Certificates()
		if err := cs.Delete(cID); err != nil {
			return err
		}
	} else {
		return fmt.Errorf("operation aborted")
	}

	return nil
}

func readInputFromFile(path string) (string, error) {
	fileBytes, err := ioutil.ReadFile(path)
	if err != nil {
		return "", err
	}

	return string(fileBytes), nil
}

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
)

// Certificate creates the certificate command.
func Certificate() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:   "certificate",
			Short: "Provides commands that manage SSL certificates and private keys",
			Long: `The subcommands of 'doctl compute certificate' allow you to store and manage your SSL certificates, private keys, and certificate paths.

After storage, they can be referred to in your doctl and API workflows by using their certificate ID.`,
		},
	}
	certDetails := `

- The certificate ID
- The name you gave the certificate
- Comma-separated list of domain names associated with the certificate
- The SHA-1 Fingerprint for the certificate
- The certificate's expiration date
- The certificate's creation date
- The certificate type
- The certificate State`
	CmdBuilderWithDocs(cmd, RunCertificateGet, "get <id>", "Retreives details about a certificate", `This command retrieves the following details about a certificate:`+certDetails, Writer,
		aliasOpt("g"), displayerType(&displayers.Certificate{}))
	cmdCertificateCreate := CmdBuilderWithDocs(cmd, RunCertificateCreate, "create",
		"Creates a new certificate", `This command allows you to create a certificate. There are two supported certificate types: Let's Encrypt certificates, and custom certificates.

Let's Encrypt certificates are free and will be auto-renewed and managed for you by DigitalOcean.

To create a Let's Encrypt certificate, you'll need to add the domain(s) to your account at cloud.digitalocean.com, or via 'doctl compute domain create', then provide a certificate name and a comma-separated list of the domain names you'd like to associate with the certificate:

	doctl compute certificate create --type lets_encrypt --name mycert --dns-names example.org

To upload a custom certificate, you'll need to provide a certificate name, the path to the certificate, the path to the private key for the certificate, and the path to the certificate chain:

	doctl compute certificate create --type custom --name mycert --leaf-certificate-path cert.pem --certificate-chain-path fullchain.pem --private-key-path privkey.pem`, Writer, aliasOpt("c"))
	AddStringFlag(cmdCertificateCreate, doctl.ArgCertificateName, "", "",
		"Certificate name", requiredOpt())
	AddStringSliceFlag(cmdCertificateCreate, doctl.ArgCertificateDNSNames, "",
		[]string{}, "Comma-separated list of domain names, required for lets_encrypt certificate")
	AddStringFlag(cmdCertificateCreate, doctl.ArgPrivateKeyPath, "", "",
		"Path to a private key for the certificate, required for custom certificate")
	AddStringFlag(cmdCertificateCreate, doctl.ArgLeafCertificatePath, "", "",
		"Path to a certificate leaf, required for custom certificate")
	AddStringFlag(cmdCertificateCreate, doctl.ArgCertificateChainPath, "", "",
		"Path to a certificate chain")
	AddStringFlag(cmdCertificateCreate, doctl.ArgCertificateType, "", "",
		"Certificate type [custom|lets_encrypt]")

	CmdBuilderWithDocs(cmd, RunCertificateList, "list", "Retrieves list of the account's stored certificates", `This command retrieves a list of all certificates associated with the account. The following details are shown for each certificate:`+certDetails, Writer,
		aliasOpt("ls"), displayerType(&displayers.Certificate{}))

	cmdCertificateDelete := CmdBuilderWithDocs(cmd, RunCertificateDelete, "delete <id>",
		"Deletes the specified certificate", `This command deletes the certificate whose ID is specified.

You can see the IDs of all available certificates associated with your account by calling 'doctl compute certificate list'.`, Writer, aliasOpt("d", "rm"))
	AddBoolFlag(cmdCertificateDelete, doctl.ArgForce, doctl.ArgShortForce, false,
		"Delete the certificate without a comfirmation prompt")

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

	item := &displayers.Certificate{Certificates: do.Certificates{*cer}}
	return c.Display(item)
}

// RunCertificateCreate creates a certificate.
func RunCertificateCreate(c *CmdConfig) error {
	name, err := c.Doit.GetString(c.NS, doctl.ArgCertificateName)
	if err != nil {
		return err
	}

	domainList, err := c.Doit.GetStringSlice(c.NS, doctl.ArgCertificateDNSNames)
	if err != nil {
		return err
	}

	cType, err := c.Doit.GetString(c.NS, doctl.ArgCertificateType)
	if err != nil {
		return err
	}

	r := &godo.CertificateRequest{
		Name:     name,
		DNSNames: domainList,
		Type:     cType,
	}

	pkPath, err := c.Doit.GetString(c.NS, doctl.ArgPrivateKeyPath)
	if err != nil {
		return err
	}

	if len(pkPath) > 0 {
		pc, err := readInputFromFile(pkPath)
		if err != nil {
			return err
		}

		r.PrivateKey = pc
	}

	lcPath, err := c.Doit.GetString(c.NS, doctl.ArgLeafCertificatePath)
	if err != nil {
		return err
	}

	if len(lcPath) > 0 {
		lc, err := readInputFromFile(lcPath)
		if err != nil {
			return err
		}

		r.LeafCertificate = lc
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

	item := &displayers.Certificate{Certificates: do.Certificates{*cer}}
	return c.Display(item)
}

// RunCertificateList lists certificates.
func RunCertificateList(c *CmdConfig) error {
	cs := c.Certificates()
	list, err := cs.List()
	if err != nil {
		return err
	}

	item := &displayers.Certificate{Certificates: list}
	return c.Display(item)
}

// RunCertificateDelete deletes a certificate by its identifier.
func RunCertificateDelete(c *CmdConfig) error {
	if len(c.Args) != 1 {
		return doctl.NewMissingArgsErr(c.NS)
	}
	cID := c.Args[0]

	force, err := c.Doit.GetBool(c.NS, doctl.ArgForce)
	if err != nil {
		return err
	}

	if force || AskForConfirm("Delete this certificate?") == nil {
		cs := c.Certificates()
		if err := cs.Delete(cID); err != nil {
			return err
		}
	} else {
		return fmt.Errorf("Operation aborted.")
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

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
			Short: "Display commands that manage SSL certificates and private keys",
			Long: `The subcommands of ` + "`" + `doctl compute certificate` + "`" + ` allow you to store and manage your SSL certificates, private keys, and certificate paths.

Once a certificate has been stored, it is assigned a unique certificate ID that can be referenced in your doctl and API workflows.`,
		},
	}
	certDetails := `

- The certificate ID
- The name you gave the certificate
- A comma-separated list of domain names associated with the certificate
- The SHA-1 fingerprint of the certificate
- The certificate's expiration date given in ISO8601 date/time format
- The certificate's creation date given in ISO8601 date/time format
- The certificate type (` + "`" + `custom` + "`" + ` or ` + "`" + `lets_encrypt` + "`" + `)
- The certificate state (` + "`" + `pending` + "`" + `, ` + "`" + `verified` + "`" + `, or ` + "`" + `error` + "`" + `)`

	CmdBuilder(cmd, RunCertificateGet, "get <id>", "Retrieve details about a certificate", `This command retrieves the following details about a certificate:`+certDetails, Writer,
		aliasOpt("g"), displayerType(&displayers.Certificate{}))
	cmdCertificateCreate := CmdBuilder(cmd, RunCertificateCreate, "create",
		"Create a new certificate", `This command allows you to create a certificate. There are two supported certificate types: Let's Encrypt certificates, and custom certificates.

Let's Encrypt certificates are free and will be auto-renewed and managed for you by DigitalOcean.

To create a Let's Encrypt certificate, you'll need to add the domain(s) to your account at cloud.digitalocean.com, or via `+"`"+`doctl compute domain create`+"`"+`, then provide a certificate name and a comma-separated list of the domain names you'd like to associate with the certificate:

	doctl compute certificate create --type lets_encrypt --name mycert --dns-names example.org

To upload a custom certificate, you'll need to provide a certificate name, the path to the certificate, the path to the private key for the certificate, and the path to the certificate chain, all in PEM format:

	doctl compute certificate create --type custom --name mycert --leaf-certificate-path cert.pem --certificate-chain-path fullchain.pem --private-key-path privkey.pem`, Writer, aliasOpt("c"))
	AddStringFlag(cmdCertificateCreate, doctl.ArgCertificateName, "", "",
		"Certificate name", requiredOpt())
	AddStringSliceFlag(cmdCertificateCreate, doctl.ArgCertificateDNSNames, "",
		[]string{}, "Comma-separated list of domains for which the certificate will be issued. The domains must be managed using DigitalOcean's DNS.")
	AddStringFlag(cmdCertificateCreate, doctl.ArgPrivateKeyPath, "", "",
		"The path to a PEM-formatted private-key corresponding to the SSL certificate.")
	AddStringFlag(cmdCertificateCreate, doctl.ArgLeafCertificatePath, "", "",
		"The path to a PEM-formatted public SSL certificate.")
	AddStringFlag(cmdCertificateCreate, doctl.ArgCertificateChainPath, "", "",
		"The path to a full PEM-formatted trust chain between the certificate authority's certificate and your domain's SSL certificate.")
	AddStringFlag(cmdCertificateCreate, doctl.ArgCertificateType, "", "",
		"Certificate type [custom|lets_encrypt]")

	CmdBuilder(cmd, RunCertificateList, "list", "Retrieve list of the account's stored certificates", `This command retrieves a list of all certificates associated with the account. The following details are shown for each certificate:`+certDetails, Writer,
		aliasOpt("ls"), displayerType(&displayers.Certificate{}))

	cmdCertificateDelete := CmdBuilder(cmd, RunCertificateDelete, "delete <id>",
		"Delete the specified certificate", `This command deletes the specified certificate.

Use `+"`"+`doctl compute certificate list`+"`"+` to see all available certificates associated with your account.`, Writer, aliasOpt("d", "rm"))
	AddBoolFlag(cmdCertificateDelete, doctl.ArgForce, doctl.ArgShortForce, false,
		"Delete the certificate without a confirmation prompt")

	return cmd
}

// RunCertificateGet retrieves an existing certificate by its identifier.
func RunCertificateGet(c *CmdConfig) error {
	err := ensureOneArg(c)
	if err != nil {
		return err
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
	err := ensureOneArg(c)
	if err != nil {
		return err
	}
	cID := c.Args[0]

	force, err := c.Doit.GetBool(c.NS, doctl.ArgForce)
	if err != nil {
		return err
	}

	if force || AskForConfirmDelete("certificate", 1) == nil {
		cs := c.Certificates()
		if err := cs.Delete(cID); err != nil {
			return err
		}
	} else {
		return errOperationAborted
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

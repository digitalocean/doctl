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
	"errors"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/commands/displayers"
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
	"github.com/spf13/cobra"
)

// CDN creates the CDN command
func CDN() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:   "cdn",
			Short: "Display commands that manage CDNs",
			Long: `The subcommands of ` + "`" + `doctl compute cdn` + "`" + ` are for managing Content Delivery Networks (CDNs).

Content hosted in DigitalOcean's object storage solution, Spaces, can optionally be served by our globally distributed CDNs. This allows you to deliver content to users based on their geographic location.

To use a custom subdomain to access the CDN endpoint, provide the ID of a DigitalOcean-managed TLS certificate and the fully qualified domain name (FQDN) for the custom subdomain.`,
		},
	}

	CDNnotes := `

The Time To Live (TTL) value is the length of time in seconds that a file is cached by the CDN before being refreshed. If a request to access a file occurs after the TTL has expired, the CDN delivers the file by requesting it directly from the origin URL, re-caching the file, and resetting the TTL.`

	CDNDetails := `

- The ID for the CDN, in UUID format
- The fully qualified domain name (FQDN) for the origin server, which provides the content to the CDN. Currently, only Spaces are supported with CDNs.
- The fully qualified domain name (FQDN) of the endpoint from which the CDN-backed content is served.
- The "Time To Live" (TTL) value for cached content, in seconds. The default is 3,600 (one hour).
- An optional custom subdomain when the CDN can be accessed
- The ID of a DigitalOcean-managed TLS certificate used for SSL when a custom subdomain is provided.
- The date and time when the CDN was created, in ISO8601 date/time format`
	TTLDesc := "The \"Time To Live\" (TTL) value for cached content, in seconds"
	DomainDesc := "Specify a custom domain to use with the CDN"
	CertIDDesc := "Specify a certificate ID for the custom domain"
	cmdCDNList := CmdBuilder(cmd, RunCDNList, "list", "List CDNs that have already been created", `Retrieves a list of your existing Content Delivery Networks (CDNs) and their following details:`+CDNDetails, Writer,
		aliasOpt("ls"), displayerType(&displayers.CDN{}))
	cmdCDNList.Example = `The following example retrieves a list of CDNs for your account. The command uses the ` + "`" + `--format` + "`" + ` flag to only return each CDN` + "`" + `'` + "`" + `s origin endpoint, CDN endpoint, and certificate ID: doctl compute cdn list --format ID,Origin,Endpoint,CertificateID`

	cmdCDNCreate := CmdBuilder(cmd, RunCDNCreate, "create <cdn-origin>", "Create a CDN", `Creates a Content Delivery Network (CDN) on the origin server you specify and automatically generates an endpoint. You can also use a custom subdomain you own to create an additional endpoint, which must be secured with SSL.`+CDNnotes, Writer,
		aliasOpt("c"), displayerType(&displayers.CDN{}))
	AddIntFlag(cmdCDNCreate, doctl.ArgCDNTTL, "", 3600, TTLDesc)
	AddStringFlag(cmdCDNCreate, doctl.ArgCDNDomain, "", "", DomainDesc)
	AddStringFlag(cmdCDNCreate, doctl.ArgCDNCertificateID, "", "", CertIDDesc)
	cmdCDNCreate.Example = `The following example creates a CDN for the custom domain ` + "`" + `cdn.example.com ` + "`" + ` using a DigitalOcean Spaces origin endpoint and SSL certificate ID for the custom domain: doctl compute cdn create https://tester-two.blr1.digitaloceanspaces.com --domain cdn.example.com --certificate-id f81d4fae-7dec-11d0-a765-00a0c91e6bf6`

	cmdRunCDNDelete := CmdBuilder(cmd, RunCDNDelete, "delete <cdn-id>", "Delete a CDN", `Deletes the CDN specified by the ID.

You can retrieve a list of CDN IDs by calling `+"`"+`doctl compute cdn list`+"`"+``, Writer,
		aliasOpt("rm"))
	AddBoolFlag(cmdRunCDNDelete, doctl.ArgForce, doctl.ArgShortForce, false, "Delete the specified CDN without prompting for confirmation")
	cmdRunCDNDelete.Example = `The following example deletes a CDN with the ID ` + "`" + `418b7972-fc67-41ea-ab4b-6f9477c4f7d8` + "`" + `: doctl compute cdn delete 418b7972-fc67-41ea-ab4b-6f9477c4f7d8`

	cmdRunCDNGet := CmdBuilder(cmd, RunCDNGet, "get <cdn-id>", "Retrieve details about a specific CDN", `Lists the following details for the specified Content Delivery Network (CDNs):`+CDNDetails+CDNnotes, Writer, aliasOpt("g"),
		displayerType(&displayers.CDN{}))
	cmdRunCDNGet.Example = `The following example retrieves the origin endpoint, CDN endpoint, and certificate ID for a CDN with the ID ` + "`" + `418b7972-fc67-41ea-ab4b-6f9477c4f7d8` + "`" + `: doctl compute cdn get 418b7972-fc67-41ea-ab4b-6f9477c4f7d8 --format ID,Origin,Endpoint,CertificateID`

	cmdCDNUpdate := CmdBuilder(cmd, RunCDNUpdate, "update <cdn-id>", "Update the configuration for a CDN", `Updates the configuration details of an existing Content Delivery Network (CDN).`, Writer,
		aliasOpt("u"), displayerType(&displayers.CDN{}))
	AddIntFlag(cmdCDNUpdate, doctl.ArgCDNTTL, "", 3600, TTLDesc)
	AddStringFlag(cmdCDNUpdate, doctl.ArgCDNDomain, "", "", DomainDesc)
	AddStringFlag(cmdCDNUpdate, doctl.ArgCDNCertificateID, "", "", CertIDDesc)
	cmdCDNUpdate.Example = `The following example updates the TTL for a CDN with the ID ` + "`" + `418b7972-fc67-41ea-ab4b-6f9477c4f7d8` + "`" + ` to 600 seconds: doctl compute cdn update 418b7972-fc67-41ea-ab4b-6f9477c4f7d8 --ttl 600`

	cmdCDNFlushCache := CmdBuilder(cmd, RunCDNFlushCache, "flush <cdn-id>", "Flush the cache of a CDN", `Flushes the cache of a Content Delivery Network (CDN), which:

- purges all copies of the files in the cache
- re-caches the files
- retrieves files from the origin server for any requests that hit the CDN endpoint until all the files are re-cached

This ensures that recently updated files on the origin server are immediately available via the CDN.

To purge specific files, you can use the `+"`"+`--files`+"`"+` flag and supply a path to the file in the Spaces bucket. The path may be for a single file or may contain a wildcard (`+"`"+`*`+"`"+`) to recursively purge all files under a directory. When only a wildcard is provided, or no path is provided, all cached files will be purged.
Examples:		
 doctl compute cdn flush <cdn-id>  --files /path/to/assets/*
 doctl compute cdn flush <cdn-id>  --files "/path/to/file.one, /path/to/file.two"
 doctl compute cdn flush <cdn-id>  --files /path/to/file.one --files /path/to/file.two
 doctl compute cdn flush <cdn-id>  --files * `, Writer,
		aliasOpt("fc"))
	AddStringSliceFlag(cmdCDNFlushCache, doctl.ArgCDNFiles, "", []string{"*"}, "cdn files")
	cmdCDNFlushCache.Example = `The following example flushes the cache of the ` + "`" + `/path/to/assets` + "`" + ` directory in a CDN: doctl compute cdn flush 418b7972-fc67-41ea-ab4b-6f9477c4f7d8 --files /path/to/assets/*`

	return cmd
}

// RunCDNList returns a list of CDNs.
func RunCDNList(c *CmdConfig) error {
	cdns, err := c.CDNs().List()
	if err != nil {
		return err
	}

	return c.Display(&displayers.CDN{CDNs: cdns})
}

// RunCDNGet returns an individual CDN.
func RunCDNGet(c *CmdConfig) error {
	if len(c.Args) == 0 {
		return doctl.NewMissingArgsErr(c.NS)
	}

	id := c.Args[0]
	item, err := c.CDNs().Get(id)
	if err != nil {
		return err
	}

	return c.Display(&displayers.CDN{CDNs: []do.CDN{*item}})
}

// RunCDNCreate creates a cdn.
func RunCDNCreate(c *CmdConfig) error {
	if len(c.Args) == 0 {
		return doctl.NewMissingArgsErr(c.NS)
	}

	origin := c.Args[0]

	ttl, err := c.Doit.GetInt(c.NS, doctl.ArgCDNTTL)
	if err != nil {
		return err
	}
	if ttl <= 0 {
		return errors.New("TTL cannot be zero or negative.")
	}

	domain, certID, err := getCDNDomainAndCertID(c)
	if err != nil {
		return err
	}

	createCDN := &godo.CDNCreateRequest{
		Origin:        origin,
		TTL:           uint32(ttl),
		CustomDomain:  domain,
		CertificateID: certID,
	}

	item, err := c.CDNs().Create(createCDN)
	if err != nil {
		return err
	}

	return c.Display(&displayers.CDN{CDNs: []do.CDN{*item}})
}

// RunCDNUpdate updates an individual cdn
func RunCDNUpdate(c *CmdConfig) error {
	if len(c.Args) == 0 {
		return doctl.NewMissingArgsErr(c.NS)
	}

	id := c.Args[0]

	cs := c.CDNs()

	var item *do.CDN
	if c.Doit.IsSet(doctl.ArgCDNTTL) {
		ttl, err := c.Doit.GetInt(c.NS, doctl.ArgCDNTTL)
		if err != nil {
			return err
		}
		if ttl <= 0 {
			return errors.New("TTL cannot be zero or negative.")
		}

		updateCDN := &godo.CDNUpdateTTLRequest{TTL: uint32(ttl)}

		item, err = cs.UpdateTTL(id, updateCDN)
		if err != nil {
			return err
		}
	}

	if c.Doit.IsSet(doctl.ArgCDNDomain) {
		domain, certID, err := getCDNDomainAndCertID(c)
		if err != nil {
			return err
		}

		updateCDN := &godo.CDNUpdateCustomDomainRequest{
			CustomDomain:  domain,
			CertificateID: certID,
		}

		item, err = cs.UpdateCustomDomain(id, updateCDN)
		if err != nil {
			return err
		}
	}

	if item != nil {
		return c.Display(&displayers.CDN{CDNs: []do.CDN{*item}})
	}

	return errors.New("Nothing to update.")
}

func getCDNDomainAndCertID(c *CmdConfig) (string, string, error) {
	var (
		domain, certID string
		err            error
	)
	domain, err = c.Doit.GetString(c.NS, doctl.ArgCDNDomain)
	if err != nil {
		return "", "", err
	}

	if domain != "" {
		certID, err = c.Doit.GetString(c.NS, doctl.ArgCDNCertificateID)
		if err != nil {
			return "", "", err
		}

		if certID == "" {
			return "", "", errors.New("A certificate ID is required to set a custom domain.")
		}
	}
	return domain, certID, err
}

// RunCDNDelete deletes a cdn.
func RunCDNDelete(c *CmdConfig) error {
	if len(c.Args) == 0 {
		return doctl.NewMissingArgsErr(c.NS)
	}

	force, err := c.Doit.GetBool(c.NS, doctl.ArgForce)
	if err != nil {
		return err
	}

	if force || AskForConfirmDelete("CDN", 1) == nil {
		id := c.Args[0]
		return c.CDNs().Delete(id)
	}

	return errOperationAborted
}

// RunCDNFlushCache flushes the cache of an individual cdn
func RunCDNFlushCache(c *CmdConfig) error {
	if len(c.Args) == 0 {
		return doctl.NewMissingArgsErr(c.NS)
	}

	id := c.Args[0]

	files, err := c.Doit.GetStringSlice(c.NS, doctl.ArgCDNFiles)
	if err != nil {
		return err
	}

	flushCDN := &godo.CDNFlushCacheRequest{Files: files}

	return c.CDNs().FlushCache(id, flushCDN)
}

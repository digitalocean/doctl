/*
Copyright 2026 The Doctl Authors All rights reserved.
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
	"os"
	"time"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/commands/displayers"
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"

	"github.com/spf13/cobra"
)

// Security creates the security command hierarchy.
func Security() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:   "security",
			Short: "Display commands to manage CSPM scans",
			Long: `The sub-commands of ` + "`" + `doctl security` + "`" + ` manage CSPM scans.

You can create scans, view existing scans, and list resources affected by scan findings.`,
			GroupID: manageResourcesGroup,
		},
	}

	cmd.AddCommand(SecurityScan())

	return cmd
}

func SecurityScan() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:     "scans",
			Aliases: []string{"scan"},
			Short:   "Display commands for managing CSPM scans",
			Long:    `The commands under ` + "`" + `doctl security scans` + "`" + ` are for managing CSPM scans.`,
		},
	}

	cmdScanCreate := CmdBuilder(cmd, RunCmdSecurityScanCreate, "create", "Create a CSPM scan", `Creates a new CSPM scan.`, Writer,
		aliasOpt("c"), displayerType(&displayers.SecurityScan{}))
	AddBoolFlag(cmdScanCreate, doctl.ArgCommandWait, "", false, "Boolean that specifies whether to wait for a scan to complete before returning control to the terminal")
	cmdScanCreate.Example = `The following example creates a CSPM scan for all droplets: doctl security scan create`

	cmdScanGet := CmdBuilder(cmd, RunCmdSecurityScanGet, "get <scan-uuid>", "Get a CSPM scan", `Retrieves a CSPM scan and its findings.`, Writer,
		aliasOpt("g"), displayerType(&displayers.SecurityScan{}))
	AddStringFlag(cmdScanGet, doctl.ArgSecurityScanFindingType, "", "", "Filter findings by type")
	AddStringFlag(cmdScanGet, doctl.ArgSecurityScanFindingSeverity, "", "", "Filter findings by severity")
	cmdScanGet.Example = `The following example retrieves a CSPM scan with findings filtered by severity: doctl security scan get 497dcba3-ecbf-4587-a2dd-5eb0665e6880 --severity critical`

	cmdScanLatest := CmdBuilder(cmd, RunCmdSecurityScanLatest, "latest", "Get the latest CSPM scan", `Retrieves the latest CSPM scan and its findings.`, Writer,
		displayerType(&displayers.SecurityScan{}))
	AddStringFlag(cmdScanLatest, doctl.ArgSecurityScanFindingType, "", "", "Filter findings by type")
	AddStringFlag(cmdScanLatest, doctl.ArgSecurityScanFindingSeverity, "", "", "Filter findings by severity")
	cmdScanLatest.Example = `The following example retrieves the latest CSPM scan with high severity findings: doctl security scans latest --severity high`

	cmdScanList := CmdBuilder(cmd, RunCmdSecurityScanList, "list", "List CSPM scans", `Retrieves a list of CSPM scans.`, Writer,
		aliasOpt("ls"), displayerType(&displayers.SecurityScans{}))
	cmdScanList.Example = `The following example lists all CSPM scans: doctl security scan list`

	cmdScanFindingAffectedResources := CmdBuilder(cmd, RunCmdSecurityFindingAffectedResources, "affected-resources <scan-uuid>", "List scan finding affected resources", `Retrieves the resources affected by the issue identified in a scan finding.`, Writer, displayerType(&displayers.SecurityAffectedResource{}))
	AddStringFlag(cmdScanFindingAffectedResources, doctl.ArgSecurityFindingUUID, "", "", "Finding UUID to show affected resources for", requiredOpt())
	cmdScanFindingAffectedResources.Example = `The following example lists affected resources for a finding: doctl security scans affected-resources --finding-uuid 50e14f43-dd4e-412f-864d-78943ea28d91 497dcba3-ecbf-4587-a2dd-5eb0665e6880 `

	return cmd
}

// RunCmdSecurityScanCreate creates a CSPM scan.
func RunCmdSecurityScanCreate(c *CmdConfig) error {
	scan, err := c.Security().CreateScan(&godo.CreateScanRequest{})
	if err != nil {
		return err
	}

	wait, err := c.Doit.GetBool(c.NS, doctl.ArgCommandWait)
	if err != nil {
		return err
	}

	if wait {
		security := c.Security()
		notice("Scan in progress, waiting for scan to complete")

		err := waitForScanComplete(security, scan.ID)
		if err != nil {
			return fmt.Errorf(
				"scan did not complete: %v",
				err,
			)
		}

		scan, _ = security.GetScan(scan.ID, nil)
	}

	notice("Scan completed")

	item := &displayers.SecurityScan{Scan: *scan}
	return c.Display(item)
}

// RunCmdSecurityScanGet retrieves a CSPM scan by UUID.
func RunCmdSecurityScanGet(c *CmdConfig) error {
	if len(c.Args) == 0 {
		return doctl.NewMissingArgsErr(c.NS)
	}

	opts, err := securityScanFindingOptions(c)
	if err != nil {
		return err
	}

	scan, err := c.Security().GetScan(c.Args[0], opts)
	if err != nil {
		return err
	}

	item := &displayers.SecurityScan{Scan: *scan}
	return c.Display(item)
}

// RunCmdSecurityScanLatest retrieves the latest CSPM scan.
func RunCmdSecurityScanLatest(c *CmdConfig) error {
	opts, err := securityScanFindingOptions(c)
	if err != nil {
		return err
	}

	scan, err := c.Security().GetLatestScan(opts)
	if err != nil {
		return err
	}

	item := &displayers.SecurityScan{Scan: *scan}
	return c.Display(item)
}

// RunCmdSecurityScanList lists CSPM scans.
func RunCmdSecurityScanList(c *CmdConfig) error {
	scans, err := c.Security().ListScans()
	if err != nil {
		return err
	}

	item := &displayers.SecurityScans{Scans: scans}
	return c.Display(item)
}

// RunCmdSecurityFindingAffectedResources lists affected resources for a finding.
func RunCmdSecurityFindingAffectedResources(c *CmdConfig) error {
	if len(c.Args) == 0 {
		return doctl.NewMissingArgsErr(c.NS)
	}
	scanUUID := c.Args[0]

	findingUUID, err := c.Doit.GetString(c.NS, doctl.ArgSecurityFindingUUID)
	if err != nil {
		return err
	}

	resources, err := c.Security().ListFindingAffectedResources(scanUUID, findingUUID)
	if err != nil {
		return err
	}

	item := &displayers.SecurityAffectedResource{AffectedResources: resources}
	return c.Display(item)
}

func securityScanFindingOptions(c *CmdConfig) (*godo.ScanFindingsOptions, error) {
	severity, err := c.Doit.GetString(c.NS, doctl.ArgSecurityScanFindingSeverity)
	if err != nil {
		return nil, err
	}

	findingType, err := c.Doit.GetString(c.NS, doctl.ArgSecurityScanFindingType)
	if err != nil {
		return nil, err
	}

	if severity == "" && findingType == "" {
		return nil, nil
	}

	return &godo.ScanFindingsOptions{
		Severity: severity,
		Type:     findingType,
	}, nil
}

func waitForScanComplete(scans do.SecurityService, id string) error {
	const maxAttempts = 16
	const wantStatus = "complete"
	const errStatus = "error"
	attempts := 0
	printNewLineSet := false

	for range maxAttempts {
		if attempts != 0 {
			fmt.Fprint(os.Stderr, ".")
			if !printNewLineSet {
				printNewLineSet = true
				defer fmt.Fprintln(os.Stderr)
			}
		}

		scan, err := scans.GetScan(id, nil)
		if err != nil {
			return err
		}

		if scan.Status == errStatus {
			return fmt.Errorf(
				"scan (%s) entered status `ERROR`",
				id,
			)
		}

		if scan.Status == wantStatus {
			return nil
		}

		attempts++
		time.Sleep(10 * time.Second)
	}

	return fmt.Errorf(
		"timeout waiting for scan (%s) to complete",
		id,
	)
}

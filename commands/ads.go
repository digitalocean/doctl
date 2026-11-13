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
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/commands/displayers"
	"github.com/digitalocean/godo"
	"github.com/spf13/cobra"
	"sigs.k8s.io/yaml"
)

// adsStdin is where `--data-file -` and `--manifest-file -` read from. Tests replace it.
var adsStdin io.Reader = os.Stdin

// ADS creates the ads command and its subcommands.
func ADS() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:   "ads",
			Short: "Display commands for managing Agent Data Services (ADS)",
			Long: `The subcommands of ` + "`" + `doctl ads` + "`" + ` manage Agent Data Services (ADS): workspaces, query tokens, workspace secrets, and the data flows that load data into a workspace.

Most commands need the workspace's region, passed with ` + "`" + `--region` + "`" + `. Run ` + "`" + `doctl ads plan list` + "`" + ` to see the regions and query engine sizes available to your team.`,
			GroupID: manageResourcesGroup,
		},
	}

	cmd.AddCommand(adsPlan())
	cmd.AddCommand(adsWorkspace())
	cmd.AddCommand(adsToken())
	cmd.AddCommand(adsSecret())
	cmd.AddCommand(adsDataFlow())
	cmd.AddCommand(adsMetrics())

	return cmd
}

func addADSRegionFlag(cmd *Command) {
	AddStringFlag(cmd, doctl.ArgRegionSlug, "", "", "Region of the workspace, for example `nyc3`", requiredOpt())
}

func addADSWorkspaceFlag(cmd *Command) {
	AddStringFlag(cmd, doctl.ArgADSWorkspace, "", "", "UUID of the workspace", requiredOpt())
}

// adsRegion reads the required --region flag.
func adsRegion(c *CmdConfig) (string, error) {
	region, err := c.Doit.GetString(c.NS, doctl.ArgRegionSlug)
	if err != nil {
		return "", err
	}
	if region = strings.TrimSpace(region); region == "" {
		return "", doctl.NewMissingArgsErr(fmt.Sprintf("%s.%s", c.NS, doctl.ArgRegionSlug))
	}
	return region, nil
}

// adsRegionWorkspace reads the required --region and --workspace flags.
func adsRegionWorkspace(c *CmdConfig) (string, string, error) {
	region, err := adsRegion(c)
	if err != nil {
		return "", "", err
	}
	workspace, err := c.Doit.GetString(c.NS, doctl.ArgADSWorkspace)
	if err != nil {
		return "", "", err
	}
	if workspace = strings.TrimSpace(workspace); workspace == "" {
		return "", "", doctl.NewMissingArgsErr(fmt.Sprintf("%s.%s", c.NS, doctl.ArgADSWorkspace))
	}
	return region, workspace, nil
}

func adsPlan() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:   "plan",
			Short: "Display commands for ADS plans",
			Long:  "The subcommands of `doctl ads plan` list the regions and query engine sizes you can choose when you create a workspace.",
		},
	}

	CmdBuilder(cmd, RunADSPlanList, "list", "List ADS plans",
		"Lists the regions where your team can create a workspace, and the query engine sizes offered in each.", Writer,
		aliasOpt("ls"),
		displayerType(&displayers.ADSPlan{}))

	return cmd
}

// RunADSPlanList lists ADS plans.
func RunADSPlanList(c *CmdConfig) error {
	resp, err := c.ADS().ListPlans()
	if err != nil {
		return err
	}
	return c.Display(&displayers.ADSPlan{Plans: resp.Plans})
}

func adsWorkspace() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:     "workspace",
			Aliases: []string{"workspaces", "ws"},
			Short:   "Display commands for ADS workspaces",
			Long:    "The subcommands of `doctl ads workspace` create, list, and delete workspaces, and retrieve the credentials for a workspace's query engine.",
		},
	}

	create := CmdBuilder(cmd, RunADSWorkspaceCreate, "create", "Create a workspace",
		"Creates a workspace in a region. Without `--name` or `--query-instance-size`, the server picks a name and the `small` query engine size.", Writer,
		aliasOpt("c"),
		displayerType(&displayers.ADSWorkspace{}))
	addADSRegionFlag(create)
	AddStringFlag(create, doctl.ArgADSName, "", "", "Name of the workspace")
	AddStringFlag(create, doctl.ArgADSQueryInstanceSize, "", "", "Query engine size: `small`, `medium`, or `large`")
	create.Example = "The following example creates a workspace in `nyc3`: doctl ads workspace create --region nyc3 --name analytics --query-instance-size medium"

	list := CmdBuilder(cmd, RunADSWorkspaceList, "list", "List workspaces",
		"Lists the workspaces in a region.", Writer,
		aliasOpt("ls"),
		displayerType(&displayers.ADSWorkspace{}))
	addADSRegionFlag(list)

	get := CmdBuilder(cmd, RunADSWorkspaceGet, "get <workspace-uuid>", "Get a workspace",
		"Retrieves a workspace.", Writer,
		aliasOpt("g"),
		displayerType(&displayers.ADSWorkspace{}))
	get.Args = cobra.ExactArgs(1)
	addADSRegionFlag(get)

	deleteCmd := CmdBuilder(cmd, RunADSWorkspaceDelete, "delete <workspace-uuid>", "Delete a workspace",
		"Deletes a workspace, including its secrets and data flows. This cannot be undone.", Writer,
		aliasOpt("d", "rm"))
	deleteCmd.Args = cobra.ExactArgs(1)
	addADSRegionFlag(deleteCmd)
	AddBoolFlag(deleteCmd, doctl.ArgForce, doctl.ArgShortForce, false, "Delete the workspace without a confirmation prompt")

	creds := CmdBuilder(cmd, RunADSWorkspaceQueryCredentials, "query-credentials <workspace-uuid>", "Get the query credentials of a workspace",
		"Retrieves the user and password for a workspace's query engine.\n\nThe output includes the password in plain text. Treat it as a secret: do not paste it into tickets, chat, or logs.", Writer,
		displayerType(&displayers.ADSQueryCredentials{}))
	creds.Args = cobra.ExactArgs(1)
	addADSRegionFlag(creds)

	return cmd
}

// RunADSWorkspaceCreate creates a workspace.
func RunADSWorkspaceCreate(c *CmdConfig) error {
	region, err := adsRegion(c)
	if err != nil {
		return err
	}
	name, err := c.Doit.GetString(c.NS, doctl.ArgADSName)
	if err != nil {
		return err
	}
	size, err := c.Doit.GetString(c.NS, doctl.ArgADSQueryInstanceSize)
	if err != nil {
		return err
	}

	ws, err := c.ADS().CreateWorkspace(region, &godo.ADSWorkspaceCreateRequest{
		Name:              name,
		QueryInstanceSize: size,
	})
	if err != nil {
		return err
	}
	return c.Display(&displayers.ADSWorkspace{Workspaces: []godo.ADSWorkspace{*ws}})
}

// RunADSWorkspaceList lists workspaces.
func RunADSWorkspaceList(c *CmdConfig) error {
	region, err := adsRegion(c)
	if err != nil {
		return err
	}
	resp, err := c.ADS().ListWorkspaces(region)
	if err != nil {
		return err
	}
	return c.Display(&displayers.ADSWorkspace{Workspaces: resp.Workspaces})
}

// RunADSWorkspaceGet retrieves a workspace.
func RunADSWorkspaceGet(c *CmdConfig) error {
	if len(c.Args) != 1 {
		return doctl.NewMissingArgsErr(c.NS)
	}
	region, err := adsRegion(c)
	if err != nil {
		return err
	}
	ws, err := c.ADS().GetWorkspace(region, c.Args[0])
	if err != nil {
		return err
	}
	return c.Display(&displayers.ADSWorkspace{Workspaces: []godo.ADSWorkspace{*ws}})
}

// RunADSWorkspaceDelete deletes a workspace.
func RunADSWorkspaceDelete(c *CmdConfig) error {
	if len(c.Args) != 1 {
		return doctl.NewMissingArgsErr(c.NS)
	}
	region, err := adsRegion(c)
	if err != nil {
		return err
	}
	force, err := c.Doit.GetBool(c.NS, doctl.ArgForce)
	if err != nil {
		return err
	}
	if err := confirmDelete(force, "ADS workspace", 1); err != nil {
		return err
	}
	return c.ADS().DeleteWorkspace(region, c.Args[0])
}

// RunADSWorkspaceQueryCredentials retrieves the query user and password of a workspace.
func RunADSWorkspaceQueryCredentials(c *CmdConfig) error {
	if len(c.Args) != 1 {
		return doctl.NewMissingArgsErr(c.NS)
	}
	region, err := adsRegion(c)
	if err != nil {
		return err
	}
	creds, err := c.ADS().GetWorkspaceQueryCredentials(region, c.Args[0])
	if err != nil {
		return err
	}
	warn("The output below contains the workspace's query password. Treat it as a secret.")
	return c.Display(&displayers.ADSQueryCredentials{Credentials: *creds})
}

func adsToken() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:     "token",
			Aliases: []string{"tokens"},
			Short:   "Display commands for ADS query tokens",
			Long:    "The subcommands of `doctl ads token` create and list tokens for querying workspaces.",
		},
	}

	create := CmdBuilder(cmd, RunADSTokenCreate, "create", "Create a query token",
		"Creates a token in a region. The access token is shown only once, in the output of this command. It cannot be retrieved again, so save it somewhere safe.", Writer,
		aliasOpt("c"),
		displayerType(&displayers.ADSTokenCreated{}))
	addADSRegionFlag(create)
	AddStringFlag(create, doctl.ArgADSName, "", "", "Name of the token")
	AddStringFlag(create, doctl.ArgADSTokenPurpose, "", "", "Purpose of the token. Possible value: `query`")
	AddStringSliceFlag(create, doctl.ArgADSTokenWorkspaceUUIDs, "", []string{}, "Comma-separated UUIDs of the workspaces the token can access")
	AddIntFlag(create, doctl.ArgADSTokenExpiresIn, "", 0, "Lifetime of the token in seconds. The server default applies when unset")
	AddStringSliceFlag(create, doctl.ArgADSTokenPermissions, "", []string{}, "Comma-separated permissions granted to the token")
	create.Example = "The following example creates a query token for one workspace: doctl ads token create --region nyc3 --name reporting --purpose query --workspace-uuids f81d4fae-7dec-11d0-a765-00a0c91e6bf6"

	list := CmdBuilder(cmd, RunADSTokenList, "list", "List query tokens",
		"Lists the tokens in a region. Access tokens are never included in the list.", Writer,
		aliasOpt("ls"),
		displayerType(&displayers.ADSToken{}))
	addADSRegionFlag(list)
	AddStringFlag(list, doctl.ArgADSTokenPurpose, "", "", "Only list tokens with this purpose. Possible value: `query`")
	AddStringFlag(list, doctl.ArgADSTokenWorkspaceUUID, "", "", "Only list tokens that can access this workspace")
	AddIntFlag(list, doctl.ArgADSPage, "", 0, "Page of results to return")
	AddIntFlag(list, doctl.ArgADSPerPage, "", 0, "Number of results per page")

	return cmd
}

func adsTokenPurposeFlag(c *CmdConfig) (godo.ADSTokenPurpose, error) {
	purpose, err := c.Doit.GetString(c.NS, doctl.ArgADSTokenPurpose)
	if err != nil {
		return "", err
	}
	switch strings.ToLower(strings.TrimSpace(purpose)) {
	case "":
		return "", nil
	case "query", strings.ToLower(string(godo.ADSTokenPurposeQuery)):
		return godo.ADSTokenPurposeQuery, nil
	default:
		return "", fmt.Errorf("unsupported --%s %q: the only supported purpose is query", doctl.ArgADSTokenPurpose, purpose)
	}
}

// RunADSTokenCreate creates a query token.
func RunADSTokenCreate(c *CmdConfig) error {
	region, err := adsRegion(c)
	if err != nil {
		return err
	}
	name, err := c.Doit.GetString(c.NS, doctl.ArgADSName)
	if err != nil {
		return err
	}
	purpose, err := adsTokenPurposeFlag(c)
	if err != nil {
		return err
	}
	workspaces, err := c.Doit.GetStringSlice(c.NS, doctl.ArgADSTokenWorkspaceUUIDs)
	if err != nil {
		return err
	}
	expiresIn, err := c.Doit.GetInt(c.NS, doctl.ArgADSTokenExpiresIn)
	if err != nil {
		return err
	}
	if expiresIn < 0 {
		return fmt.Errorf("--%s must not be negative", doctl.ArgADSTokenExpiresIn)
	}
	permissions, err := c.Doit.GetStringSlice(c.NS, doctl.ArgADSTokenPermissions)
	if err != nil {
		return err
	}

	token, err := c.ADS().CreateToken(&godo.ADSTokenCreateRequest{
		Region:           region,
		Purpose:          purpose,
		WorkspaceUUIDs:   workspaces,
		Name:             name,
		ExpiresInSeconds: int64(expiresIn),
		Permissions:      permissions,
	})
	if err != nil {
		return err
	}
	warn("Save this access token now. It is shown only once and cannot be retrieved again.")
	return c.Display(&displayers.ADSTokenCreated{Token: *token})
}

// RunADSTokenList lists query tokens.
func RunADSTokenList(c *CmdConfig) error {
	region, err := adsRegion(c)
	if err != nil {
		return err
	}
	purpose, err := adsTokenPurposeFlag(c)
	if err != nil {
		return err
	}
	workspace, err := c.Doit.GetString(c.NS, doctl.ArgADSTokenWorkspaceUUID)
	if err != nil {
		return err
	}
	page, err := c.Doit.GetInt(c.NS, doctl.ArgADSPage)
	if err != nil {
		return err
	}
	perPage, err := c.Doit.GetInt(c.NS, doctl.ArgADSPerPage)
	if err != nil {
		return err
	}
	if page < 0 || perPage < 0 {
		return fmt.Errorf("--%s and --%s must not be negative", doctl.ArgADSPage, doctl.ArgADSPerPage)
	}

	resp, err := c.ADS().ListTokens(&godo.ADSTokenListRequest{
		Region:        region,
		Purpose:       purpose,
		Page:          int64(page),
		PerPage:       int64(perPage),
		WorkspaceUUID: workspace,
	})
	if err != nil {
		return err
	}
	return c.Display(&displayers.ADSToken{Tokens: resp.Tokens})
}

func adsSecret() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:     "secret",
			Aliases: []string{"secrets"},
			Short:   "Display commands for ADS workspace secrets",
			Long: `The subcommands of ` + "`" + `doctl ads secret` + "`" + ` manage the secrets stored in a workspace. Data flows reference these secrets by key.

A secret is a set of string properties. doctl never prints property values: ` + "`" + `get` + "`" + ` shows the secret's metadata and property names only.`,
		},
	}

	put := CmdBuilder(cmd, RunADSSecretPut, "put <secret-key>", "Create or replace a workspace secret",
		"Writes a workspace secret, replacing any existing version with the same key.\n\nPass properties with `--data key=value`, repeated or comma-separated, or read them from a flat JSON or YAML object with `--data-file`. Use `--data-file -` to read from stdin, which keeps values out of your shell history. The values are not printed back.", Writer,
		displayerType(&displayers.ADSSecretWrite{}))
	put.Args = cobra.ExactArgs(1)
	addADSRegionFlag(put)
	addADSWorkspaceFlag(put)
	AddStringMapStringFlag(put, doctl.ArgADSSecretData, "", map[string]string{}, "Secret property as `key=value`. Repeat the flag for more properties")
	AddStringFlag(put, doctl.ArgADSSecretDataFile, "", "", "Path to a JSON or YAML object of string properties. Set to - to read from stdin")
	put.Example = "The following example stores database credentials read from stdin: doctl ads secret put warehouse-db --region nyc3 --workspace f81d4fae-7dec-11d0-a765-00a0c91e6bf6 --data-file - < creds.json"

	get := CmdBuilder(cmd, RunADSSecretGet, "get <secret-key>", "Get a workspace secret's metadata",
		"Retrieves a workspace secret's metadata and property names. Property values are never returned.", Writer,
		aliasOpt("g"),
		displayerType(&displayers.ADSSecret{}))
	get.Args = cobra.ExactArgs(1)
	addADSRegionFlag(get)
	addADSWorkspaceFlag(get)

	list := CmdBuilder(cmd, RunADSSecretList, "list", "List workspace secrets",
		"Lists the secret keys stored in a workspace.", Writer,
		aliasOpt("ls"),
		displayerType(&displayers.ADSSecretKey{}))
	addADSRegionFlag(list)
	addADSWorkspaceFlag(list)

	deleteCmd := CmdBuilder(cmd, RunADSSecretDelete, "delete <secret-key>", "Delete a workspace secret",
		"Permanently deletes a workspace secret. This cannot be undone.", Writer,
		aliasOpt("d", "rm"))
	deleteCmd.Args = cobra.ExactArgs(1)
	addADSRegionFlag(deleteCmd)
	addADSWorkspaceFlag(deleteCmd)
	AddBoolFlag(deleteCmd, doctl.ArgForce, doctl.ArgShortForce, false, "Delete the secret without a confirmation prompt")

	return cmd
}

var errADSSecretDataFile = fmt.Errorf("--%s must be a JSON or YAML object whose values are all strings", doctl.ArgADSSecretDataFile)

// readADSFile reads path, or adsStdin when path is "-".
func readADSFile(path string) ([]byte, error) {
	if path == "-" {
		data, err := io.ReadAll(adsStdin)
		if err != nil {
			return nil, fmt.Errorf("reading stdin: %w", err)
		}
		return data, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}
	return data, nil
}

// adsSecretData reads the secret properties from --data or --data-file.
func adsSecretData(c *CmdConfig) (map[string]string, error) {
	data, err := c.Doit.GetStringMapString(c.NS, doctl.ArgADSSecretData)
	if err != nil {
		return nil, err
	}
	path, err := c.Doit.GetString(c.NS, doctl.ArgADSSecretDataFile)
	if err != nil {
		return nil, err
	}

	if path != "" {
		if len(data) > 0 {
			return nil, fmt.Errorf("--%s cannot be combined with --%s", doctl.ArgADSSecretDataFile, doctl.ArgADSSecretData)
		}
		raw, err := readADSFile(path)
		if err != nil {
			return nil, err
		}
		// Parse errors can quote the input, which here is secret, so they
		// are replaced rather than wrapped.
		jsonData, err := yaml.YAMLToJSON(raw)
		if err != nil {
			return nil, errADSSecretDataFile
		}
		data = map[string]string{}
		if err := json.Unmarshal(jsonData, &data); err != nil {
			return nil, errADSSecretDataFile
		}
	}

	if len(data) == 0 {
		return nil, fmt.Errorf("at least one property is required: set --%s or --%s", doctl.ArgADSSecretData, doctl.ArgADSSecretDataFile)
	}
	for key := range data {
		if strings.TrimSpace(key) == "" {
			return nil, errors.New("secret property names must not be empty")
		}
	}
	return data, nil
}

// RunADSSecretPut writes a workspace secret.
func RunADSSecretPut(c *CmdConfig) error {
	if len(c.Args) != 1 {
		return doctl.NewMissingArgsErr(c.NS)
	}
	region, workspace, err := adsRegionWorkspace(c)
	if err != nil {
		return err
	}
	data, err := adsSecretData(c)
	if err != nil {
		return err
	}

	resp, err := c.ADS().PutWorkspaceSecret(&godo.ADSPutWorkspaceSecretRequest{
		Region:    region,
		UUID:      workspace,
		SecretKey: c.Args[0],
		Data:      data,
	})
	if err != nil {
		return err
	}
	return c.Display(&displayers.ADSSecretWrite{Key: c.Args[0], Metadata: resp.Data})
}

// RunADSSecretGet retrieves a workspace secret's metadata and property names.
func RunADSSecretGet(c *CmdConfig) error {
	if len(c.Args) != 1 {
		return doctl.NewMissingArgsErr(c.NS)
	}
	region, workspace, err := adsRegionWorkspace(c)
	if err != nil {
		return err
	}
	resp, err := c.ADS().GetWorkspaceSecret(region, workspace, c.Args[0])
	if err != nil {
		return err
	}

	item := &displayers.ADSSecret{Key: c.Args[0]}
	if resp.Data != nil {
		item.Keys = resp.Data.Keys
		item.Metadata = resp.Data.Metadata
		// The API returns property names in Keys. Older responses carried
		// them as the keys of Data; use those names but never the values.
		if len(item.Keys) == 0 {
			for name := range resp.Data.Data {
				item.Keys = append(item.Keys, name)
			}
			sort.Strings(item.Keys)
		}
	}
	return c.Display(item)
}

// RunADSSecretList lists the secret keys of a workspace.
func RunADSSecretList(c *CmdConfig) error {
	region, workspace, err := adsRegionWorkspace(c)
	if err != nil {
		return err
	}
	resp, err := c.ADS().ListWorkspaceSecrets(region, workspace)
	if err != nil {
		return err
	}
	return c.Display(&displayers.ADSSecretKey{Keys: resp.SecretKeys})
}

// RunADSSecretDelete deletes a workspace secret.
func RunADSSecretDelete(c *CmdConfig) error {
	if len(c.Args) != 1 {
		return doctl.NewMissingArgsErr(c.NS)
	}
	region, workspace, err := adsRegionWorkspace(c)
	if err != nil {
		return err
	}
	force, err := c.Doit.GetBool(c.NS, doctl.ArgForce)
	if err != nil {
		return err
	}
	if err := confirmDelete(force, "ADS workspace secret", 1); err != nil {
		return err
	}
	return c.ADS().DeleteWorkspaceSecret(region, workspace, c.Args[0])
}

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
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/commands/displayers"
	"github.com/digitalocean/godo"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

// secretValuesReader reads interactive secret value input. It can be replaced in tests.
var secretValuesReader = bufio.NewReader(os.Stdin)

// secretRegionReader reads interactive region input. It can be replaced in tests.
var secretRegionReader = bufio.NewReader(os.Stdin)

// secretNameReader reads interactive secret name input. It can be replaced in tests.
var secretNameReader = bufio.NewReader(os.Stdin)

// secretVersionReader reads interactive version input. It can be replaced in tests.
var secretVersionReader = bufio.NewReader(os.Stdin)

var exampleSecretRegions = []string{"nyc3", "sfo3", "ams3", "fra1", "sgp1", "lon1"}

const secretRegionFlagDesc = "Region where the secret is stored. If omitted, you are prompted when running interactively."

const secretVersionFlagDesc = "Current version of the secret to update. If omitted, the current version is used when it can be determined from the API."

// Secrets creates the secrets command hierarchy.
func Secrets() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:   "secrets",
			Short: "Display commands to manage Secrets Manager",
			Long: `The subcommands of ` + "`" + `doctl secrets` + "`" + ` manage Secrets Manager secret containers.

Each secret is a named container in a region that holds one or more key-value pairs.`,
			GroupID: manageResourcesGroup,
		},
	}

	cmdCreate := CmdBuilder(cmd, RunCmdSecretsCreate, "create <name>", "Create a secret", `Creates a secret container in the specified region and stores key-value pairs inside it.

If no `+"`"+`--value`+"`"+` flags are provided, key-value pairs are read interactively from stdin.`, Writer,
		aliasOpt("c"), displayerType(&displayers.SecretWriteResult{}))
	AddStringFlag(cmdCreate, doctl.ArgRegionSlug, "", "", secretRegionFlagDesc)
	AddStringSliceFlag(cmdCreate, doctl.ArgSecretValue, "", nil,
		"Key-value pair in key=value format (repeatable). If omitted, values are read interactively.")
	cmdCreate.Example = `The following example creates a secret with key-value pairs: doctl secrets create prod-db-creds --region nyc3 --value password=super-secret --value api_key=abc123`

	cmdGet := CmdBuilder(cmd, RunCmdSecretsGet, "get <name>", "Get a secret", `Retrieves a secret container and its key-value pairs.`, Writer,
		aliasOpt("g"), displayerType(&displayers.Secret{}))
	AddStringFlag(cmdGet, doctl.ArgRegionSlug, "", "", secretRegionFlagDesc)
	cmdGet.Example = `The following example retrieves a secret: doctl secrets get prod-db-creds --region nyc3`

	cmdList := CmdBuilder(cmd, RunCmdSecretsList, "list", "List secrets", `Retrieves a list of secret containers across all regions. Values are not included.`, Writer,
		aliasOpt("ls"), displayerType(&displayers.Secrets{}))
	cmdList.Example = `The following example lists all secrets: doctl secrets list`

	cmdListVersions := CmdBuilder(cmd, RunCmdSecretsListVersions, "list-versions <name>", "List secret versions", `Retrieves version history for a secret container.`, Writer,
		displayerType(&displayers.SecretVersions{}))
	AddStringFlag(cmdListVersions, doctl.ArgRegionSlug, "", "", secretRegionFlagDesc)
	cmdListVersions.Example = `The following example lists versions for a secret: doctl secrets list-versions prod-db-creds --region nyc3`

	cmdUpdate := CmdBuilder(cmd, RunCmdSecretsUpdate, "update <name>", "Update a secret", `Updates a secret container by replacing all key-value pairs with a new version.

This replaces the entire contents of the secret. Include every key you want to keep.`, Writer,
		displayerType(&displayers.SecretWriteResult{}))
	AddStringFlag(cmdUpdate, doctl.ArgRegionSlug, "", "", secretRegionFlagDesc)
	AddIntFlag(cmdUpdate, doctl.ArgSecretVersion, "", 0, secretVersionFlagDesc)
	AddStringSliceFlag(cmdUpdate, doctl.ArgSecretValue, "", nil,
		"Key-value pair in key=value format (repeatable). If omitted, values are read interactively.")
	cmdUpdate.Example = `The following example updates a secret: doctl secrets update prod-db-creds --region nyc3 --version 1 --value password=new-secret --value api_key=abc123`

	cmdDelete := CmdBuilder(cmd, RunCmdSecretsDelete, "delete <name>", "Delete a secret", `Schedules a secret container for soft deletion.`, Writer, aliasOpt("d", "rm"))
	AddStringFlag(cmdDelete, doctl.ArgRegionSlug, "", "", secretRegionFlagDesc)
	cmdDelete.Example = `The following example deletes a secret: doctl secrets delete prod-db-creds --region nyc3`

	cmdRestore := CmdBuilder(cmd, RunCmdSecretsRestore, "restore <name>", "Restore a secret", `Restores a secret container that was scheduled for deletion.`, Writer)
	AddStringFlag(cmdRestore, doctl.ArgRegionSlug, "", "", secretRegionFlagDesc)
	cmdRestore.Example = `The following example restores a secret: doctl secrets restore prod-db-creds --region nyc3`

	return cmd
}

// RunCmdSecretsCreate creates a secret container.
func RunCmdSecretsCreate(c *CmdConfig) error {
	name, err := resolveSecretName(c)
	if err != nil {
		return err
	}

	region, err := resolveSecretRegion(c)
	if err != nil {
		return err
	}

	values, err := collectSecretValues(c)
	if err != nil {
		return err
	}

	result, err := c.Secrets().Create(&godo.SecretCreateRequest{
		Name:   name,
		Region: region,
		Values: values,
	})
	if err != nil {
		return err
	}

	return c.Display(&displayers.SecretWriteResult{Result: *result})
}

// RunCmdSecretsGet retrieves a secret container.
func RunCmdSecretsGet(c *CmdConfig) error {
	name, err := resolveSecretName(c)
	if err != nil {
		return err
	}

	region, err := resolveSecretRegion(c)
	if err != nil {
		return err
	}

	secret, err := c.Secrets().Get(name, region)
	if err != nil {
		return err
	}

	return c.Display(&displayers.Secret{Secret: *secret})
}

// RunCmdSecretsList lists secret containers.
func RunCmdSecretsList(c *CmdConfig) error {
	list, err := c.Secrets().List()
	if err != nil {
		return err
	}

	if len(list.UnavailableRegions) > 0 {
		notice("Some regions were unavailable: %s", strings.Join(list.UnavailableRegions, ", "))
	}

	return c.Display(&displayers.Secrets{Secrets: list.Secrets})
}

// RunCmdSecretsListVersions lists versions for a secret container.
func RunCmdSecretsListVersions(c *CmdConfig) error {
	name, err := resolveSecretName(c)
	if err != nil {
		return err
	}

	region, err := resolveSecretRegion(c)
	if err != nil {
		return err
	}

	versions, err := c.Secrets().ListVersions(name, region)
	if err != nil {
		return err
	}

	return c.Display(&displayers.SecretVersions{Versions: versions})
}

// RunCmdSecretsUpdate updates a secret container.
func RunCmdSecretsUpdate(c *CmdConfig) error {
	name, err := resolveSecretName(c)
	if err != nil {
		return err
	}

	region, err := resolveSecretRegion(c)
	if err != nil {
		return err
	}

	version, err := resolveSecretVersion(c, name, region)
	if err != nil {
		return err
	}

	values, err := collectSecretValuesForUpdate(c, name, region)
	if err != nil {
		return err
	}

	result, err := c.Secrets().Update(name, &godo.SecretUpdateRequest{
		Region:  region,
		Version: version,
		Values:  values,
	})
	if err != nil {
		return err
	}

	return c.Display(&displayers.SecretWriteResult{Result: *result})
}

// RunCmdSecretsDelete schedules a secret container for deletion.
func RunCmdSecretsDelete(c *CmdConfig) error {
	name, err := resolveSecretName(c)
	if err != nil {
		return err
	}

	region, err := resolveSecretRegion(c)
	if err != nil {
		return err
	}

	return c.Secrets().Delete(name, region)
}

// RunCmdSecretsRestore restores a secret container scheduled for deletion.
func RunCmdSecretsRestore(c *CmdConfig) error {
	name, err := resolveSecretName(c)
	if err != nil {
		return err
	}

	region, err := resolveSecretRegion(c)
	if err != nil {
		return err
	}

	return c.Secrets().Restore(name, region)
}

func collectSecretValues(c *CmdConfig) (map[string]string, error) {
	flags, err := c.Doit.GetStringSlice(c.NS, doctl.ArgSecretValue)
	if err != nil {
		return nil, err
	}
	if len(flags) > 0 {
		return parseSecretValues(flags)
	}
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return nil, fmt.Errorf("must specify at least one --%s when not running interactively", doctl.ArgSecretValue)
	}

	name := ""
	if len(c.Args) > 0 {
		name = c.Args[0]
	}
	region, _ := c.Doit.GetString(c.NS, doctl.ArgRegionSlug)

	return readSecretValuesInteractive(os.Stderr, secretValuesReader, secretValuePromptCreate(name, region))
}

func collectSecretValuesForUpdate(c *CmdConfig, name, region string) (map[string]string, error) {
	flags, err := c.Doit.GetStringSlice(c.NS, doctl.ArgSecretValue)
	if err != nil {
		return nil, err
	}
	if len(flags) > 0 {
		return parseSecretValues(flags)
	}
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return nil, fmt.Errorf("must specify at least one --%s when not running interactively", doctl.ArgSecretValue)
	}

	return readSecretValuesInteractive(os.Stderr, secretValuesReader, secretValuePromptUpdate(name, region))
}

func resolveSecretName(c *CmdConfig) (string, error) {
	if len(c.Args) > 0 && c.Args[0] != "" {
		return c.Args[0], nil
	}
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return "", doctl.NewMissingArgsErr(c.NS)
	}

	name, err := readSecretNameInteractive(os.Stderr, secretNameReader)
	if err != nil {
		return "", err
	}

	c.Args = []string{name}

	return name, nil
}

func readSecretNameInteractive(out io.Writer, in *bufio.Reader) (string, error) {
	fmt.Fprint(out, "Enter the secret name: ")

	line, err := in.ReadString('\n')
	if err != nil {
		return "", err
	}

	name := strings.TrimSpace(line)
	if name == "" {
		return "", fmt.Errorf("secret name is required")
	}

	return name, nil
}

func resolveSecretVersion(c *CmdConfig, name, region string) (int, error) {
	versionPtr, err := c.Doit.GetIntPtr(c.NS, doctl.ArgSecretVersion)
	if err != nil {
		return 0, err
	}
	if versionPtr != nil {
		if *versionPtr <= 0 {
			return 0, fmt.Errorf("version must be a positive integer")
		}
		return *versionPtr, nil
	}

	secret, err := c.Secrets().Get(name, region)
	if err == nil && secret.Version > 0 {
		notice("Using current version %d of %q in %s", secret.Version, name, region)
		c.Doit.Set(c.NS, doctl.ArgSecretVersion, secret.Version)
		return secret.Version, nil
	}

	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return 0, fmt.Errorf("must specify --%s when the current version cannot be determined", doctl.ArgSecretVersion)
	}

	version, err := readSecretVersionInteractive(os.Stderr, secretVersionReader, name, region)
	if err != nil {
		return 0, err
	}

	c.Doit.Set(c.NS, doctl.ArgSecretVersion, version)

	return version, nil
}

func readSecretVersionInteractive(out io.Writer, in *bufio.Reader, name, region string) (int, error) {
	fmt.Fprintf(out, "Version is required. Enter the current version of %q in %s to update.\nVersion: ", name, region)

	line, err := in.ReadString('\n')
	if err != nil {
		return 0, err
	}

	line = strings.TrimSpace(line)
	if line == "" {
		return 0, fmt.Errorf("version is required")
	}

	version, err := strconv.Atoi(line)
	if err != nil || version <= 0 {
		return 0, fmt.Errorf("version must be a positive integer")
	}

	return version, nil
}

func resolveSecretRegion(c *CmdConfig) (string, error) {
	region, err := c.Doit.GetString(c.NS, doctl.ArgRegionSlug)
	if err != nil {
		return "", err
	}
	if region != "" {
		return region, nil
	}
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return "", fmt.Errorf("must specify --%s when not running interactively", doctl.ArgRegionSlug)
	}

	region, err = readSecretRegionInteractive(os.Stderr, secretRegionReader)
	if err != nil {
		return "", err
	}

	c.Doit.Set(c.NS, doctl.ArgRegionSlug, region)

	return region, nil
}

func readSecretRegionInteractive(out io.Writer, in *bufio.Reader) (string, error) {
	fmt.Fprintf(out, "Region is required. Enter the region slug where the secret is stored.\nExamples: %s\nRegion: ", strings.Join(exampleSecretRegions, ", "))

	line, err := in.ReadString('\n')
	if err != nil {
		return "", err
	}

	region := strings.TrimSpace(line)
	if region == "" {
		return "", fmt.Errorf("region is required")
	}

	return region, nil
}

func secretValuePromptCreate(name, region string) string {
	if name != "" && region != "" {
		return fmt.Sprintf(`Creating secret %q in %s.
Enter key-value pairs to store in this secret (key=value format, one per line).
Press Enter on an empty line when done.`, name, region)
	}

	return `Enter key-value pairs to store in this secret (key=value format, one per line).
Press Enter on an empty line when done.`
}

func secretValuePromptUpdate(name, region string) string {
	return fmt.Sprintf(`Updating secret %q in %s.
This replaces all key-value pairs in the secret. Enter the full set you want to keep.
Enter key-value pairs (key=value format, one per line). Press Enter on an empty line when done.`, name, region)
}

func parseSecretValues(lines []string) (map[string]string, error) {
	if len(lines) == 0 {
		return nil, fmt.Errorf("at least one key-value pair is required")
	}

	values := make(map[string]string, len(lines))
	for _, line := range lines {
		key, value, err := parseSecretValueLine(line)
		if err != nil {
			return nil, err
		}
		if _, exists := values[key]; exists {
			return nil, fmt.Errorf("duplicate key %q", key)
		}
		values[key] = value
	}

	return values, nil
}

func parseSecretValueLine(line string) (string, string, error) {
	parts := strings.SplitN(strings.TrimSpace(line), "=", 2)
	if len(parts) != 2 || parts[0] == "" {
		return "", "", fmt.Errorf("invalid key-value pair %q, expected key=value", line)
	}

	return parts[0], parts[1], nil
}

func readSecretValuesInteractive(out io.Writer, in *bufio.Reader, prompt string) (map[string]string, error) {
	fmt.Fprintln(out, prompt)

	values := make(map[string]string)
	for {
		line, err := in.ReadString('\n')
		if err != nil {
			return nil, err
		}

		line = strings.TrimSpace(line)
		if line == "" {
			break
		}

		key, value, err := parseSecretValueLine(line)
		if err != nil {
			return nil, err
		}
		if _, exists := values[key]; exists {
			return nil, fmt.Errorf("duplicate key %q", key)
		}
		values[key] = value
	}

	if len(values) == 0 {
		return nil, fmt.Errorf("at least one key-value pair is required")
	}

	return values, nil
}

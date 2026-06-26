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
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/commands/charm/input"
	"github.com/digitalocean/doctl/commands/charm/list"
	"github.com/digitalocean/doctl/commands/displayers"
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

// promptSecretKeyFunc prompts for a secret key name. It can be replaced in tests.
var promptSecretKeyFunc = defaultPromptSecretKey

// promptSecretValueFunc prompts for a secret value. It can be replaced in tests.
var promptSecretValueFunc = defaultPromptSecretValue

var exampleSecretRegions = []string{"nyc3", "sfo3", "ams3", "fra1", "sgp1", "lon1"}

const (
	secretRegionFlagDesc  = "Region where the secret is stored. If omitted, you are prompted when running with --interactive."
	secretVersionFlagDesc = "Current version of the secret to update. If omitted, the current version is used when it can be determined from the API."
	secretValueFlagDesc   = "Key-value pair in key=value format (repeatable). Values may be read from a file with key=@path or from stdin with key=-. If omitted, keys and masked values are read with --interactive."
	secretMaskedValue     = "********"
)

type secretRegionListItem struct {
	slug string
}

func (i secretRegionListItem) Title() string       { return i.slug }
func (i secretRegionListItem) Description() string { return "" }
func (i secretRegionListItem) FilterValue() string { return i.slug }

func secretsPromptsEnabled() bool {
	return Interactive && term.IsTerminal(int(os.Stdin.Fd()))
}

func secretNotice(msg string, args ...any) {
	fmt.Fprintf(os.Stderr, "%s: %s\n", colorNotice, fmt.Sprintf(msg, args...))
}

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

If no `+"`"+`--value`+"`"+` or `+"`"+`--from-env-file`+"`"+` flags are provided, key-value pairs are read interactively with --interactive. You are prompted for each key, then each value is masked.`, Writer,
		aliasOpt("c"), displayerType(&displayers.SecretWriteResult{}))
	AddStringFlag(cmdCreate, doctl.ArgRegionSlug, "", "", secretRegionFlagDesc)
	AddStringSliceFlag(cmdCreate, doctl.ArgSecretValue, "", nil, secretValueFlagDesc)
	AddStringFlag(cmdCreate, doctl.ArgSecretFromEnvFile, "", "", "Path to an env file containing key-value pairs to store in the secret.")
	cmdCreate.Example = `The following example creates a secret with key-value pairs: doctl secrets create prod-db-creds --region nyc3 --value password=@./pw.txt --value api_key=-`

	cmdGet := CmdBuilder(cmd, RunCmdSecretsGet, "get <name>", "Get a secret", `Retrieves a secret container and its key-value pairs.

Secret values are masked by default. Use --show to reveal them, or --key with --raw to print a single value for scripting.`, Writer,
		aliasOpt("g"), displayerType(&displayers.Secret{}))
	AddStringFlag(cmdGet, doctl.ArgRegionSlug, "", "", secretRegionFlagDesc)
	AddStringFlag(cmdGet, doctl.ArgKey, "", "", "Return only the value for this key.")
	AddBoolFlag(cmdGet, doctl.ArgSecretShow, "", false, "Reveal secret values instead of masking them.")
	AddBoolFlag(cmdGet, doctl.ArgSecretRaw, "", false, "Write the value for --key to stdout with no formatting.")
	cmdGet.Example = `The following example retrieves a secret: doctl secrets get prod-db-creds --region nyc3 --key password --raw`

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
	AddStringSliceFlag(cmdUpdate, doctl.ArgSecretValue, "", nil, secretValueFlagDesc)
	AddStringFlag(cmdUpdate, doctl.ArgSecretFromEnvFile, "", "", "Path to an env file containing key-value pairs to store in the secret.")
	cmdUpdate.Example = `The following example updates a secret: doctl secrets update prod-db-creds --region nyc3 --version 1 --value password=@./pw.txt`

	cmdDelete := CmdBuilder(cmd, RunCmdSecretsDelete, "delete <name>", "Delete a secret", `Schedules a secret container for soft deletion.`, Writer, aliasOpt("d", "rm"))
	AddStringFlag(cmdDelete, doctl.ArgRegionSlug, "", "", secretRegionFlagDesc)
	AddBoolFlag(cmdDelete, doctl.ArgForce, doctl.ArgShortForce, false, "Delete the secret without a confirmation prompt.")
	cmdDelete.Example = `The following example deletes a secret: doctl secrets delete prod-db-creds --region nyc3 --force`

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

	key, err := c.Doit.GetString(c.NS, doctl.ArgKey)
	if err != nil {
		return err
	}

	show, err := c.Doit.GetBool(c.NS, doctl.ArgSecretShow)
	if err != nil {
		return err
	}

	raw, err := c.Doit.GetBool(c.NS, doctl.ArgSecretRaw)
	if err != nil {
		return err
	}

	if raw && key == "" {
		return fmt.Errorf("--%s requires --%s", doctl.ArgSecretRaw, doctl.ArgKey)
	}

	secret, err := c.Secrets().Get(name, region)
	if err != nil {
		return err
	}

	if raw {
		value, ok := secret.Values[key]
		if !ok {
			return fmt.Errorf("key %q not found in secret %q", key, name)
		}
		_, err = fmt.Fprint(c.Out, value)
		return err
	}

	displaySecret := cloneSecret(*secret)
	if key != "" {
		value, ok := secret.Values[key]
		if !ok {
			return fmt.Errorf("key %q not found in secret %q", key, name)
		}
		displaySecret.Values = map[string]string{key: value}
	}

	if !show {
		displaySecret = maskSecretValues(displaySecret)
	}

	return c.Display(&displayers.Secret{Secret: displaySecret})
}

// RunCmdSecretsList lists secret containers.
func RunCmdSecretsList(c *CmdConfig) error {
	list, err := c.Secrets().List()
	if err != nil {
		return err
	}

	if len(list.UnavailableRegions) > 0 {
		secretNotice("Some regions were unavailable: %s", strings.Join(list.UnavailableRegions, ", "))
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

	force, err := c.Doit.GetBool(c.NS, doctl.ArgForce)
	if err != nil {
		return err
	}

	if force || AskForConfirmDelete("secret", 1) == nil {
		return c.Secrets().Delete(name, region)
	}

	return errOperationAborted
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
	values, err := loadSecretValuesFromFlags(c)
	if err != nil {
		return nil, err
	}
	if len(values) > 0 {
		return values, nil
	}
	if !secretsPromptsEnabled() {
		return nil, fmt.Errorf("must specify at least one --%s or --%s when running with --no-interactive",
			doctl.ArgSecretValue, doctl.ArgSecretFromEnvFile)
	}

	name := ""
	if len(c.Args) > 0 {
		name = c.Args[0]
	}
	region, _ := c.Doit.GetString(c.NS, doctl.ArgRegionSlug)

	return readSecretValuesInteractive(os.Stderr, secretValuePromptCreate(name, region))
}

func collectSecretValuesForUpdate(c *CmdConfig, name, region string) (map[string]string, error) {
	values, err := loadSecretValuesFromFlags(c)
	if err != nil {
		return nil, err
	}
	if len(values) > 0 {
		return values, nil
	}
	if !secretsPromptsEnabled() {
		return nil, fmt.Errorf("must specify at least one --%s or --%s when running with --no-interactive",
			doctl.ArgSecretValue, doctl.ArgSecretFromEnvFile)
	}

	return readSecretValuesInteractive(os.Stderr, secretValuePromptUpdate(name, region))
}

func loadSecretValuesFromFlags(c *CmdConfig) (map[string]string, error) {
	values := make(map[string]string)

	envFile, err := c.Doit.GetString(c.NS, doctl.ArgSecretFromEnvFile)
	if err != nil {
		return nil, err
	}
	if envFile != "" {
		envValues, err := loadSecretValuesFromEnvFile(envFile)
		if err != nil {
			return nil, err
		}
		for key, value := range envValues {
			values[key] = value
		}
	}

	flags, err := c.Doit.GetStringSlice(c.NS, doctl.ArgSecretValue)
	if err != nil {
		return nil, err
	}
	if len(flags) > 0 {
		flagValues, err := parseSecretValues(flags)
		if err != nil {
			return nil, err
		}
		for key, value := range flagValues {
			values[key] = value
		}
	}

	return values, nil
}

func loadSecretValuesFromEnvFile(path string) (map[string]string, error) {
	envs, err := godotenv.Read(path)
	if err != nil {
		return nil, fmt.Errorf("reading env file: %w", err)
	}
	if len(envs) == 0 {
		return nil, fmt.Errorf("env file %q contains no key-value pairs", path)
	}

	return envs, nil
}

func defaultPromptSecretKey() (string, error) {
	return input.New("Enter key name (submit empty to finish): ").Prompt()
}

func defaultPromptSecretValue(key string) (string, error) {
	return input.New(fmt.Sprintf("Enter value for %q: ", key),
		input.WithHidden(),
		input.WithRequired(),
	).Prompt()
}

func resolveSecretName(c *CmdConfig) (string, error) {
	if len(c.Args) > 0 && c.Args[0] != "" {
		return c.Args[0], nil
	}
	if !secretsPromptsEnabled() {
		return "", doctl.NewMissingArgsErr(c.NS)
	}

	name, err := input.New("Enter the secret name: ", input.WithRequired()).Prompt()
	if err != nil {
		return "", err
	}

	c.Args = []string{name}

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
		secretNotice("Using current version %d of %q in %s", secret.Version, name, region)
		c.Doit.Set(c.NS, doctl.ArgSecretVersion, secret.Version)
		return secret.Version, nil
	}

	if !secretsPromptsEnabled() {
		return 0, fmt.Errorf("must specify --%s when the current version cannot be determined", doctl.ArgSecretVersion)
	}

	versionStr, err := input.New(
		fmt.Sprintf("Enter the current version of %q in %s to update: ", name, region),
		input.WithRequired(),
		input.WithValidator(func(value string) error {
			version, err := strconv.Atoi(strings.TrimSpace(value))
			if err != nil || version <= 0 {
				return fmt.Errorf("version must be a positive integer")
			}
			return nil
		}),
	).Prompt()
	if err != nil {
		return 0, err
	}

	version, err := strconv.Atoi(strings.TrimSpace(versionStr))
	if err != nil {
		return 0, fmt.Errorf("version must be a positive integer")
	}

	c.Doit.Set(c.NS, doctl.ArgSecretVersion, version)

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
	if !secretsPromptsEnabled() {
		return "", fmt.Errorf("must specify --%s when running with --no-interactive", doctl.ArgRegionSlug)
	}

	region, err = promptSecretRegion()
	if err != nil {
		return "", err
	}

	c.Doit.Set(c.NS, doctl.ArgRegionSlug, region)

	return region, nil
}

func promptSecretRegion() (string, error) {
	items := make([]list.Item, len(exampleSecretRegions))
	for i, region := range exampleSecretRegions {
		items[i] = secretRegionListItem{slug: region}
	}

	selected, err := list.New(items).Select()
	if err != nil {
		return "", err
	}

	item, ok := selected.(secretRegionListItem)
	if !ok {
		return "", fmt.Errorf("invalid region selection")
	}

	return item.slug, nil
}

func secretValuePromptCreate(name, region string) string {
	if name != "" && region != "" {
		return fmt.Sprintf(`Creating secret %q in %s.
Enter key-value pairs to store in this secret.
Submit an empty key name when you are done.`, name, region)
	}

	return `Enter key-value pairs to store in this secret.
Submit an empty key name when you are done.`
}

func secretValuePromptUpdate(name, region string) string {
	return fmt.Sprintf(`Updating secret %q in %s.
This replaces all key-value pairs in the secret. Enter the full set you want to keep.
Submit an empty key name when you are done.`, name, region)
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

		resolved, err := resolveSecretValueInput(value)
		if err != nil {
			return nil, err
		}

		values[key] = resolved
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

func resolveSecretValueInput(value string) (string, error) {
	if value == "-" {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return "", err
		}
		return strings.TrimRight(string(data), "\r\n"), nil
	}

	if strings.HasPrefix(value, "@") {
		path := strings.TrimPrefix(value, "@")
		if path == "" {
			return "", fmt.Errorf("file path is required after @")
		}

		content, err := readInputFromFile(path)
		if err != nil {
			return "", err
		}

		return strings.TrimRight(content, "\r\n"), nil
	}

	return value, nil
}

func maskSecretValues(secret do.Secret) do.Secret {
	maskedValues := make(map[string]string, len(secret.Values))
	for key := range secret.Values {
		maskedValues[key] = secretMaskedValue
	}

	masked := cloneSecret(secret)
	masked.Values = maskedValues

	return masked
}

func cloneSecret(secret do.Secret) do.Secret {
	values := make(map[string]string, len(secret.Values))
	for key, value := range secret.Values {
		values[key] = value
	}

	return do.Secret{
		Secret: &godo.Secret{
			Name:              secret.Name,
			Region:            secret.Region,
			Version:           secret.Version,
			Values:            values,
			CreatedAt:         secret.CreatedAt,
			UpdatedAt:         secret.UpdatedAt,
			DeleteRequestedAt: secret.DeleteRequestedAt,
		},
	}
}

func readSecretValuesInteractive(out io.Writer, prompt string) (map[string]string, error) {
	fmt.Fprintln(out, prompt)

	values := make(map[string]string)
	for {
		key, err := promptSecretKeyFunc()
		if err != nil {
			return nil, err
		}

		key = strings.TrimSpace(key)
		if key == "" {
			break
		}

		if _, exists := values[key]; exists {
			return nil, fmt.Errorf("duplicate key %q", key)
		}

		value, err := promptSecretValueFunc(key)
		if err != nil {
			return nil, err
		}

		values[key] = value
	}

	if len(values) == 0 {
		return nil, fmt.Errorf("at least one key-value pair is required")
	}

	return values, nil
}

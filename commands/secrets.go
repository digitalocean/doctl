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
	"sort"
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
	exampleSecretName   = "my-secret"
	exampleSecretKey    = "key"
	exampleSecretKeyAlt = "other-key"
	exampleSecretValue  = "value"

	secretRegionFlagDesc = "Region where the secret is stored. If omitted, you are prompted when running with --interactive."
	secretValueFlagDesc  = "Key-value pair in key=value format (repeatable). Values may be read from a file with key=@path or from stdin with key=-. If omitted, keys and masked values are read with --interactive."
	secretMaskedValue    = "********"
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
	cmdCreate.Example = `The following example creates a secret with key-value pairs: doctl secrets create ` + exampleSecretName + ` --region nyc3 --value ` + exampleSecretKey + `=@./value.txt --value ` + exampleSecretKeyAlt + `=-`

	cmdGet := CmdBuilder(cmd, RunCmdSecretsGet, "get <name>", "Get a secret", `Retrieves a secret container and its key-value pairs.

Secret values are masked by default. Use --show to reveal them, or --key with --raw to print a single value for scripting.`, Writer,
		aliasOpt("g"), displayerType(&displayers.Secret{}))
	AddStringFlag(cmdGet, doctl.ArgRegionSlug, "", "", secretRegionFlagDesc)
	AddStringFlag(cmdGet, doctl.ArgKey, "", "", "Return only the value for this key.")
	AddBoolFlag(cmdGet, doctl.ArgSecretShow, "", false, "Reveal secret values instead of masking them.")
	AddBoolFlag(cmdGet, doctl.ArgSecretRaw, "", false, "Write the value for --key to stdout with no formatting.")
	cmdGet.Example = `The following example retrieves a secret: doctl secrets get ` + exampleSecretName + ` --region nyc3 --key ` + exampleSecretKey + ` --raw`

	cmdList := CmdBuilder(cmd, RunCmdSecretsList, "list", "List secrets", `Retrieves a list of secret containers across all regions. Values are not included.`, Writer,
		aliasOpt("ls"), displayerType(&displayers.Secrets{}))
	cmdList.Example = `The following example lists all secrets: doctl secrets list`

	cmdListVersions := CmdBuilder(cmd, RunCmdSecretsListVersions, "list-versions <name>", "List secret versions", `Retrieves version history for a secret container.`, Writer,
		displayerType(&displayers.SecretVersions{}))
	AddStringFlag(cmdListVersions, doctl.ArgRegionSlug, "", "", secretRegionFlagDesc)
	cmdListVersions.Example = `The following example lists versions for a secret: doctl secrets list-versions ` + exampleSecretName + ` --region nyc3`

	cmdSet := CmdBuilder(cmd, RunCmdSecretsSet, "set <name>", "Set keys on a secret", `Adds or updates key-value pairs on a secret without removing existing keys.

Fetches the current secret, merges in the provided keys, and writes a new version.`, Writer,
		displayerType(&displayers.SecretWriteResult{}))
	AddStringFlag(cmdSet, doctl.ArgRegionSlug, "", "", secretRegionFlagDesc)
	AddStringSliceFlag(cmdSet, doctl.ArgSecretValue, "", nil, secretValueFlagDesc)
	AddStringFlag(cmdSet, doctl.ArgSecretFromEnvFile, "", "", "Path to an env file containing key-value pairs to store in the secret.")
	cmdSet.Example = `The following example sets a key on a secret: doctl secrets set ` + exampleSecretName + ` --region nyc3 --value ` + exampleSecretKey + `=@./value.txt`

	cmdUnset := CmdBuilder(cmd, RunCmdSecretsUnset, "unset <name>", "Remove keys from a secret", `Removes key-value pairs from a secret without affecting other keys.

Fetches the current secret, removes the specified keys, and writes a new version.`, Writer,
		displayerType(&displayers.SecretWriteResult{}))
	AddStringFlag(cmdUnset, doctl.ArgRegionSlug, "", "", secretRegionFlagDesc)
	AddStringSliceFlag(cmdUnset, doctl.ArgKey, "", nil, "Key to remove (repeatable).", requiredOpt())
	cmdUnset.Example = `The following example removes a key from a secret: doctl secrets unset ` + exampleSecretName + ` --region nyc3 --key ` + exampleSecretKeyAlt

	cmdUpdate := CmdBuilder(cmd, RunCmdSecretsUpdate, "update <name>", "Replace a secret", `Replaces all key-value pairs in a secret with a new version.

This removes any keys not included in the update. Use `+"`"+`secrets set`+"`"+` to add or change keys, or `+"`"+`secrets unset`+"`"+` to remove keys. Pass `+"`"+`--replace`+"`"+` to confirm full replacement.`, Writer,
		displayerType(&displayers.SecretWriteResult{}))
	AddStringFlag(cmdUpdate, doctl.ArgRegionSlug, "", "", secretRegionFlagDesc)
	AddBoolFlag(cmdUpdate, doctl.ArgSecretReplace, "", false, "Replace the entire secret. Required to perform a full replacement.")
	AddBoolFlag(cmdUpdate, doctl.ArgForce, doctl.ArgShortForce, false, "Replace the secret without prompting when keys would be removed.")
	AddStringSliceFlag(cmdUpdate, doctl.ArgSecretValue, "", nil, secretValueFlagDesc)
	AddStringFlag(cmdUpdate, doctl.ArgSecretFromEnvFile, "", "", "Path to an env file containing key-value pairs to store in the secret.")
	cmdUpdate.Example = `The following example replaces a secret: doctl secrets update ` + exampleSecretName + ` --region nyc3 --replace --value ` + exampleSecretKey + `=@./value.txt --value ` + exampleSecretKeyAlt + `=` + exampleSecretValue

	cmdDelete := CmdBuilder(cmd, RunCmdSecretsDelete, "delete <name>", "Delete a secret", `Schedules a secret container for soft deletion.`, Writer, aliasOpt("d", "rm"))
	AddStringFlag(cmdDelete, doctl.ArgRegionSlug, "", "", secretRegionFlagDesc)
	AddBoolFlag(cmdDelete, doctl.ArgForce, doctl.ArgShortForce, false, "Delete the secret without a confirmation prompt.")
	cmdDelete.Example = `The following example deletes a secret: doctl secrets delete ` + exampleSecretName + ` --region nyc3 --force`

	cmdRestore := CmdBuilder(cmd, RunCmdSecretsRestore, "restore <name>", "Restore a secret", `Restores a secret container that was scheduled for deletion.`, Writer)
	AddStringFlag(cmdRestore, doctl.ArgRegionSlug, "", "", secretRegionFlagDesc)
	cmdRestore.Example = `The following example restores a secret: doctl secrets restore ` + exampleSecretName + ` --region nyc3`

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

// RunCmdSecretsSet adds or updates keys on a secret without removing existing keys.
func RunCmdSecretsSet(c *CmdConfig) error {
	name, err := resolveSecretName(c)
	if err != nil {
		return err
	}

	region, err := resolveSecretRegion(c)
	if err != nil {
		return err
	}

	updates, err := collectSecretValuesForSet(c, name, region)
	if err != nil {
		return err
	}

	current, err := c.Secrets().Get(name, region)
	if err != nil {
		return err
	}

	values := mergeSecretValues(current.Values, updates)

	return writeSecret(c, name, region, values, current.Version)
}

// RunCmdSecretsUnset removes keys from a secret.
func RunCmdSecretsUnset(c *CmdConfig) error {
	name, err := resolveSecretName(c)
	if err != nil {
		return err
	}

	region, err := resolveSecretRegion(c)
	if err != nil {
		return err
	}

	keys, err := collectSecretKeys(c)
	if err != nil {
		return err
	}

	current, err := c.Secrets().Get(name, region)
	if err != nil {
		return err
	}

	values, err := unsetSecretKeys(current.Values, keys)
	if err != nil {
		return err
	}

	return writeSecret(c, name, region, values, current.Version)
}

// RunCmdSecretsUpdate replaces all key-value pairs in a secret.
func RunCmdSecretsUpdate(c *CmdConfig) error {
	replace, err := c.Doit.GetBool(c.NS, doctl.ArgSecretReplace)
	if err != nil {
		return err
	}
	if !replace {
		return fmt.Errorf("update replaces all keys; use `secrets set` to add or change keys, `secrets unset` to remove keys, or pass --%s to replace the entire secret",
			doctl.ArgSecretReplace)
	}

	name, err := resolveSecretName(c)
	if err != nil {
		return err
	}

	region, err := resolveSecretRegion(c)
	if err != nil {
		return err
	}

	newValues, err := collectSecretValuesForReplace(c, name, region)
	if err != nil {
		return err
	}

	current, err := c.Secrets().Get(name, region)
	if err != nil {
		return err
	}

	removed := removedSecretKeys(current.Values, newValues)
	if err := confirmSecretKeysRemoved(c, name, removed); err != nil {
		return err
	}

	return writeSecret(c, name, region, newValues, current.Version)
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

func collectSecretValuesForSet(c *CmdConfig, name, region string) (map[string]string, error) {
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

	return readSecretValuesInteractive(os.Stderr, secretValuePromptSet(name, region))
}

func collectSecretValuesForReplace(c *CmdConfig, name, region string) (map[string]string, error) {
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

	return readSecretValuesInteractive(os.Stderr, secretValuePromptReplace(name, region))
}

func collectSecretKeys(c *CmdConfig) ([]string, error) {
	keys, err := c.Doit.GetStringSlice(c.NS, doctl.ArgKey)
	if err != nil {
		return nil, err
	}
	if len(keys) == 0 {
		return nil, fmt.Errorf("must specify at least one --%s", doctl.ArgKey)
	}

	return keys, nil
}

func mergeSecretValues(current, updates map[string]string) map[string]string {
	values := cloneStringMap(current)
	for key, value := range updates {
		values[key] = value
	}

	return values
}

func unsetSecretKeys(current map[string]string, keys []string) (map[string]string, error) {
	values := cloneStringMap(current)
	for _, key := range keys {
		if _, ok := values[key]; !ok {
			return nil, fmt.Errorf("key %q not found in secret", key)
		}
		delete(values, key)
	}
	if len(values) == 0 {
		return nil, fmt.Errorf("secret must contain at least one key-value pair")
	}

	return values, nil
}

func removedSecretKeys(current, next map[string]string) []string {
	var removed []string
	for key := range current {
		if _, ok := next[key]; !ok {
			removed = append(removed, key)
		}
	}
	sort.Strings(removed)

	return removed
}

func confirmSecretKeysRemoved(c *CmdConfig, name string, keys []string) error {
	if len(keys) == 0 {
		return nil
	}

	force, err := c.Doit.GetBool(c.NS, doctl.ArgForce)
	if err != nil {
		return err
	}
	if force {
		return nil
	}

	message := fmt.Sprintf("remove keys %s from %q", strings.Join(keys, ", "), name)
	if err := AskForConfirm(message); err != nil {
		if err == ErrExitSilently {
			return err
		}
		return errOperationAborted
	}

	return nil
}

func writeSecret(c *CmdConfig, name, region string, values map[string]string, version int) error {
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

func cloneStringMap(in map[string]string) map[string]string {
	out := make(map[string]string, len(in))
	for key, value := range in {
		out[key] = value
	}

	return out
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

func secretValuePromptSet(name, region string) string {
	return fmt.Sprintf(`Setting keys on secret %q in %s.
Enter key-value pairs to add or update. Existing keys not listed here are kept.
Submit an empty key name when you are done.`, name, region)
}

func secretValuePromptReplace(name, region string) string {
	return fmt.Sprintf(`Replacing secret %q in %s.
This replaces all key-value pairs in the secret. Enter the full set you want to keep.
Submit an empty key name when you are done.`, name, region)
}

// parseSecretValues parses --value pairs. The key=value / @file / - grammar
// lives in parseKeyValueInputs, shared with the agents --secret flag; only the
// "a secret needs at least one pair" rule is specific to this command.
func parseSecretValues(lines []string) (map[string]string, error) {
	if len(lines) == 0 {
		return nil, fmt.Errorf("at least one key-value pair is required")
	}
	return parseKeyValueInputs(lines)
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

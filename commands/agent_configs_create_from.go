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
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strings"

	"github.com/digitalocean/doctl/commands/charm"
	"github.com/digitalocean/doctl/commands/charm/confirm"
	"github.com/digitalocean/doctl/commands/charm/input"
	"github.com/digitalocean/doctl/commands/charm/list"
	"github.com/digitalocean/doctl/commands/charm/selection"
	"github.com/digitalocean/godo"
	"github.com/erikgeiser/promptkit"
	"sigs.k8s.io/yaml"
)

// agentConfigNameRE mirrors the API's config-name rule so a bad name is caught
// at the prompt instead of after the create round trip.
var agentConfigNameRE = regexp.MustCompile(`^[A-Za-z0-9]([A-Za-z0-9._-]{0,62}[A-Za-z0-9])?$`)

const (
	secretChoiceNew  = "Enter a new value"
	secretChoiceKeep = "Keep the current value"
)

// runAgentsConfigCreateFrom is the terminal create-from flow: pick a config
// (unless --from names one), show it, then name the new config, optionally
// edit the manifest in $EDITOR, and decide keep or new for each secret.
// Nothing is created until the final confirm, and Esc/Ctrl-C at any prompt
// creates nothing.
func runAgentsConfigCreateFrom(c *CmdConfig, from string) error {
	err := createAgentConfigFrom(c, from)
	if isPromptCanceled(err) {
		fmt.Fprintln(c.Out, colorize("Canceled. Nothing was created.", colMuted))
		return nil
	}
	return err
}

func createAgentConfigFrom(c *CmdConfig, from string) error {
	svc := c.HostedAgents()
	stylingEnabled = detectStyling()
	if from == "" {
		picked, err := pickAgentConfig(c)
		if err != nil {
			return err
		}
		from = picked
	}
	src, err := svc.GetAgentConfig(from)
	if err != nil {
		return err
	}
	// UseNumber keeps int64 manifest fields exact through the YAML round trip.
	var doc map[string]any
	dec := json.NewDecoder(bytes.NewReader(src.Manifest))
	dec.UseNumber()
	if err := dec.Decode(&doc); err != nil {
		return fmt.Errorf("reading the manifest of %s: %w", src.Name, err)
	}
	printCreateFromSource(c, src, doc)
	sourceDoc, _ := manifestYAML(doc) // snapshot to diff against after edits

	name, err := input.New("Name for the new config:",
		input.WithInitialValue(src.Name+"-2"),
		input.WithRequired(),
		input.WithValidator(func(s string) error {
			switch {
			case s == src.Name:
				return fmt.Errorf("must differ from %s while it exists", src.Name)
			case !agentConfigNameRE.MatchString(s):
				return fmt.Errorf("letters, digits, '-', '.', '_'; start and end alphanumeric")
			}
			return nil
		}),
	).Prompt()
	if err != nil {
		return err
	}

	edit, err := confirm.New("Edit the manifest in "+editorName()+" first?",
		confirm.WithDefaultChoice(confirm.No)).Prompt()
	if err != nil {
		return err
	}
	if edit == confirm.Yes {
		if doc, err = editManifestInEditor(doc, name); err != nil {
			return err
		}
	}

	// Keep starts off on every slot: nothing is copied unless chosen.
	sourceSlots := map[string]godo.HostedAgentConfigCredentialSlot{}
	for _, s := range src.Credentials {
		sourceSlots[s.Name] = s
	}
	var reuse, typed []string
	for _, slot := range manifestSecrets(doc) {
		if !isTenantSecretSource(slot.Source) || slot.HasValue {
			continue
		}
		label := slot.Name
		if slot.Provider != "" {
			label += " (" + slot.Provider + ")"
		}
		prev, onSource := sourceSlots[slot.Name]
		if onSource && isTenantSecretSource(prev.Source) && prev.Provider == slot.Provider {
			choice, err := selection.New([]string{secretChoiceNew, secretChoiceKeep},
				selection.WithFiltering(false),
				selection.WithPrompt(label+":")).Select()
			if err != nil {
				return err
			}
			if choice == secretChoiceKeep {
				reuse = append(reuse, slot.Name)
				continue
			}
		}
		value, err := input.New("New value for "+label+":", input.WithHidden(), input.WithRequired()).Prompt()
		if err != nil {
			return err
		}
		setManifestSecretValue(doc, slot.Name, value)
		typed = append(typed, slot.Name)
	}

	printCreateFromPlan(c, src, name, changedManifestKeys(sourceDoc, doc), reuse, typed)
	ok, err := confirm.New(fmt.Sprintf("Create %s from %s?", name, src.Name),
		confirm.WithDefaultChoice(confirm.Yes)).Prompt()
	if err != nil {
		return err
	}
	if ok != confirm.Yes {
		return promptkit.ErrAborted
	}

	manifest, err := manifestYAML(doc)
	if err != nil {
		return err
	}
	req := &godo.HostedAgentConfigCreateRequest{Name: name, ManifestYAML: manifest, Insights: src.Insights}
	if len(reuse) > 0 {
		// The API's pair rule: both set, or both omitted when nothing is kept.
		req.SourceConfigID, req.ReuseSecrets = src.ID, reuse
	}
	cfg, err := svc.CreateAgentConfig(req)
	if err != nil {
		return err
	}
	return printCreatedAgentConfig(c, cfg)
}

// agentConfigItem is one row in the create-from picker.
type agentConfigItem struct{ cfg godo.HostedAgentConfigSummary }

func (i agentConfigItem) Title() string       { return i.cfg.Name }
func (i agentConfigItem) FilterValue() string { return i.cfg.Name }
func (i agentConfigItem) Description() string {
	parts := []string{i.cfg.ID}
	if schema := shortAgentSchema(i.cfg.AgentSpecSchemaVersion); schema != "" {
		parts = append(parts, schema)
	}
	if !i.cfg.CreatedAt.Time.IsZero() {
		parts = append(parts, "created "+formatCreatedAt(i.cfg.CreatedAt.Time))
	}
	return strings.Join(parts, " · ")
}

// pickAgentConfig lists the team's configs and returns the one chosen with the
// arrow keys and Enter (/ filters by name).
func pickAgentConfig(c *CmdConfig) (string, error) {
	var items []agentConfigItem
	opt := &godo.HostedAgentConfigListOptions{}
	for {
		page, next, err := c.HostedAgents().ListAgentConfigs(opt)
		if err != nil {
			return "", err
		}
		for _, cfg := range page {
			items = append(items, agentConfigItem{cfg})
		}
		if next == "" {
			break
		}
		opt.PageToken = next
	}
	if len(items) == 0 {
		return "", fmt.Errorf("no agent configs to create from; create one with --spec and --name")
	}
	ll := list.New(list.Items(items))
	ll.Model().Title = "Create a new config from…"
	ll.Model().SetStatusBarItemName("config", "configs")
	picked, err := ll.Select()
	if err != nil {
		return "", err
	}
	return picked.(agentConfigItem).cfg.ID, nil
}

// printCreateFromSource is the first screen: the loaded config as GET returned
// it. Secret values are never available, so each slot shows name and provider.
func printCreateFromSource(c *CmdConfig, src *godo.HostedAgentConfig, doc map[string]any) {
	var body strings.Builder
	fmt.Fprintf(&body, "%s\n\n", boldColor("Creating from", colHighlight))
	body.WriteString(cardRow("Name", src.Name))
	body.WriteString(cardRow("ID", colorize(src.ID, colMuted)))
	body.WriteString(cardRow("Insights", colorize(insightsSummary(src.Insights), colMuted)))
	for i, slot := range manifestSecrets(doc) {
		label := ""
		if i == 0 {
			label = "Secrets"
		}
		kind := slot.Source
		if kind == "" {
			kind = "tenantSecret"
		}
		if slot.Provider != "" {
			kind += " · " + slot.Provider
		}
		body.WriteString(cardRow(label, slot.Name+"  "+colorize(kind+" · value hidden", colMuted)))
	}
	renderAgentCard(c.Out, body.String())
	if manifest, err := manifestYAML(doc); err == nil {
		fmt.Fprintln(c.Out, colorize("Manifest", colMuted))
		for _, line := range strings.Split(strings.TrimRight(manifest, "\n"), "\n") {
			fmt.Fprintln(c.Out, colorize("  "+line, colMuted))
		}
		fmt.Fprintln(c.Out)
	}
}

// printCreateFromPlan summarizes the request before the confirm.
func printCreateFromPlan(c *CmdConfig, src *godo.HostedAgentConfig, name string, changed, reuse, typed []string) {
	none := colorize("none", colMuted)
	joined := func(names []string) string {
		if len(names) == 0 {
			return none
		}
		return strings.Join(names, ", ")
	}
	var body strings.Builder
	fmt.Fprintf(&body, "%s\n\n", boldColor("Ready to create", colHighlight))
	body.WriteString(cardRow("New config", name))
	body.WriteString(cardRow("From", src.Name+" "+colorize("("+src.ID+")", colMuted)))
	body.WriteString(cardRow("Changed", joined(changed)))
	body.WriteString(cardRow("Keeps", joined(reuse)))
	body.WriteString(cardRow("New values", joined(typed)))
	body.WriteString(cardRow("Insights", colorize(insightsSummary(src.Insights)+" (copied)", colMuted)))
	fmt.Fprintln(&body)
	body.WriteString(colorize("A new config is created. "+src.Name+" and its sessions are unchanged.", colMuted))
	renderAgentCard(c.Out, body.String())
}

// changedManifestKeys names the top-level manifest keys the edit changed,
// ignoring secret values typed afterwards (those are listed separately).
func changedManifestKeys(sourceYAML string, doc map[string]any) []string {
	var before map[string]any
	if err := yaml.Unmarshal([]byte(sourceYAML), &before); err != nil {
		return nil
	}
	var changed []string
	seen := map[string]bool{}
	for _, m := range []map[string]any{before, doc} {
		for k := range m {
			if seen[k] || k == "secrets" || k == "spec" {
				continue
			}
			seen[k] = true
			a, _ := json.Marshal(before[k])
			b, _ := json.Marshal(doc[k])
			if !bytes.Equal(a, b) {
				changed = append(changed, k)
			}
		}
	}
	sort.Strings(changed)
	return changed
}

func insightsSummary(in *godo.HostedAgentInsightsOptIn) string {
	if in == nil {
		return "product default"
	}
	var parts []string
	for _, s := range []struct {
		name string
		v    *bool
	}{{"metrics", in.Metrics}, {"logs", in.Logs}, {"traces", in.Traces}} {
		if s.v != nil {
			state := "off"
			if *s.v {
				state = "on"
			}
			parts = append(parts, s.name+" "+state)
		}
	}
	if len(parts) == 0 {
		return "product default"
	}
	return strings.Join(parts, ", ")
}

// manifestSlot is one secret declared in a manifest document, flat or legacy.
type manifestSlot struct {
	Name, Source, Provider string
	HasValue               bool
}

func isTenantSecretSource(source string) bool {
	return source == "" || source == "tenantSecret"
}

// manifestSecrets lists the declared secrets: flat `secrets` is a map of slot
// name to a long-form object or a shorthand string ("oauth/<provider>", or a
// value); the legacy envelope has a `spec.secrets` array.
func manifestSecrets(doc map[string]any) []manifestSlot {
	var out []manifestSlot
	if spec, ok := doc["spec"].(map[string]any); ok {
		items, _ := spec["secrets"].([]any)
		for _, item := range items {
			m, _ := item.(map[string]any)
			out = append(out, slotFromMap(fmt.Sprint(m["name"]), m))
		}
		return out
	}
	secrets, _ := doc["secrets"].(map[string]any)
	for name, v := range secrets {
		switch v := v.(type) {
		case map[string]any:
			out = append(out, slotFromMap(name, v))
		case string:
			if provider, ok := strings.CutPrefix(v, "oauth/"); ok {
				out = append(out, manifestSlot{Name: name, Source: "oauth", Provider: provider})
			} else {
				out = append(out, manifestSlot{Name: name, HasValue: strings.TrimSpace(v) != ""})
			}
		default:
			out = append(out, manifestSlot{Name: name})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func slotFromMap(name string, m map[string]any) manifestSlot {
	str := func(k string) string { s, _ := m[k].(string); return s }
	return manifestSlot{
		Name:     name,
		Source:   str("source"),
		Provider: str("provider"),
		HasValue: strings.TrimSpace(str("value")) != "",
	}
}

// setManifestSecretValue writes a typed value onto the slot, in either shape.
func setManifestSecretValue(doc map[string]any, name, value string) {
	if spec, ok := doc["spec"].(map[string]any); ok {
		items, _ := spec["secrets"].([]any)
		for _, item := range items {
			if m, ok := item.(map[string]any); ok && m["name"] == name {
				m["value"] = value
			}
		}
		return
	}
	secrets, _ := doc["secrets"].(map[string]any)
	if m, ok := secrets[name].(map[string]any); ok {
		m["value"] = value
		return
	}
	secrets[name] = map[string]any{"source": "tenantSecret", "value": value}
}

// reusedSlotWithValue returns a --reuse-secret slot that also has a value in
// the spec (the API would 400 on it), or "" when there is none.
func reusedSlotWithValue(manifest []byte, reuse []string) string {
	var doc map[string]any
	if err := yaml.Unmarshal(manifest, &doc); err != nil {
		return "" // the API owns the parse error
	}
	listed := map[string]bool{}
	for _, name := range reuse {
		listed[name] = true
	}
	for _, slot := range manifestSecrets(doc) {
		if listed[slot.Name] && slot.HasValue {
			return slot.Name
		}
	}
	return ""
}

func manifestYAML(doc map[string]any) (string, error) {
	out, err := yaml.Marshal(doc)
	if err != nil {
		return "", fmt.Errorf("rendering manifest: %w", err)
	}
	return string(out), nil
}

func editorName() string {
	for _, env := range []string{"VISUAL", "EDITOR"} {
		if e := strings.TrimSpace(os.Getenv(env)); e != "" {
			return e
		}
	}
	return "vi"
}

// editManifestInEditor opens the manifest in $VISUAL/$EDITOR and re-reads it,
// offering another edit when the result is not valid YAML.
func editManifestInEditor(doc map[string]any, name string) (map[string]any, error) {
	manifest, err := manifestYAML(doc)
	if err != nil {
		return nil, err
	}
	f, err := os.CreateTemp("", "doctl-agent-config-*.yaml")
	if err != nil {
		return nil, err
	}
	defer os.Remove(f.Name())
	header := "# Manifest for " + name + ". Secret values are not shown: after saving,\n" +
		"# choose keep or a new value for each secret.\n"
	if _, err := f.WriteString(header + manifest); err != nil {
		return nil, err
	}
	f.Close()
	for {
		args := append(strings.Fields(editorName()), f.Name())
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
		if err := cmd.Run(); err != nil {
			return nil, fmt.Errorf("running %s: %w", args[0], err)
		}
		raw, err := os.ReadFile(f.Name())
		if err != nil {
			return nil, err
		}
		var edited map[string]any
		parseErr := yaml.Unmarshal(raw, &edited)
		if parseErr == nil && edited != nil {
			return edited, nil
		}
		if parseErr == nil {
			parseErr = errors.New("the manifest is empty")
		}
		again, err := confirm.New(fmt.Sprintf("Invalid manifest (%v). Edit again?", parseErr),
			confirm.WithDefaultChoice(confirm.Yes)).Prompt()
		if err != nil {
			return nil, err
		}
		if again != confirm.Yes {
			return nil, promptkit.ErrAborted
		}
	}
}

func isPromptCanceled(err error) bool {
	return err != nil && (errors.Is(err, promptkit.ErrAborted) ||
		errors.Is(err, charm.ErrCanceled) || err.Error() == "canceled")
}

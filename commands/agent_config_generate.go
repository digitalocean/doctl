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
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/digitalocean/doctl"
)

// `harness-runtime config generate` writes an agent manifest and nothing else:
// no session, no durable config, no API write of any kind. The generated file is
// the input to `config create` / `create --spec`, so authoring it is separable
// from deciding when to run it.
//
// Every field is reachable two ways — a flag, or a wizard prompt — and the two
// converge on the same generateAnswers value, so an interactive run is always
// reproducible as a single scripted command.

// addAgentConfigGenerateFlags registers the manifest-authoring flags. They are
// grouped in the same order as the manifest sections they write, so `--help`
// reads like the file it produces.
func addAgentConfigGenerateFlags(cmd *Command) {
	// Identity.
	AddStringFlag(cmd, doctl.ArgAgentName, "", "", "Session name written to the manifest. Team-unique among live sessions; omit to let the server generate one.")
	AddStringFlag(cmd, doctl.ArgAgentDescription, "", "", "Free-form description stored with the manifest")
	AddStringSliceFlag(cmd, doctl.ArgAgentLabel, "", nil, "Manifest label as KEY=VALUE. Repeatable.")

	// Agent.
	AddStringFlag(cmd, doctl.ArgAgentHarness, "", "", "Coding-agent harness the manifest runs: "+generateHarnessIDList()+". Prompted for when omitted on a terminal.")
	AddStringFlag(cmd, doctl.ArgAgentImage, "", "", "Digest-pinned container image. Only with --"+doctl.ArgAgentHarness+" custom.")
	AddStringSliceFlag(cmd, doctl.ArgAgentEntrypoint, "", nil, "Container entrypoint argv. Only with --"+doctl.ArgAgentHarness+" custom.")
	AddStringFlag(cmd, doctl.ArgAgentTriggerPrompt, "", "", "Initial prompt embedded in the runtime config. Only with --"+doctl.ArgAgentHarness+" codex-agentapi.")

	// Workspace.
	AddStringSliceFlag(cmd, doctl.ArgAgentRepo, "", nil, "GitHub repository the session works against (owner/repo or an https://github.com URL). Repeatable; the first is the session's repo hint.")
	AddBoolFlag(cmd, doctl.ArgAgentGitHubAccess, "", false, "Declare the brokered GitHub credential slot ("+generateGitHubOAuthSlot+": "+generateGitHubOAuthValue+") and allow GitHub egress")
	AddStringFlag(cmd, doctl.ArgAgentTemplate, "", "", "Pin a sandbox template. Omit to derive it from the harness (recommended).")
	AddBoolFlag(cmd, doctl.ArgAgentPersistentWorkspace, "", false, "Keep /workspace across idle suspend and resume")
	AddStringFlag(cmd, doctl.ArgAgentSize, "", "", "Sandbox size slug, from "+agentCLI+" sizes list. Omit for the service default.")

	// Environment and credentials.
	AddStringSliceFlag(cmd, doctl.ArgAgentEnv, "", nil, "Non-secret guest environment entry as KEY=VALUE. Repeatable. The guest environment is debug-readable — put credentials in --"+doctl.ArgAgentSecret+" instead.")
	AddStringSliceFlag(cmd, doctl.ArgAgentSecret, "", nil, "Declare a credential slot: NAME for a ${NAME} placeholder doctl expands at create time, or NAME=oauth/<provider> for a brokered slot. Repeatable. Values are never written to the manifest, so the file stays committable — supply them with "+agentCLI+" config create --"+doctl.ArgAgentSecret+" NAME=VALUE.")
	AddStringSliceFlag(cmd, doctl.ArgAgentInlineSecret, "", nil, "Write a literal credential into the manifest as NAME=VALUE (also NAME=@path and NAME=-). Repeatable. Prefer --"+doctl.ArgAgentSecret+": an inlined value lands in plaintext in the generated file.")

	// Inference and model.
	AddStringFlag(cmd, doctl.ArgAgentInference, "", "", "Where the agent gets inference: "+inferenceProviderDO+" (DigitalOcean Serverless Inference, recommended), "+inferenceProviderNative+" (its own vendor's API), or "+inferenceProviderCustom+" (any other OpenAI-compatible endpoint, with --"+doctl.ArgAgentInferenceURL+"). Anything but "+inferenceProviderNative+" also requires --"+doctl.ArgAgentModel+", so "+inferenceProviderNative+" is the default when not prompting.")
	AddStringFlag(cmd, doctl.ArgAgentInferenceURL, "", "", "OpenAI-compatible base URL, usually ending in /v1. Implies --"+doctl.ArgAgentInference+" "+inferenceProviderCustom+", and adds the host to egress.")
	AddStringFlag(cmd, doctl.ArgAgentModel, "", "", "Model id, written to the environment variables this harness reads. Required unless --"+doctl.ArgAgentInference+" is "+inferenceProviderNative+", where omitting it uses the agent's own default. Ids are provider-specific: anthropic-claude-4.5-sonnet at DigitalOcean ("+doInferenceModelsCmd+"), claude-sonnet-4-5 at Anthropic.")
	AddStringFlag(cmd, doctl.ArgAgentModelRouting, "", "", "Model routing profile recorded in the manifest's model block. Requires --"+doctl.ArgAgentModel+".")

	// Lifecycle.
	AddStringFlag(cmd, doctl.ArgAgentIdleTimeout, "", "", "Idle suspend for the session as a duration (10m, 90s, 1h30m). Omit for the team default.")
	AddStringFlag(cmd, doctl.ArgAgentMaxLifetime, "", "", "Absolute session lifetime ceiling as a duration (e.g. 8h), regardless of activity")

	// Network, tools, permissions.
	AddStringSliceFlag(cmd, doctl.ArgAgentEgress, "", nil, "Host the session may reach. Repeatable. Intersected server-side with team and platform policy.")
	AddStringFlag(cmd, doctl.ArgAgentToolsPreset, "", "", "Attach a curated DO tool bundle: default ("+strings.Join(generateDefaultToolsPreset, ", ")+"), or none to attach nothing. Defaults to default.")
	AddStringSliceFlag(cmd, doctl.ArgAgentTool, "", nil, "DO catalog tool attachment: do.actions for everything it advertises, or do.actions:web_search,execute_code to select. Repeatable.")
	AddStringSliceFlag(cmd, doctl.ArgAgentMCPServer, "", nil, "Inline MCP server as NAME=https://host/mcp. Repeatable. Its host is added to egress for you.")
	AddStringSliceFlag(cmd, doctl.ArgAgentMCPTools, "", nil, "Tools to select on an inline MCP server, as NAME:tool[,tool]. Repeatable.")
	AddStringSliceFlag(cmd, doctl.ArgAgentMCPAuthSecret, "", nil, "Credential slot an inline MCP server authenticates with, as NAME=SECRET_SLOT. The slot must also be declared with --"+doctl.ArgAgentSecret+".")
	AddStringFlag(cmd, doctl.ArgAgentPermissionDefault, "", "", "Disposition for tool calls no rule matches: allow, ask, or deny")
	AddStringSliceFlag(cmd, doctl.ArgAgentAllowTool, "", nil, "Allow a tool, as TOOL or TOOL:command-match (e.g. bash:\"git status\"). Repeatable.")
	AddStringSliceFlag(cmd, doctl.ArgAgentAskTool, "", nil, "Require approval for a tool, as TOOL or TOOL:command-match. Repeatable.")
	AddStringSliceFlag(cmd, doctl.ArgAgentDenyTool, "", nil, "Deny a tool, as TOOL or TOOL:command-match (e.g. bash:\"rm -rf *\"). Repeatable.")

	// Skills.
	AddStringSliceFlag(cmd, doctl.ArgAgentSkill, "", nil, "Inline a skill from a markdown file, as NAME=path/to/SKILL.md. Repeatable.")
	AddStringSliceFlag(cmd, doctl.ArgAgentSkillDescription, "", nil, "Override a skill's description, as NAME=description. Repeatable. Defaults to the file's first heading.")

	// Serving mode.
	AddStringFlag(cmd, doctl.ArgAgentMode, "", "", "Lifecycle mode: interactive (one session, one machine) or served (an autoscaled fleet)")
	AddIntFlag(cmd, doctl.ArgAgentServingMin, "", 0, "Minimum fleet replicas. Only with --"+doctl.ArgAgentMode+" served.")
	AddIntFlag(cmd, doctl.ArgAgentServingMax, "", 0, "Maximum fleet replicas. Only with --"+doctl.ArgAgentMode+" served.")
	AddIntFlag(cmd, doctl.ArgAgentServingConcurrency, "", 0, "Target concurrent runs per replica. Only with --"+doctl.ArgAgentMode+" served.")
	AddStringFlag(cmd, doctl.ArgAgentServingScaleToZeroIdle, "", "", "Idle duration before the fleet scales to zero (e.g. 10m). Only with --"+doctl.ArgAgentMode+" served.")

	// Output.
	AddStringFlag(cmd, doctl.ArgAgentOut, "", "", "Write the manifest to this path instead of stdout")
	AddStringFlag(cmd, doctl.ArgAgentManifestFormat, "", "", "Manifest encoding: yaml (default) or json")
	AddBoolFlag(cmd, doctl.ArgAgentOverwrite, "", false, "Replace the file at --"+doctl.ArgAgentOut+" if it already exists")
	AddBoolFlag(cmd, doctl.ArgAgentSkipValidate, "", false, "Skip the client-side manifest check that otherwise runs before the manifest is written")
	AddBoolFlag(cmd, doctl.ArgAgentNoInteractive, "", false, "Do not prompt. Every question not answered by a flag takes its default. Unlike the wizard, nothing is inferred from the working directory, so the same flags always produce the same manifest.")
}

// RunAgentsConfigGenerate authors a manifest from flags, prompts, or both, and
// writes it to stdout or a file. It performs no API writes.
func RunAgentsConfigGenerate(c *CmdConfig) error {
	answers, err := agentGenerateAnswersFromFlags(c)
	if err != nil {
		return err
	}

	outPath, err := c.Doit.GetString(c.NS, doctl.ArgAgentOut)
	if err != nil {
		return err
	}
	outPath = strings.TrimSpace(outPath)

	format, err := resolveGenerateFormat(c)
	if err != nil {
		return err
	}

	// Chrome (the review card, warnings, next steps) must never mix with the
	// manifest itself: when the manifest goes to stdout, notes go to stderr, so
	// `generate > agents.yaml` yields a clean file either way.
	dest := &generateDestination{path: outPath, format: format}
	notes := c.Out
	if dest.path == "" {
		notes = os.Stderr
	}

	noInteractive, err := c.Doit.GetBool(c.NS, doctl.ArgAgentNoInteractive)
	if err != nil {
		return err
	}

	if !noInteractive && agentGenerateCanPrompt() {
		if err := runAgentGenerateWizard(c, answers, dest, notes); err != nil {
			if isPromptCanceled(err) {
				fmt.Fprintf(notes, "%s\n", colorize("Canceled — nothing was written.", colMuted))
				return nil
			}
			return err
		}
	} else {
		// The harness is the one question with no sensible default, so it is
		// also the only flag that is ever required.
		if strings.TrimSpace(answers.Harness) == "" {
			return fmt.Errorf("--%s is required when not prompting; pass one of %s", doctl.ArgAgentHarness, generateHarnessIDList())
		}
		applyGenerateEnterDefaults(answers)
	}

	if err := applyGenerateDerivations(answers); err != nil {
		return err
	}

	manifest, err := buildGeneratedManifest(answers)
	if err != nil {
		return err
	}
	encoded, err := encodeGeneratedManifest(manifest, dest.format)
	if err != nil {
		return err
	}

	skipValidate, err := c.Doit.GetBool(c.NS, doctl.ArgAgentSkipValidate)
	if err != nil {
		return err
	}
	if !skipValidate {
		// Validation is advisory here by design: `generate` writes a draft, and a
		// warning about a field the server accepts-but-does-not-yet-enforce should
		// not stop a user from saving it. Hard errors still fail.
		v := validateAgentManifest(encoded)
		if !v.ok() {
			return v.error()
		}
		printAgentManifestWarnings(notes, v.Warnings)
	}

	return writeGeneratedManifest(c, answers, manifest, encoded, dest, notes)
}

// generateDestination is where the manifest lands. The wizard may set it, so it
// is threaded through rather than re-read from flags after prompting.
type generateDestination struct {
	path      string
	format    string
	overwrite bool
}

// agentGenerateAnswersFromFlags reads every manifest flag. Unset flags leave
// their field zero, which is what the wizard treats as "still to ask" and what
// the builder treats as "omit, and let the server default apply".
func agentGenerateAnswersFromFlags(c *CmdConfig) (*generateAnswers, error) {
	a := &generateAnswers{}

	str := func(flag string) (string, error) {
		v, err := c.Doit.GetString(c.NS, flag)
		return strings.TrimSpace(v), err
	}

	var err error
	if a.Name, err = str(doctl.ArgAgentName); err != nil {
		return nil, err
	}
	if a.Description, err = str(doctl.ArgAgentDescription); err != nil {
		return nil, err
	}
	labels, err := c.Doit.GetStringSlice(c.NS, doctl.ArgAgentLabel)
	if err != nil {
		return nil, err
	}
	if a.Labels, err = parseGenerateKeyValues(doctl.ArgAgentLabel, labels); err != nil {
		return nil, err
	}

	if a.Harness, err = str(doctl.ArgAgentHarness); err != nil {
		return nil, err
	}
	if a.Harness != "" {
		// Fail on a bad harness now rather than after a dozen prompts.
		if a.Harness, err = resolveGenerateHarness(a.Harness); err != nil {
			return nil, err
		}
	}
	if a.Image, err = str(doctl.ArgAgentImage); err != nil {
		return nil, err
	}
	if a.Entrypoint, err = c.Doit.GetStringSlice(c.NS, doctl.ArgAgentEntrypoint); err != nil {
		return nil, err
	}
	if a.Prompt, err = str(doctl.ArgAgentTriggerPrompt); err != nil {
		return nil, err
	}

	repos, err := c.Doit.GetStringSlice(c.NS, doctl.ArgAgentRepo)
	if err != nil {
		return nil, err
	}
	for _, repo := range repos {
		ref, err := normalizeHarnessRepoRef(repo)
		if err != nil {
			return nil, err
		}
		if ref != "" {
			a.Repos = append(a.Repos, ref)
		}
	}
	if a.GitHubAccess, err = c.Doit.GetBool(c.NS, doctl.ArgAgentGitHubAccess); err != nil {
		return nil, err
	}
	if a.Template, err = str(doctl.ArgAgentTemplate); err != nil {
		return nil, err
	}
	if a.PersistentWorkspace, err = c.Doit.GetBoolPtr(c.NS, doctl.ArgAgentPersistentWorkspace); err != nil {
		return nil, err
	}
	if a.Size, err = str(doctl.ArgAgentSize); err != nil {
		return nil, err
	}

	env, err := c.Doit.GetStringSlice(c.NS, doctl.ArgAgentEnv)
	if err != nil {
		return nil, err
	}
	if a.Env, err = parseGenerateKeyValues(doctl.ArgAgentEnv, env); err != nil {
		return nil, err
	}
	if err := rejectCredentialLookingEnv(a.Env); err != nil {
		return nil, err
	}

	if err := readGenerateSecretFlags(c, a); err != nil {
		return nil, err
	}

	if a.InferenceProvider, err = str(doctl.ArgAgentInference); err != nil {
		return nil, err
	}
	if a.InferenceURL, err = str(doctl.ArgAgentInferenceURL); err != nil {
		return nil, err
	}
	if a.InferenceURL != "" && a.InferenceProvider == "" {
		// A URL says everything the provider flag would.
		a.InferenceProvider = inferenceProviderCustom
	}
	if a.Model, err = str(doctl.ArgAgentModel); err != nil {
		return nil, err
	}
	if a.ModelRouting, err = str(doctl.ArgAgentModelRouting); err != nil {
		return nil, err
	}
	if a.IdleTimeout, err = str(doctl.ArgAgentIdleTimeout); err != nil {
		return nil, err
	}
	if a.MaxLifetime, err = str(doctl.ArgAgentMaxLifetime); err != nil {
		return nil, err
	}

	if a.Egress, err = c.Doit.GetStringSlice(c.NS, doctl.ArgAgentEgress); err != nil {
		return nil, err
	}
	if a.ToolsPreset, err = str(doctl.ArgAgentToolsPreset); err != nil {
		return nil, err
	}
	if err := validateToolsPreset(a.ToolsPreset); err != nil {
		return nil, err
	}
	if a.Tools, err = c.Doit.GetStringSlice(c.NS, doctl.ArgAgentTool); err != nil {
		return nil, err
	}

	mcpServers, err := c.Doit.GetStringSlice(c.NS, doctl.ArgAgentMCPServer)
	if err != nil {
		return nil, err
	}
	mcpTools, err := c.Doit.GetStringSlice(c.NS, doctl.ArgAgentMCPTools)
	if err != nil {
		return nil, err
	}
	mcpAuth, err := c.Doit.GetStringSlice(c.NS, doctl.ArgAgentMCPAuthSecret)
	if err != nil {
		return nil, err
	}
	if a.MCPServers, err = parseGenerateMCPServers(mcpServers, mcpTools, mcpAuth); err != nil {
		return nil, err
	}

	if a.PermissionDefault, err = str(doctl.ArgAgentPermissionDefault); err != nil {
		return nil, err
	}
	if a.PermissionDefault != "" {
		if _, ok := validPermissionDefaults[a.PermissionDefault]; !ok {
			return nil, fmt.Errorf("--%s must be allow, ask, or deny (got %q)", doctl.ArgAgentPermissionDefault, a.PermissionDefault)
		}
	}
	if err := readGeneratePermissionRules(c, a); err != nil {
		return nil, err
	}

	skills, err := c.Doit.GetStringSlice(c.NS, doctl.ArgAgentSkill)
	if err != nil {
		return nil, err
	}
	skillDescriptions, err := c.Doit.GetStringSlice(c.NS, doctl.ArgAgentSkillDescription)
	if err != nil {
		return nil, err
	}
	if a.Skills, err = parseGenerateSkills(skills, skillDescriptions); err != nil {
		return nil, err
	}

	if a.Mode, err = str(doctl.ArgAgentMode); err != nil {
		return nil, err
	}
	if a.Serving, err = readGenerateServing(c); err != nil {
		return nil, err
	}

	return a, nil
}

// readGenerateSecretFlags turns --secret and --inline-secret into ordered slot
// declarations.
//
// --secret carries no value into the file on purpose: the manifest is a file
// people commit, and the create-time --secret flag already exists to supply
// values. --inline-secret is the explicit opt-out for a throwaway local file.
func readGenerateSecretFlags(c *CmdConfig, a *generateAnswers) error {
	slots, err := c.Doit.GetStringSlice(c.NS, doctl.ArgAgentSecret)
	if err != nil {
		return err
	}
	for _, raw := range slots {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		name, value, hasValue := strings.Cut(raw, "=")
		name = strings.TrimSpace(name)
		value = strings.TrimSpace(value)
		switch {
		case !hasValue || value == "":
			a.addSecretSlot(generateSecretSlot{Name: name, Value: secretPlaceholder(name)})
		case strings.HasPrefix(value, "oauth/"):
			a.addSecretSlot(generateSecretSlot{Name: name, Value: value})
		default:
			return fmt.Errorf("--%s %s=… carries a credential value, which is never written to a generated manifest. Declare the slot with `--%s %s` and pass the value at create time, or use `--%s %s=VALUE` to embed it deliberately",
				doctl.ArgAgentSecret, name, doctl.ArgAgentSecret, name, doctl.ArgAgentInlineSecret, name)
		}
	}

	inline, err := c.Doit.GetStringSlice(c.NS, doctl.ArgAgentInlineSecret)
	if err != nil {
		return err
	}
	// Inline values do reach the file, so they get the full grammar: a literal,
	// @path, or - for stdin, exactly as `create --secret` spells it.
	values, err := parseKeyValueInputs(inline)
	if err != nil {
		return fmt.Errorf("--%s: %w", doctl.ArgAgentInlineSecret, err)
	}
	for _, name := range sortedKeys(values) {
		if a.hasSecret(name) {
			return fmt.Errorf("secret slot %q is declared twice (--%s and --%s)", name, doctl.ArgAgentSecret, doctl.ArgAgentInlineSecret)
		}
		a.addSecretSlot(generateSecretSlot{Name: name, Value: values[name], Inline: true})
	}
	return nil
}

func readGeneratePermissionRules(c *CmdConfig, a *generateAnswers) error {
	for _, spec := range []struct {
		flag   string
		action string
	}{
		{doctl.ArgAgentAllowTool, "allow"},
		{doctl.ArgAgentAskTool, "ask"},
		{doctl.ArgAgentDenyTool, "deny"},
	} {
		values, err := c.Doit.GetStringSlice(c.NS, spec.flag)
		if err != nil {
			return err
		}
		rules, err := parseGeneratePermissionRules(spec.flag, spec.action, values)
		if err != nil {
			return err
		}
		a.PermissionRules = append(a.PermissionRules, rules...)
	}
	return nil
}

// readGenerateServing collects the fleet flags, returning nil when none were
// given so `mode: interactive` never sprouts an empty serving block.
func readGenerateServing(c *CmdConfig) (*generatedServing, error) {
	min, err := c.Doit.GetIntPtr(c.NS, doctl.ArgAgentServingMin)
	if err != nil {
		return nil, err
	}
	max, err := c.Doit.GetIntPtr(c.NS, doctl.ArgAgentServingMax)
	if err != nil {
		return nil, err
	}
	concurrency, err := c.Doit.GetIntPtr(c.NS, doctl.ArgAgentServingConcurrency)
	if err != nil {
		return nil, err
	}
	idle, err := c.Doit.GetString(c.NS, doctl.ArgAgentServingScaleToZeroIdle)
	if err != nil {
		return nil, err
	}
	idle = strings.TrimSpace(idle)
	if min == nil && max == nil && concurrency == nil && idle == "" {
		return nil, nil
	}
	if err := validateGenerateDuration(doctl.ArgAgentServingScaleToZeroIdle, idle); err != nil {
		return nil, err
	}
	serving := &generatedServing{Min: min, Max: max, TargetConcurrency: concurrency}
	if idle != "" {
		serving.ScaleToZero = &generatedScaleToZero{Idle: idle}
	}
	if min != nil && max != nil && *max < *min {
		return nil, fmt.Errorf("--%s must be greater than or equal to --%s", doctl.ArgAgentServingMax, doctl.ArgAgentServingMin)
	}
	return serving, nil
}

// rejectCredentialLookingEnv refuses a credential in `env`, reusing the same
// detector `validate` warns with. Here it is an error rather than a warning:
// the file has not been written yet, so the fix is free.
func rejectCredentialLookingEnv(env map[string]string) error {
	for _, key := range sortedKeys(env) {
		if _, reserved := reservedAgentEnvKeys[strings.ToUpper(key)]; reserved {
			return fmt.Errorf("--%s %s: reserved platform environment key", doctl.ArgAgentEnv, key)
		}
		value := env[key]
		if isCredentialLookingEnvKey(key) || hasCredentialPrefix(value) {
			return fmt.Errorf("--%s %s looks like a credential; declare it with --%s %s instead (the guest environment is debug-readable and stored as-is)", doctl.ArgAgentEnv, key, doctl.ArgAgentSecret, key)
		}
	}
	return nil
}

func validateToolsPreset(preset string) error {
	switch strings.ToLower(strings.TrimSpace(preset)) {
	case "", generateToolsPresetNone, generateToolsPresetDefault:
		return nil
	default:
		return fmt.Errorf("--%s must be %s or %s (got %q)", doctl.ArgAgentToolsPreset, generateToolsPresetNone, generateToolsPresetDefault, preset)
	}
}

// resolveGenerateFormat picks the manifest encoding: the explicit flag first,
// then doctl's global -o json, then YAML.
func resolveGenerateFormat(c *CmdConfig) (string, error) {
	format, err := c.Doit.GetString(c.NS, doctl.ArgAgentManifestFormat)
	if err != nil {
		return "", err
	}
	format = strings.ToLower(strings.TrimSpace(format))
	switch format {
	case generateFormatYAML, "yml", generateFormatJSON:
		return format, nil
	case "":
		if Output == "json" {
			return generateFormatJSON, nil
		}
		return generateFormatYAML, nil
	default:
		return "", fmt.Errorf("--%s must be %s or %s (got %q)", doctl.ArgAgentManifestFormat, generateFormatYAML, generateFormatJSON, format)
	}
}

// --- output -----------------------------------------------------------------

// writeGeneratedManifest emits the manifest and, when a human is watching, the
// review card and the commands to run next.
func writeGeneratedManifest(c *CmdConfig, a *generateAnswers, m *generatedManifest, encoded []byte, dest *generateDestination, notes io.Writer) error {
	overwrite, err := c.Doit.GetBool(c.NS, doctl.ArgAgentOverwrite)
	if err != nil {
		return err
	}
	overwrite = overwrite || dest.overwrite

	if dest.path == "" {
		stylingEnabled = detectStyling()
		// The label goes to stderr with the rest of the chrome, so stdout still
		// carries nothing but the manifest.
		if isTerminalWriter(notes) {
			fmt.Fprintf(notes, "\n%s\n", colorize("Manifest · "+dest.format, colMuted))
		}
		if _, err := io.WriteString(c.Out, highlightManifest(encoded)); err != nil {
			return err
		}
		if isTerminalWriter(notes) {
			printGenerateNextSteps(notes, a, m, "")
		}
		return nil
	}

	path := filepath.Clean(dest.path)
	if !overwrite {
		if _, err := os.Stat(path); err == nil {
			return fmt.Errorf("%s already exists; pass --%s to replace it, or choose another --%s", path, doctl.ArgAgentOverwrite, doctl.ArgAgentOut)
		} else if !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	if err := os.WriteFile(path, encoded, 0600); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}

	stylingEnabled = detectStyling()
	printAgentSuccess(notes, fmt.Sprintf("Wrote %s", path))
	printGenerateNextSteps(notes, a, m, path)
	return nil
}

// highlightManifest colors the document's structure — keys against values,
// sequence markers, and the placeholders still waiting on a value.
//
// Color is the only decoration used here, and deliberately so: this block
// exists to be copied, and a border or an added indent would come along with
// it. ANSI codes do not, since a terminal keeps only plain text in its
// selection buffer.
//
// It applies solely when stdout is a terminal, so `generate > agents.yaml` and
// `generate | config create --spec -` still receive the exact bytes that were
// validated.
func highlightManifest(encoded []byte) string {
	if !stylingEnabled {
		return string(encoded)
	}
	lines := strings.Split(string(encoded), "\n")
	for i, line := range lines {
		lines[i] = highlightManifestLine(line)
	}
	return strings.Join(lines, "\n")
}

func highlightManifestLine(line string) string {
	indent := line[:len(line)-len(strings.TrimLeft(line, " "))]
	node := line[len(indent):]
	switch {
	case node == "":
		return line
	case strings.HasPrefix(node, "#"):
		return indent + colorize(node, colMuted)
	// A JSON brace or bracket alone on its line is pure syntax.
	case isManifestPunctuation(node):
		return indent + colorize(node, colMuted)
	case node == "-":
		return indent + colorize(node, colMuted)
	case strings.HasPrefix(node, "- "):
		// Dim the marker, then treat what follows as its own node: it may be a
		// scalar or the first key of a mapping in a sequence.
		return indent + colorize("- ", colMuted) + highlightManifestLine(node[2:])
	}
	if key, sep, value, ok := splitManifestKey(node); ok {
		out := boldColor(key, colHighlight) + colorize(sep, colMuted)
		if value != "" {
			out += highlightManifestValue(value)
		}
		return indent + out
	}
	return indent + highlightManifestValue(node)
}

// splitManifestKey finds the mapping key a line opens with, if any. Only a
// colon at end of line or before a space separates a key from its value, which
// is what keeps the one inside `https://…` from splitting a URL in half.
func splitManifestKey(node string) (key, sep, value string, ok bool) {
	quoted := false
	for i := 0; i < len(node); i++ {
		switch node[i] {
		case '"':
			quoted = !quoted
		case ':':
			if quoted {
				continue
			}
			if i+1 == len(node) {
				return node[:i], ":", "", true
			}
			if node[i+1] == ' ' {
				return node[:i], ": ", node[i+2:], true
			}
		}
	}
	return "", "", "", false
}

// highlightManifestValue leaves scalars in the terminal's own foreground so
// they read against the colored keys, and picks out the ${VAR} placeholders —
// the one part of a generated manifest that still needs the user to do
// something, and what the closing "export" hint refers to.
func highlightManifestValue(value string) string {
	// A JSON container opening in value position is syntax, same as the brace
	// that closes it further down.
	if value == "{" || value == "[" {
		return colorize(value, colMuted)
	}
	suffix := ""
	if strings.HasSuffix(value, ",") {
		suffix = colorize(",", colMuted)
		value = value[:len(value)-1]
	}
	return manifestEnvRef.ReplaceAllStringFunc(value, func(ref string) string {
		return colorize(ref, colWarning)
	}) + suffix
}

func isManifestPunctuation(node string) bool {
	switch node {
	case "{", "}", "[", "]", "},", "],", "{}", "[]":
		return true
	}
	return false
}

// printGenerateNextSteps closes every run with what to do with the manifest.
// The commands are complete and pasteable, so the path from "generated" to
// "running" is copy, paste, enter.
//
// The manifest declares credentials as ${VAR} placeholders, and the commands
// that consume it expand those from the environment before anything else — so
// the exports come first. (Notably `config create --secret` cannot substitute
// for them: expansion happens while the file is read, before that flag is
// applied.) On a terminal those commands prompt instead, but a printed export
// line is what makes the sequence work in a script too.
func printGenerateNextSteps(w io.Writer, a *generateAnswers, m *generatedManifest, path string) {
	stylingEnabled = detectStyling()

	ref := path
	if ref == "" {
		// Nothing was written, so the follow-ups can only name a file the user
		// still has to create; say so rather than printing a broken command.
		fmt.Fprintf(w, "\n%s\n", colorize("Save this manifest, then:", colMuted))
		ref = "agents.yaml"
	} else {
		fmt.Fprintf(w, "\n%s\n", colorize("Next steps", colMuted))
	}

	var needExport []string
	for _, slot := range a.Secrets {
		// A brokered slot is filled by the platform, and an inlined one is
		// already in the file.
		if slot.Inline || strings.HasPrefix(slot.Value, "oauth/") {
			continue
		}
		if strings.TrimSpace(os.Getenv(slot.Name)) == "" {
			needExport = append(needExport, slot.Name+"=…")
		}
	}
	if len(needExport) > 0 {
		fmt.Fprint(w, cardRow("set", "export "+strings.Join(needExport, " ")))
	}

	name := strings.TrimSpace(m.Name)
	if name == "" {
		name = "my-config"
	}
	fmt.Fprint(w, cardRow("reuse", fmt.Sprintf("%s config create --%s %s --%s %s", agentCLI, doctl.ArgAgentSpec, ref, doctl.ArgAgentName, name)))
	fmt.Fprint(w, cardRow("run", fmt.Sprintf("%s launch --%s %s", agentCLI, doctl.ArgAgentSpec, ref)))
}

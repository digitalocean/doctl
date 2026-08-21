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
	"regexp"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/digitalocean/doctl"
	"github.com/muesli/reflow/wordwrap"
	"golang.org/x/term"
	yaml "gopkg.in/yaml.v2"
)

// knownAgentAdapters is the accept-list from contracts/agent.v1alpha1.schema.json
// (spec.runtime.adapter). Kept in doctl so client-side validation fails fast with
// a clear field path instead of a generic API 400.
var knownAgentAdapters = map[string]struct{}{
	"claude-code":      {},
	"opencode":         {},
	"codex":            {},
	"cursor":           {},
	"hermes":           {},
	"codex-agentapi":   {},
	"custom":           {},
	"codex-cli":        {}, // deprecated alias
	"cursor-cli":       {}, // deprecated alias
	"openai-agents":    {}, // declared, not yet runnable
	"claude-agent-sdk": {},
	"crewai":           {},
	"langgraph":        {},
}

// reservedAgentEnvKeys mirrors harness-api agentspec.ReservedEnvKeys for the
// keys customers commonly hit. Full server list may be wider; unknown reserved
// keys are still rejected by the API.
var reservedAgentEnvKeys = map[string]struct{}{
	"TEAM_ID":                        {},
	"SESSION_ID":                     {},
	"SANDBOX_ID":                     {},
	"CONTROL_PLANE_CALLBACK_URL":     {},
	"SANDBOX_TEMPLATE":               {},
	"SANDBOX_AGENT_WORKDIR":          {},
	"SIZE_SLUG":                      {},
	"SANDBOX_IMAGE_ID":               {},
	"SANDBOX_OCI_REF":                {},
	"OHP_FORWARDER_ENABLED":          {},
	"OHP_FORWARDER_ENDPOINT":         {},
	"OHP_FORWARDER_AUTHORITY":        {},
	"HARNESS_CONFIG_ID":              {},
	"HARNESS_SKILLS":                 {},
	"PLANO_PERMISSION_POLICY_B64":    {},
	"PLANO_PERMISSION_DESCRIPTOR_DG": {},
}

// secretEnvKeyRE matches env KEYS that read like a credential holder (same
// heuristic as harness-api agentspec.secretEnvKeyRE).
var secretEnvKeyRE = regexp.MustCompile(
	`(?i)(^|[^A-Z0-9])(API_KEYS?|TOKENS?|SECRETS?|PASSWORDS?|PRIVATE_KEYS?|CREDENTIALS?)([^A-Z0-9]|$)`,
)

var credentialEnvKeySuffixRE = regexp.MustCompile(`(?i)(^|_)PAT$`)

var credentialEnvKeyNames = map[string]struct{}{
	"ANTHROPIC_KEY":     {},
	"OPENAI_KEY":        {},
	"HARNESS_KEY":       {},
	"AWS_ACCESS_KEY_ID": {},
	"APIKEY":            {},
}

// credentialPrefixes are issuer-assigned token prefixes; a literal env value
// starting with one is almost certainly a credential pasted into debug-readable
// guest env (harness-api agentspec.credentialPrefixes).
var credentialPrefixes = []string{
	"dop_v1_",
	"doo_v1_",
	"sk-",
	"ghp_",
	"gho_",
	"ghs_",
	"github_pat_",
	"glpat-",
	"xoxb-",
	"xoxp-",
	"AKIA",
}

// agentManifestValidation is the result of client-side agents.yaml checks.
type agentManifestValidation struct {
	Errors   []string
	Warnings []string
}

func (v *agentManifestValidation) ok() bool {
	return v == nil || len(v.Errors) == 0
}

func (v *agentManifestValidation) error() error {
	if v.ok() {
		return nil
	}
	return fmt.Errorf("manifest validation failed:\n  - %s", strings.Join(v.Errors, "\n  - "))
}

// validateAgentManifest performs client-side schema checks for flat and legacy
// envelope manifests. It catches the high-signal failures that otherwise become
// opaque create/session bugs (missing agent/adapter, credentials in env, reserved
// keys, MODEL vs HARNESS_INFERENCE_MODEL). Server-side agentspec remains
// authoritative for the full contract.
func validateAgentManifest(manifest []byte) *agentManifestValidation {
	out := &agentManifestValidation{}
	if len(bytesTrimSpace(manifest)) == 0 {
		out.Errors = append(out.Errors, "manifest is empty")
		return out
	}

	var doc map[string]any
	if err := yaml.Unmarshal(manifest, &doc); err != nil {
		out.Errors = append(out.Errors, fmt.Sprintf("invalid YAML: %v", err))
		return out
	}
	if len(doc) == 0 {
		out.Errors = append(out.Errors, "manifest is empty")
		return out
	}

	legacy := hasYAMLKey(doc, "apiVersion")
	if legacy {
		validateEnvelopeManifest(doc, out)
	} else {
		validateFlatManifest(doc, out)
	}

	adapter := manifestAdapter(doc, legacy)
	env, secretNames, envPath, secretsPath := extractManifestEnvAndSecrets(doc, legacy)
	validateManifestEnvAndSecrets(adapter, env, secretNames, envPath, secretsPath, out)
	return out
}

func manifestAdapter(doc map[string]any, legacy bool) string {
	if legacy {
		if spec, ok := yamlMap(doc["spec"]); ok {
			if runtime, ok := yamlMap(spec["runtime"]); ok {
				if adapter, ok := yamlString(runtime["adapter"]); ok {
					return strings.TrimSpace(adapter)
				}
			}
			if adapter, ok := yamlString(spec["adapter"]); ok {
				return strings.TrimSpace(adapter)
			}
		}
		return ""
	}
	agent, _ := yamlString(doc["agent"])
	return strings.TrimSpace(agent)
}

func bytesTrimSpace(b []byte) []byte {
	return []byte(strings.TrimSpace(string(b)))
}

func hasYAMLKey(doc map[string]any, key string) bool {
	_, ok := doc[key]
	return ok
}

func validateFlatManifest(doc map[string]any, out *agentManifestValidation) {
	agent, _ := yamlString(doc["agent"])
	agent = strings.TrimSpace(agent)
	if agent == "" {
		out.Errors = append(out.Errors, `flat manifest requires top-level "agent" (e.g. agent: opencode)`)
	} else {
		validateAdapter(agent, "agent", out)
	}

	if name, ok := yamlString(doc["name"]); ok && strings.TrimSpace(name) != "" {
		if err := validateHostedAgentIdentifier(strings.TrimSpace(name)); err != nil {
			out.Errors = append(out.Errors, fmt.Sprintf("name: %v", err))
		}
	}
}

func validateEnvelopeManifest(doc map[string]any, out *agentManifestValidation) {
	apiVersion, _ := yamlString(doc["apiVersion"])
	if apiVersion != "agents.digitalocean.com/v1alpha1" {
		out.Errors = append(out.Errors, fmt.Sprintf(`apiVersion must be "agents.digitalocean.com/v1alpha1" (got %q)`, apiVersion))
	}
	kind, _ := yamlString(doc["kind"])
	if kind != "Agent" {
		out.Errors = append(out.Errors, fmt.Sprintf(`kind must be "Agent" (got %q)`, kind))
	}

	spec, ok := yamlMap(doc["spec"])
	if !ok {
		out.Errors = append(out.Errors, `envelope manifest requires "spec"`)
		return
	}

	runtime, ok := yamlMap(spec["runtime"])
	if !ok {
		// Common footgun: adapter under spec instead of spec.runtime.
		if adapter, has := yamlString(spec["adapter"]); has && strings.TrimSpace(adapter) != "" {
			out.Errors = append(out.Errors, `spec.adapter is invalid; use spec.runtime.adapter`)
			validateAdapter(strings.TrimSpace(adapter), "spec.runtime.adapter", out)
			return
		}
		out.Errors = append(out.Errors, `spec.runtime is required`)
		return
	}
	adapter, _ := yamlString(runtime["adapter"])
	adapter = strings.TrimSpace(adapter)
	if adapter == "" {
		out.Errors = append(out.Errors, `spec.runtime.adapter is required`)
	} else {
		validateAdapter(adapter, "spec.runtime.adapter", out)
	}

	if meta, ok := yamlMap(doc["metadata"]); ok {
		if name, ok := yamlString(meta["name"]); ok && strings.TrimSpace(name) != "" {
			if err := validateHostedAgentIdentifier(strings.TrimSpace(name)); err != nil {
				out.Errors = append(out.Errors, fmt.Sprintf("metadata.name: %v", err))
			}
		}
	}
}

func validateAdapter(adapter, path string, out *agentManifestValidation) {
	if _, ok := knownAgentAdapters[adapter]; !ok {
		out.Errors = append(out.Errors, fmt.Sprintf("%s %q is not a known adapter (want claude-code, opencode, codex, cursor, hermes, codex-agentapi, custom, …)", path, adapter))
		return
	}
	switch adapter {
	case "codex-cli":
		out.Warnings = append(out.Warnings, fmt.Sprintf(`%s "codex-cli" is deprecated; use "codex"`, path))
	case "cursor-cli":
		out.Warnings = append(out.Warnings, fmt.Sprintf(`%s "cursor-cli" is deprecated; use "cursor"`, path))
	case "openai-agents", "claude-agent-sdk", "crewai", "langgraph":
		out.Warnings = append(out.Warnings, fmt.Sprintf("%s %q is declared but not yet supported for session create", path, adapter))
	}
}

func extractManifestEnvAndSecrets(doc map[string]any, legacy bool) (env map[string]string, secretNames []string, envPath, secretsPath string) {
	env = map[string]string{}
	var envRaw, secretsRaw any
	if legacy {
		envPath, secretsPath = "spec.env", "spec.secrets"
		if spec, ok := yamlMap(doc["spec"]); ok {
			envRaw = spec["env"]
			secretsRaw = spec["secrets"]
		}
	} else {
		envPath, secretsPath = "env", "secrets"
		envRaw = doc["env"]
		secretsRaw = doc["secrets"]
	}

	if m, ok := yamlStringMap(envRaw); ok {
		env = m
	}

	secretNames = secretSlotNames(secretsRaw)
	return env, secretNames, envPath, secretsPath
}

func secretSlotNames(secretsRaw any) []string {
	if secretsRaw == nil {
		return nil
	}
	// Flat shorthand: secrets: { OPENAI_API_KEY: ${OPENAI_API_KEY} }
	if m, ok := yamlStringMap(secretsRaw); ok {
		names := make([]string, 0, len(m))
		for k := range m {
			names = append(names, k)
		}
		sort.Strings(names)
		return names
	}
	// Envelope / list form: secrets: [{ name, source, value }]
	list, ok := yamlList(secretsRaw)
	if !ok {
		return nil
	}
	var names []string
	for _, item := range list {
		if m, ok := yamlMap(item); ok {
			if name, ok := yamlString(m["name"]); ok && name != "" {
				names = append(names, name)
			}
		}
	}
	sort.Strings(names)
	return names
}

func validateManifestEnvAndSecrets(adapter string, env map[string]string, secretNames []string, envPath, secretsPath string, out *agentManifestValidation) {
	slots := make(map[string]struct{}, len(secretNames))
	for _, n := range secretNames {
		slots[n] = struct{}{}
	}

	keys := make([]string, 0, len(env))
	for k := range env {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, key := range keys {
		val := env[key]
		ukey := strings.ToUpper(key)
		if _, reserved := reservedAgentEnvKeys[ukey]; reserved {
			out.Errors = append(out.Errors, fmt.Sprintf("%s.%s: reserved platform environment key", envPath, key))
			continue
		}
		switch {
		case hasStringKey(slots, key):
			out.Warnings = append(out.Warnings, fmt.Sprintf("%s.%s: matches a declared %s slot; supply the value through secrets, not %s", envPath, key, secretsPath, envPath))
		case hasCredentialPrefix(val):
			out.Warnings = append(out.Warnings, fmt.Sprintf("%s.%s: value looks like a credential; declare it under %s instead", envPath, key, secretsPath))
		case isCredentialLookingEnvKey(key) && !isEnvPlaceholder(val):
			out.Warnings = append(out.Warnings, fmt.Sprintf("%s.%s: name suggests a credential; declare it under %s instead (guest env is debug-readable and stored as-is)", envPath, key, secretsPath))
		}
	}

	validateAdapterModelEnv(adapter, env, envPath, out)
}

// validateAdapterModelEnv warns on known silent-failure env key footguns from
// production incidents (wrong model/key names still create a session, then the
// agent appears "not logged in" or never streams).
func validateAdapterModelEnv(adapter string, env map[string]string, envPath string, out *agentManifestValidation) {
	_, hasModel := env["MODEL"]
	_, hasHarnessModel := env["HARNESS_INFERENCE_MODEL"]
	_, hasHarnessBase := env["HARNESS_INFERENCE_BASE_URL"]
	_, hasHarnessKey := env["HARNESS_INFERENCE_API_KEY"]
	_, hasAnthropicModel := env["ANTHROPIC_MODEL"]
	_, hasAnthropicKey := env["ANTHROPIC_API_KEY"]

	// Incident follow-up: MODEL vs HARNESS_INFERENCE_MODEL produce different
	// session-create outcomes when harness inference env is partially set.
	if (hasHarnessBase || hasHarnessKey || hasHarnessModel) && hasModel && !hasHarnessModel {
		out.Warnings = append(out.Warnings, fmt.Sprintf("%s.MODEL: set alongside harness inference env vars, but %s.HARNESS_INFERENCE_MODEL is missing; prefer HARNESS_INFERENCE_MODEL for harness-routed inference", envPath, envPath))
	}
	if hasHarnessModel && hasModel && env["MODEL"] != env["HARNESS_INFERENCE_MODEL"] {
		out.Warnings = append(out.Warnings, fmt.Sprintf("%s.MODEL: value %q differs from %s.HARNESS_INFERENCE_MODEL (%q); sessions may pick different models depending on the runtime path", envPath, env["MODEL"], envPath, env["HARNESS_INFERENCE_MODEL"]))
	}

	if adapter == "claude-code" {
		if hasModel && !hasAnthropicModel {
			out.Warnings = append(out.Warnings, fmt.Sprintf("%s.MODEL: set for adapter claude-code, but %s.ANTHROPIC_MODEL is missing; claude-code expects ANTHROPIC_MODEL (sessions may fail with \"Not logged in\")", envPath, envPath))
		}
		if hasHarnessModel && !hasAnthropicModel {
			out.Warnings = append(out.Warnings, fmt.Sprintf("%s.HARNESS_INFERENCE_MODEL: set for adapter claude-code, but %s.ANTHROPIC_MODEL is missing; prefer ANTHROPIC_MODEL + ANTHROPIC_API_KEY (or declare the key under secrets)", envPath, envPath))
		}
		if hasAnthropicKey && !hasAnthropicModel && (hasModel || hasHarnessModel) {
			out.Warnings = append(out.Warnings, fmt.Sprintf("%s.ANTHROPIC_API_KEY: set without %s.ANTHROPIC_MODEL; set ANTHROPIC_MODEL explicitly for claude-code", envPath, envPath))
		}
	}
}

func isCredentialLookingEnvKey(key string) bool {
	if secretEnvKeyRE.MatchString(key) {
		return true
	}
	if _, ok := credentialEnvKeyNames[strings.ToUpper(key)]; ok {
		return true
	}
	return credentialEnvKeySuffixRE.MatchString(key)
}

func hasCredentialPrefix(value string) bool {
	for _, prefix := range credentialPrefixes {
		if strings.HasPrefix(value, prefix) {
			return true
		}
	}
	return false
}

// isEnvPlaceholder reports ${VAR}-style references that expand locally before
// create. Credential-looking keys with only a placeholder are normal (doctl
// prompts for them); literal values are the problem.
func isEnvPlaceholder(value string) bool {
	value = strings.TrimSpace(value)
	return len(value) > 3 && strings.HasPrefix(value, "${") && strings.HasSuffix(value, "}")
}

func hasStringKey(set map[string]struct{}, key string) bool {
	_, ok := set[key]
	return ok
}

func reportAgentManifestValidation(v *agentManifestValidation) error {
	if v == nil {
		return nil
	}
	printAgentManifestWarnings(os.Stderr, v.Warnings)
	return v.error()
}

// reportDurableAgentManifestValidation is stricter for Agent Config create:
// credential/slot issues that are create-time warnings for sessions become
// errors, matching harness-api's durable-config credential contract.
func reportDurableAgentManifestValidation(v *agentManifestValidation) error {
	if v == nil {
		return nil
	}
	var kept []string
	for _, w := range v.Warnings {
		if strings.Contains(w, "matches a declared") || strings.Contains(w, "looks like a credential") || strings.Contains(w, "name suggests a credential") {
			v.Errors = append(v.Errors, w)
			continue
		}
		kept = append(kept, w)
	}
	v.Warnings = kept
	printAgentManifestWarnings(os.Stderr, v.Warnings)
	return v.error()
}

// RunAgentsValidate checks an agents.yaml / JSON manifest client-side without
// creating a session.
func RunAgentsValidate(c *CmdConfig) error {
	if Output != "json" {
		stylingEnabled = detectStyling()
	}
	specPath, err := c.Doit.GetString(c.NS, doctl.ArgAgentSpec)
	if err != nil {
		return err
	}
	if strings.TrimSpace(specPath) == "" {
		return fmt.Errorf("--%s is required", doctl.ArgAgentSpec)
	}
	raw, err := readManifest(os.Stdin, specPath)
	if err != nil {
		return err
	}
	v := validateAgentManifest(raw)
	printAgentManifestValidation(c.Out, v)
	if !v.ok() {
		return ErrExitSilently
	}
	return nil
}

// printAgentManifestValidation renders a styled validate result that stays
// readable in narrow terminals (no bordered card — those shatter when long
// lines wrap mid-word).
func printAgentManifestValidation(w io.Writer, v *agentManifestValidation) {
	if v == nil {
		v = &agentManifestValidation{}
	}

	fmt.Fprintln(w)
	switch {
	case !v.ok():
		title := "Manifest is invalid"
		if n := len(v.Errors); n == 1 {
			title = "Manifest is invalid · 1 error"
		} else if n > 1 {
			title = fmt.Sprintf("Manifest is invalid · %d errors", n)
		}
		fmt.Fprintf(w, "%s %s\n", colorize("✗", colError), boldColor(title, colError))
	case len(v.Warnings) > 0:
		title := fmt.Sprintf("Manifest looks valid · %d warning", len(v.Warnings))
		if len(v.Warnings) != 1 {
			title += "s"
		}
		fmt.Fprintf(w, "%s %s\n", colorize("⚠", colWarning), boldColor(title, colWarning))
	default:
		fmt.Fprintf(w, "%s %s\n", colorize("✓", colSuccess), boldColor("Manifest looks valid", colSuccess))
	}

	if len(v.Errors) > 0 {
		fmt.Fprintln(w)
		for _, e := range v.Errors {
			fmt.Fprint(w, validationIssueBlock("Error", e, colError))
		}
	}
	if len(v.Warnings) > 0 {
		fmt.Fprintln(w)
		for _, warnMsg := range v.Warnings {
			fmt.Fprint(w, validationIssueBlock("Warning", warnMsg, colWarning))
		}
	}
	if v.ok() && len(v.Warnings) > 0 {
		fmt.Fprintf(w, "  %s  %s\n", colorize("→", colMuted), colorize("Review before create", colMuted))
	}
}

// printAgentManifestWarnings prints create-path warnings (stderr) with a yellow
// Warning label so they stand out next to lifecycle progress lines.
func printAgentManifestWarnings(w io.Writer, warnings []string) {
	if len(warnings) == 0 {
		return
	}
	prev := stylingEnabled
	if !stylingEnabled {
		stylingEnabled = detectStyling()
	}
	defer func() { stylingEnabled = prev }()

	fmt.Fprintln(w)
	for _, msg := range warnings {
		fmt.Fprint(w, validationIssueBlock("Warning", msg, colWarning))
	}
}

// validationIssueBlock formats a Warning/Error with the keyword in color, an
// optional highlighted field path, and word-wrapped body text so narrow
// terminals never mid-split tokens inside a shattered border.
func validationIssueBlock(kind, msg string, kindColor lipgloss.Color) string {
	label := boldColor(kind, kindColor)
	msg = strings.TrimSpace(msg)

	const hang = "           " // aligns under text after "  Warning  "
	width := agentTermWidth()
	bodyWidth := width - len(hang)
	if bodyWidth < 24 {
		bodyWidth = 24
	}

	var b strings.Builder
	if path, rest, ok := splitValidationPath(msg); ok {
		fmt.Fprintf(&b, "  %s  %s\n", label, colorize(path, colHighlight))
		for _, line := range wrapDisplayText(rest, bodyWidth) {
			fmt.Fprintf(&b, "%s%s\n", hang, line)
		}
		return b.String()
	}

	lines := wrapDisplayText(msg, bodyWidth)
	if len(lines) == 0 {
		fmt.Fprintf(&b, "  %s\n", label)
		return b.String()
	}
	fmt.Fprintf(&b, "  %s  %s\n", label, lines[0])
	for _, line := range lines[1:] {
		fmt.Fprintf(&b, "%s%s\n", hang, line)
	}
	return b.String()
}

// wrapDisplayText word-wraps plain text to width, breaking only on spaces when
// possible so identifiers like ANTHROPIC_MODEL stay intact until unavoidable.
func wrapDisplayText(s string, width int) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	if width < 8 {
		width = 8
	}
	wrapped := wordwrap.String(s, width)
	lines := strings.Split(wrapped, "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		if line == "" {
			continue
		}
		out = append(out, line)
	}
	if len(out) == 0 {
		return []string{s}
	}
	return out
}

// agentTermWidth returns the current terminal width, with a sane fallback.
func agentTermWidth() int {
	if w, _, err := term.GetSize(int(os.Stdout.Fd())); err == nil && w >= 40 {
		return w
	}
	return 80
}

// splitValidationPath pulls a leading field path (e.g. spec.env.MODEL) off a
// validation message so it can be highlighted separately.
func splitValidationPath(msg string) (path, rest string, ok bool) {
	// "spec.env.MODEL: rest" or "agent "not-a-real-adapter" is not…"
	if i := strings.Index(msg, ": "); i > 0 {
		candidate := msg[:i]
		if looksLikeFieldPath(candidate) {
			return candidate, msg[i+2:], true
		}
	}
	return "", "", false
}

func looksLikeFieldPath(s string) bool {
	if s == "" || strings.ContainsAny(s, " \t") {
		return false
	}
	switch {
	case strings.HasPrefix(s, "spec."), strings.HasPrefix(s, "metadata."),
		strings.HasPrefix(s, "env."), strings.HasPrefix(s, "secrets."),
		s == "agent", s == "name", s == "apiVersion", s == "kind", s == "spec.runtime",
		strings.HasPrefix(s, "spec.runtime"):
		return true
	default:
		return strings.Contains(s, ".")
	}
}

func yamlMap(v any) (map[string]any, bool) {
	switch m := v.(type) {
	case map[string]any:
		return m, true
	case map[any]any:
		out := make(map[string]any, len(m))
		for k, val := range m {
			ks, ok := k.(string)
			if !ok {
				return nil, false
			}
			out[ks] = val
		}
		return out, true
	default:
		return nil, false
	}
}

func yamlList(v any) ([]any, bool) {
	switch t := v.(type) {
	case []any:
		return t, true
	default:
		return nil, false
	}
}

func yamlString(v any) (string, bool) {
	s, ok := v.(string)
	return s, ok
}

func yamlStringMap(v any) (map[string]string, bool) {
	m, ok := yamlMap(v)
	if !ok {
		return nil, false
	}
	out := make(map[string]string, len(m))
	for k, val := range m {
		switch t := val.(type) {
		case string:
			out[k] = t
		case nil:
			out[k] = ""
		default:
			out[k] = fmt.Sprint(t)
		}
	}
	return out, true
}

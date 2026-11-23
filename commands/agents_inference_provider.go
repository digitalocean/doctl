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
	"strings"

	"github.com/digitalocean/doctl/commands/charm/input"
	yaml "gopkg.in/yaml.v2"
)

// defaultDOSIModel is the shared DigitalOcean Serverless Inference catalogue
// id used when a create path has no model/provider yet and the harness has no
// more specific default (see defaultModelForHarness).
const defaultDOSIModel = "kimi-k3.1-pro"

// defaultClaudeCodeModel is the DO Inference catalogue id used when
// claude-code falls back to DigitalOcean Serverless Inference (blank /
// missing ANTHROPIC_API_KEY, or a provider-less claude-code manifest).
const defaultClaudeCodeModel = "anthropic-claude-4.5-sonnet"

// defaultCodexModel is the DO Inference catalogue id used when coding codex
// falls back to DigitalOcean Serverless Inference (blank / missing
// OPENAI_API_KEY, or a provider-less codex manifest). Not used for
// codex-agentapi, which still requires a real OpenAI key.
const defaultCodexModel = "openai-gpt-5.5"

// defaultMinimaxCodeModel is the DO Inference catalogue id used when
// minimax-code falls back to DigitalOcean Serverless Inference (blank /
// missing MINIMAX_API_KEY, or a provider-less minimax-code manifest). There
// is no MiniMax-branded catalogue id on DO SI, so this matches the shared
// default.
const defaultMinimaxCodeModel = defaultDOSIModel

// defaultModelForHarness returns the DigitalOcean Inference catalogue id a
// harness gets when no model is supplied. Harness-specific catalogue defaults
// win. OpenCode returns "" so create keeps OpenCode's free-model path unless
// the user passes --model / HARNESS_INFERENCE_MODEL. Unknown adapters fall
// back to defaultDOSIModel.
func defaultModelForHarness(adapter string) string {
	switch strings.ToLower(strings.TrimSpace(adapter)) {
	case openHandsAgentName, "open-hands":
		return defaultOpenHandsModel
	case hermesAgentName:
		return defaultHermesModel
	case claudeCodeAgentName, "claude":
		return defaultClaudeCodeModel
	case codexAgentName:
		return defaultCodexModel
	case minimaxCodeAgentName, "minimax":
		return defaultMinimaxCodeModel
	case "opencode", "open-code":
		return ""
	default:
		return defaultDOSIModel
	}
}

// defaultModelForBlankableVendorKey is the model named in the leave-blank
// vendor-key prompt, matching the harness that key usually belongs to.
func defaultModelForBlankableVendorKey(name string) string {
	switch name {
	case anthropicAPIKeyEnv:
		return defaultClaudeCodeModel
	case openAIAPIKeyEnv:
		return defaultCodexModel
	case minimaxAPIKeyEnv:
		return defaultMinimaxCodeModel
	default:
		return defaultDOSIModel
	}
}

// blankableVendorKeyEnvs are guest inference keys that may be left blank to
// fall back to DigitalOcean Serverless Inference. codex-agentapi still
// requires OPENAI_API_KEY for sandbox minting and is skipped separately.
var blankableVendorKeyEnvs = []string{anthropicAPIKeyEnv, openAIAPIKeyEnv, minimaxAPIKeyEnv}

// inferenceCredentialEnvs are credentials that mean the manifest already has
// an inference provider (BYOK or vendor), so the catch-all must not inject.
var inferenceCredentialEnvs = []string{
	harnessInferenceAPIKeyEnv,
	openAIAPIKeyEnv,
	anthropicAPIKeyEnv,
	minimaxAPIKeyEnv,
}

// promptOptionalVendorKey asks for a vendor API key but allows an empty answer
// to mean "use DigitalOcean Serverless Inference". Tests replace this.
var promptOptionalVendorKey = defaultPromptOptionalVendorKey

func defaultPromptOptionalVendorKey(name string) (string, error) {
	if !canPromptForEnv() {
		return "", nil
	}

	stylingEnabled = detectStyling()
	fmt.Fprintf(os.Stderr, "%s %s\n",
		colorize("•", colHighlight),
		fmt.Sprintf("%s is not set — enter it, or leave blank to use DigitalOcean Serverless Inference (%s)",
			boldColor(name, colHighlight), defaultModelForBlankableVendorKey(name)))

	opts := []input.Option{}
	if isSecretEnvVarName(name) {
		opts = append(opts, input.WithHidden())
	}
	prompt := input.New(fmt.Sprintf("Enter %s (blank = DO Serverless Inference): ", name), opts...)
	val, err := prompt.Prompt()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(val), nil
}

// hasInferenceProvider reports whether the manifest already names a model or
// supplies a real inference/vendor credential.
func hasInferenceProvider(manifest []byte) bool {
	var doc map[string]any
	if err := yaml.Unmarshal(manifest, &doc); err != nil || doc == nil {
		return false
	}
	_, legacy := doc["apiVersion"]
	env, _, _, _ := extractManifestEnvAndSecrets(doc, legacy)
	if strings.TrimSpace(env[harnessInferenceModelEnv]) != "" {
		return true
	}
	for _, name := range inferenceCredentialEnvs {
		if hasProvidedCredential(manifest, env, name) {
			return true
		}
	}
	return false
}

// hasInferenceCredentialSlot reports whether the manifest still declares a
// vendor/harness API key slot (including ${VAR} placeholders or concrete
// values). Used so the provider-less catch-all does not inject a DO SI model
// on top of a MiniMax/Anthropic/OpenAI path that expand or the guest will use.
func hasInferenceCredentialSlot(manifest []byte) bool {
	for _, name := range inferenceCredentialEnvs {
		if manifestReferencesVendorKey(manifest, name) {
			return true
		}
	}
	return false
}

func vendorKeySatisfied(manifest []byte, name string) bool {
	var doc map[string]any
	if err := yaml.Unmarshal(manifest, &doc); err != nil || doc == nil {
		return strings.TrimSpace(os.Getenv(name)) != ""
	}
	_, legacy := doc["apiVersion"]
	env, _, _, _ := extractManifestEnvAndSecrets(doc, legacy)
	if hasProvidedCredential(manifest, env, name) {
		return true
	}
	return strings.TrimSpace(os.Getenv(name)) != ""
}

func manifestReferencesVendorKey(manifest []byte, name string) bool {
	vals := manifestSecretValues(manifest)
	if v, ok := vals[name]; ok {
		if strings.TrimSpace(v) == "" || isEnvPlaceholder(v) || v == redactedSecretValue {
			return true
		}
		// Concrete value already in the slot — still a "reference" to that key,
		// but resolveInferenceProvider treats it as satisfied via vendorKeySatisfied.
		return true
	}
	var doc map[string]any
	if err := yaml.Unmarshal(manifest, &doc); err != nil || doc == nil {
		return false
	}
	_, legacy := doc["apiVersion"]
	env, _, _, _ := extractManifestEnvAndSecrets(doc, legacy)
	if v, ok := env[name]; ok {
		if strings.TrimSpace(v) == "" || isEnvPlaceholder(v) {
			return true
		}
		return true
	}
	return false
}

// injectDefaultDOSIModel sets env.HARNESS_INFERENCE_MODEL from
// defaultModelForHarness when missing. Never overwrites an existing model.
func injectDefaultDOSIModel(manifest []byte) ([]byte, error) {
	var doc map[string]any
	if err := yaml.Unmarshal(manifest, &doc); err != nil {
		return nil, fmt.Errorf("parsing manifest to inject %s: %w", harnessInferenceModelEnv, err)
	}
	if doc == nil {
		doc = map[string]any{}
	}

	container := doc
	legacy := false
	if _, ok := doc["apiVersion"]; ok {
		spec, ok := yamlMap(doc["spec"])
		if !ok {
			spec = map[string]any{}
		}
		container = spec
		legacy = true
	}

	env, ok := yamlStringMap(container["env"])
	if !ok {
		env = map[string]string{}
	}
	if strings.TrimSpace(env[harnessInferenceModelEnv]) != "" {
		return manifest, nil
	}

	adapter := ""
	if parsed, err := parseAgentManifest(manifest); err == nil && parsed != nil {
		adapter = parsed.adapter()
	}
	model := defaultModelForHarness(adapter)
	if model == "" {
		// OpenCode (and any future harness with no DO SI default) leaves the
		// guest on its own free / built-in models.
		return manifest, nil
	}
	env[harnessInferenceModelEnv] = model

	outEnv := make(map[string]any, len(env))
	for k, v := range env {
		outEnv[k] = v
	}
	container["env"] = outEnv
	if legacy {
		doc["spec"] = container
	}
	raw, err := yaml.Marshal(doc)
	if err != nil {
		return nil, fmt.Errorf("injecting %s: %w", harnessInferenceModelEnv, err)
	}
	return raw, nil
}

// stripVendorKeyFromManifest removes a vendor key from secrets and env so the
// blank→DO SI path does not leave a required ${VAR} behind for expand.
func stripVendorKeyFromManifest(manifest []byte, name string) ([]byte, error) {
	var doc map[string]any
	if err := yaml.Unmarshal(manifest, &doc); err != nil {
		return nil, fmt.Errorf("parsing manifest to strip %s: %w", name, err)
	}
	if doc == nil {
		return manifest, nil
	}

	container := doc
	legacy := false
	if _, ok := doc["apiVersion"]; ok {
		spec, ok := yamlMap(doc["spec"])
		if !ok {
			return manifest, nil
		}
		container = spec
		legacy = true
	}

	changed := false
	if secretsRaw, ok := container["secrets"]; ok && secretsRaw != nil {
		if m, ok := yamlMap(secretsRaw); ok {
			if _, exists := m[name]; exists {
				delete(m, name)
				changed = true
				if len(m) == 0 {
					delete(container, "secrets")
				} else {
					container["secrets"] = m
				}
			}
		} else if list, ok := yamlList(secretsRaw); ok {
			filtered := make([]any, 0, len(list))
			for _, item := range list {
				entry, ok := yamlMap(item)
				if !ok {
					filtered = append(filtered, item)
					continue
				}
				n, _ := entry["name"].(string)
				if n == name {
					changed = true
					continue
				}
				filtered = append(filtered, item)
			}
			if changed {
				if len(filtered) == 0 {
					delete(container, "secrets")
				} else {
					container["secrets"] = filtered
				}
			}
		}
	}

	if env, ok := yamlStringMap(container["env"]); ok {
		if _, exists := env[name]; exists {
			delete(env, name)
			changed = true
			if len(env) == 0 {
				delete(container, "env")
			} else {
				outEnv := make(map[string]any, len(env))
				for k, v := range env {
					outEnv[k] = v
				}
				container["env"] = outEnv
			}
		}
	}

	if !changed {
		return manifest, nil
	}
	if legacy {
		doc["spec"] = container
	}
	raw, err := yaml.Marshal(doc)
	if err != nil {
		return nil, fmt.Errorf("stripping %s: %w", name, err)
	}
	return raw, nil
}

func isBareSandboxManifest(manifest []byte) bool {
	return manifestAgentIs(manifest, bareSandboxAgent) ||
		manifestAgentIs(manifest, bareSandboxAgentAlt)
}

// harnessInferenceModelFromManifest returns the trimmed HARNESS_INFERENCE_MODEL
// value from a flat or legacy agentspec document, or "" when unset.
func harnessInferenceModelFromManifest(manifest []byte) string {
	var doc map[string]any
	if err := yaml.Unmarshal(manifest, &doc); err != nil || doc == nil {
		return ""
	}
	_, legacy := doc["apiVersion"]
	env, _, _, _ := extractManifestEnvAndSecrets(doc, legacy)
	return strings.TrimSpace(env[harnessInferenceModelEnv])
}

// maybePrintDefaultDOSINotice tells the user when create resolved onto a
// harness catalogue default model with DigitalOcean Serverless Inference
// (managed key). Silent when a vendor/BYOK key is present or the model was
// overridden away from the harness default. Writes to stderr when w is nil.
func maybePrintDefaultDOSINotice(w io.Writer, manifest []byte) {
	if !needsManagedModelAccessKey(manifest) {
		return
	}
	model := harnessInferenceModelFromManifest(manifest)
	if model == "" {
		return
	}
	adapter := ""
	if parsed, err := parseAgentManifest(manifest); err == nil && parsed != nil {
		adapter = parsed.adapter()
	}
	if model != defaultModelForHarness(adapter) {
		return
	}
	if w == nil {
		w = os.Stderr
	}
	printDefaultDOSINotice(w, model)
}

func printDefaultDOSINotice(w io.Writer, model string) {
	stylingEnabled = detectStyling()
	fmt.Fprintf(w, "%s %s\n",
		colorize("•", colHighlight),
		fmt.Sprintf("Using default model %s with DigitalOcean Serverless Inference (billed to your DigitalOcean account).",
			boldColor(model, colHighlight)))
	fmt.Fprintf(w, "  %s\n",
		colorize("To use your own OpenAI, Anthropic, or MiniMax key instead, create a new environment with OPENAI_API_KEY, ANTHROPIC_API_KEY, or MINIMAX_API_KEY.", colMuted))
}

// resolveInferenceProvider rewrites a create-time manifest so every non-sandbox
// session/config has an inference provider. allowPrompt enables the interactive
// "leave blank → DO SI" path for Anthropic/OpenAI/MiniMax guest keys; dry-run
// passes false so unbound keys rewrite to DO SI without blocking on a TTY.
//
// Callers must run this before needsManagedModelAccessKey so an injected
// HARNESS_INFERENCE_MODEL is visible to the mint predicate.
func resolveInferenceProvider(manifest []byte, allowPrompt bool) ([]byte, error) {
	if len(manifest) == 0 || isBareSandboxManifest(manifest) {
		return manifest, nil
	}

	// OpenAI Agents sandboxes require a real OPENAI_API_KEY for ENV_ID minting
	// and must not be rewritten onto DO SI (including the provider-less catch-all:
	// their secret slot is CODEX_API_KEY, not OPENAI_API_KEY).
	if doc, err := parseAgentManifest(manifest); err == nil && doc != nil {
		if isOpenAISandboxAdapter(doc.adapter()) {
			return manifest, nil
		}
	}

	var err error
	for _, key := range blankableVendorKeyEnvs {
		if !manifestReferencesVendorKey(manifest, key) {
			continue
		}
		if vendorKeySatisfied(manifest, key) {
			continue
		}

		if allowPrompt && canPromptForEnv() {
			val, perr := promptOptionalVendorKey(key)
			if perr != nil {
				return nil, perr
			}
			if val != "" {
				if err := os.Setenv(key, val); err != nil {
					return nil, fmt.Errorf("setting %s: %w", key, err)
				}
				continue
			}
		}

		manifest, err = stripVendorKeyFromManifest(manifest, key)
		if err != nil {
			return nil, err
		}
		manifest, err = injectDefaultDOSIModel(manifest)
		if err != nil {
			return nil, err
		}
	}

	if !hasInferenceProvider(manifest) && !hasInferenceCredentialSlot(manifest) {
		manifest, err = injectDefaultDOSIModel(manifest)
		if err != nil {
			return nil, err
		}
	}
	return manifest, nil
}

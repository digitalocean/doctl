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
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/digitalocean/doctl"
	yaml "gopkg.in/yaml.v2"
)

// The flat agent manifest, as authored for `harness-runtime config generate`.
//
// Field order follows the canonical annotated contract (harness-api
// contracts/agent.flat.yaml) section by section — identity, agent, workspace,
// environment, secrets, model, sizing, egress, tools, permissions, execution
// environment, skills, memory/budget, mode — so a generated file reads like
// the documented example rather than like a Go struct dump. Nested blocks are
// typed rather than map[string]any for the same reason: a map would emit its
// keys alphabetically (`auth` before `name`), which reads backwards.
//
// Only fields the command can populate are modeled. `mounts` is deliberately
// absent: it is accepted-but-unenforced server-side and requires `env: false`
// secret slots, which is beyond what a generator should guess at.
type generatedManifest struct {
	Name        string            `yaml:"name,omitempty" json:"name,omitempty"`
	Description string            `yaml:"description,omitempty" json:"description,omitempty"`
	Labels      map[string]string `yaml:"labels,omitempty" json:"labels,omitempty"`

	Agent      string         `yaml:"agent" json:"agent"`
	Image      string         `yaml:"image,omitempty" json:"image,omitempty"`
	Entrypoint []string       `yaml:"entrypoint,omitempty" json:"entrypoint,omitempty"`
	Config     map[string]any `yaml:"config,omitempty" json:"config,omitempty"`

	Repos []string `yaml:"repos,omitempty" json:"repos,omitempty"`

	Env     map[string]string `yaml:"env,omitempty" json:"env,omitempty"`
	Secrets map[string]string `yaml:"secrets,omitempty" json:"secrets,omitempty"`

	Model any `yaml:"model,omitempty" json:"model,omitempty"`

	Size        string `yaml:"size,omitempty" json:"size,omitempty"`
	IdleTimeout string `yaml:"idle_timeout,omitempty" json:"idle_timeout,omitempty"`
	MaxLifetime string `yaml:"max_lifetime,omitempty" json:"max_lifetime,omitempty"`

	Egress []string `yaml:"egress,omitempty" json:"egress,omitempty"`

	Tools       []any                 `yaml:"tools,omitempty" json:"tools,omitempty"`
	Permissions *generatedPermissions `yaml:"permissions,omitempty" json:"permissions,omitempty"`

	Template            string `yaml:"template,omitempty" json:"template,omitempty"`
	PersistentWorkspace *bool  `yaml:"persistent_workspace,omitempty" json:"persistent_workspace,omitempty"`

	Skills []generatedSkill `yaml:"skills,omitempty" json:"skills,omitempty"`

	Mode    string            `yaml:"mode,omitempty" json:"mode,omitempty"`
	Serving *generatedServing `yaml:"serving,omitempty" json:"serving,omitempty"`
}

// generatedModel is the object form of `model`. The string shorthand is used
// whenever only a default model is set.
type generatedModel struct {
	Default string `yaml:"default" json:"default"`
	Routing string `yaml:"routing,omitempty" json:"routing,omitempty"`
}

// generatedMCPServer is an inline (non-DO) MCP server attachment.
type generatedMCPServer struct {
	Name  string            `yaml:"name" json:"name"`
	URL   string            `yaml:"url" json:"url"`
	Tools []string          `yaml:"tools,omitempty" json:"tools,omitempty"`
	Auth  *generatedMCPAuth `yaml:"auth,omitempty" json:"auth,omitempty"`
}

// generatedMCPAuth points an inline MCP server at a declared secrets slot. The
// contract has no plaintext token field by design, so neither does this.
type generatedMCPAuth struct {
	Secret string `yaml:"secret" json:"secret"`
}

type generatedPermissions struct {
	Default string                    `yaml:"default,omitempty" json:"default,omitempty"`
	Rules   []generatedPermissionRule `yaml:"rules,omitempty" json:"rules,omitempty"`
}

type generatedPermissionRule struct {
	Tool   string            `yaml:"tool" json:"tool"`
	Match  map[string]string `yaml:"match,omitempty" json:"match,omitempty"`
	Action string            `yaml:"action" json:"action"`
}

type generatedSkill struct {
	Name         string `yaml:"name" json:"name"`
	Description  string `yaml:"description" json:"description"`
	Instructions string `yaml:"instructions" json:"instructions"`
}

type generatedServing struct {
	Min               *int                  `yaml:"min,omitempty" json:"min,omitempty"`
	Max               *int                  `yaml:"max,omitempty" json:"max,omitempty"`
	TargetConcurrency *int                  `yaml:"target_concurrency,omitempty" json:"target_concurrency,omitempty"`
	ScaleToZero       *generatedScaleToZero `yaml:"scale_to_zero,omitempty" json:"scale_to_zero,omitempty"`
}

type generatedScaleToZero struct {
	Idle string `yaml:"idle" json:"idle"`
}

// --- harness catalog --------------------------------------------------------

// generateHarness describes one selectable harness: its manifest id, the name
// shown in the picker, the sandbox template the server derives when `template`
// is omitted, how it gets inference, and any credential slots it needs beyond
// that.
type generateHarness struct {
	ID        string
	Label     string
	Template  string
	Inference *harnessInference
	// Secrets are credential slots unrelated to inference. Only the
	// OpenAI-hosted loop has one, since OpenAI runs the agent rather than the
	// sandbox calling a model endpoint itself.
	Secrets []string
}

// Where a harness gets its model. This is not a manifest field: the platform
// has no inference section, so the choice materializes as guest environment
// plus one credential slot. Two mutually exclusive groups exist, and the
// in-VM runtime prefers the native one whenever its key is present — so a
// manifest must commit to one group or the other, never mix them.
const (
	// inferenceProviderNative uses the agent's own upstream API (Anthropic for
	// claude-code, OpenAI for codex/opencode, Cursor for cursor) with that
	// vendor's own environment variables, and lets the agent pick its default
	// model. It is the default when nothing is prompted, being the only option
	// that needs no further answers.
	inferenceProviderNative = "native"
	// inferenceProviderDO routes inference through DigitalOcean Serverless
	// Inference, and is what the wizard recommends: one model access key, and
	// the platform already allows its host. It has no default model, so it
	// costs one more answer — which is why it leads the picker but not the
	// non-prompting path.
	inferenceProviderDO = "do"
	// inferenceProviderCustom points the harness at any other
	// OpenAI-compatible endpoint. Requires a URL, a model, and egress for the
	// host, none of which can be guessed.
	inferenceProviderCustom = "custom"
)

// The harness-owned inference wiring. The manifest is the only source of these
// values — nothing is injected control-plane side — and setting the key here is
// what activates endpoint routing at all.
const (
	harnessInferenceBaseURLEnv = "HARNESS_INFERENCE_BASE_URL"
	harnessInferenceAPIKeyEnv  = "HARNESS_INFERENCE_API_KEY"
	harnessInferenceModelEnv   = "HARNESS_INFERENCE_MODEL"

	// doInferenceBaseURL is the DigitalOcean Serverless Inference endpoint. The
	// /v1 suffix is part of the documented base URL; adapters that need it
	// stripped (claude-code, whose SDK appends its own resource path) strip it
	// themselves in the guest.
	doInferenceBaseURL = "https://inference.do-ai.run/v1"
	// doInferenceModelsCmd lists the model ids that endpoint accepts. They are
	// gateway slugs (anthropic-claude-4.5-sonnet), not upstream vendor ids
	// (claude-sonnet-4-5), which is the single easiest thing to get wrong here.
	doInferenceModelsCmd = "doctl serverless-inference models list"
)

// harnessInference records the environment a harness reads for inference, so
// the generator can write the right variable names for the chosen provider
// instead of a field the platform does not read.
type harnessInference struct {
	// NativeKeyEnv is the credential the agent binary looks for on its own.
	NativeKeyEnv string
	// NativeBaseURLEnv is that binary's own endpoint override. Empty when the
	// agent talks to a fixed backend.
	NativeBaseURLEnv string
	// NativeModelEnv is that binary's own model selector. Empty when the agent
	// has none and a model can only be pinned through HARNESS_INFERENCE_MODEL.
	NativeModelEnv string
	// ExtraModelEnv is a second model variable the agent binary itself reads
	// (claude-code's ANTHROPIC_MODEL). It is set for every provider, which also
	// keeps the manifest clear of doctl's own model-consistency warnings.
	ExtraModelEnv string

	// ProviderLabel names the upstream vendor in the picker.
	ProviderLabel string
	// NativeModelHint and DOModelHint are example ids in each id space.
	NativeModelHint string
	DOModelHint     string

	// NativeOnly marks an agent whose backend cannot be redirected, so
	// offering it an endpoint choice would be a lie.
	NativeOnly bool
}

// modelEnvNames lists the environment variables a model id should be written
// to for the given provider. Endpoint routing pins the model through the
// harness variable; the native path prefers the agent's own.
func (h *harnessInference) modelEnvNames(provider string) []string {
	if h == nil {
		return nil
	}
	var names []string
	if provider == inferenceProviderNative && h.NativeModelEnv != "" {
		names = append(names, h.NativeModelEnv)
	} else {
		names = append(names, harnessInferenceModelEnv)
	}
	if h.ExtraModelEnv != "" {
		names = append(names, h.ExtraModelEnv)
	}
	return names
}

// keyEnvName is the credential slot the provider needs.
func (h *harnessInference) keyEnvName(provider string) string {
	if h == nil {
		return ""
	}
	if provider == inferenceProviderNative {
		return h.NativeKeyEnv
	}
	return harnessInferenceAPIKeyEnv
}

// generateHarnessCatalog is the set offered interactively: the canonical
// supported adapter ids from the agent-manifest contract. Adapters outside this
// list (hermes, the deprecated aliases, the declared-but-unsupported
// frameworks) remain reachable through --harness, which validates against the
// same accept-list `validate` uses, so there is one source of truth for what a
// manifest may name.
var generateHarnessCatalog = []generateHarness{
	{
		ID:       "claude-code",
		Label:    "Claude Code",
		Template: "coding-claude-code",
		Inference: &harnessInference{
			NativeKeyEnv:     "ANTHROPIC_API_KEY",
			NativeBaseURLEnv: "ANTHROPIC_BASE_URL",
			// claude-code has no model env of its own; the runtime pins its
			// --model from HARNESS_INFERENCE_MODEL. ANTHROPIC_MODEL is read by
			// the CLI itself, so both are set.
			ExtraModelEnv:   "ANTHROPIC_MODEL",
			ProviderLabel:   "Anthropic",
			NativeModelHint: "claude-sonnet-4-5",
			DOModelHint:     "anthropic-claude-4.5-sonnet",
		},
	},
	{
		ID:       "opencode",
		Label:    "OpenCode",
		Template: "coding-opencode",
		Inference: &harnessInference{
			NativeKeyEnv:     "OPENAI_API_KEY",
			NativeBaseURLEnv: "OPENAI_BASE_URL",
			NativeModelEnv:   "MODEL",
			ProviderLabel:    "OpenAI",
			NativeModelHint:  "gpt-5-codex",
			DOModelHint:      "deepseek-v4-pro",
		},
	},
	{
		ID:       "codex",
		Label:    "Codex",
		Template: "coding-codex",
		Inference: &harnessInference{
			NativeKeyEnv:     "OPENAI_API_KEY",
			NativeBaseURLEnv: "OPENAI_BASE_URL",
			NativeModelEnv:   "MODEL",
			ProviderLabel:    "OpenAI",
			NativeModelHint:  "gpt-5-codex",
			DOModelHint:      "openai-gpt-5.3-codex",
		},
	},
	{
		ID:       "cursor",
		Label:    "Cursor CLI",
		Template: "coding-cursor",
		Inference: &harnessInference{
			NativeKeyEnv:  "CURSOR_API_KEY",
			ProviderLabel: "Cursor",
			// Cursor's agent talks to Cursor's own backend, which no endpoint
			// override can redirect.
			NativeOnly: true,
		},
	},
	{
		ID:       "codex-agentapi",
		Label:    "Codex (OpenAI-hosted loop)",
		Template: "coding-codex",
		// No inference wiring at all: the loop runs at OpenAI, so the sandbox
		// never calls a model endpoint and the harness variables are ignored.
		Secrets: []string{"OPENAI_API_KEY"},
	},
	{
		ID:       "custom",
		Label:    "Custom container",
		Template: "",
	},
}

// generateHarnessAliases are the spellings accepted in addition to the
// canonical ids, mirroring the courtesy `create --harness` extends.
var generateHarnessAliases = map[string]string{
	"claude":         "claude-code",
	"claude-code":    "claude-code",
	"open-code":      "opencode",
	"opencode":       "opencode",
	"codex":          "codex",
	"cursor":         "cursor",
	"cursor-agent":   "cursor",
	"openai-codex":   "codex-agentapi",
	"codex-agentapi": "codex-agentapi",
	"custom":         "custom",
}

// generateFallbackSizes is used when the live size catalog is unreachable —
// `config generate` must work offline and unauthenticated, so a failed lookup
// degrades to the documented slugs instead of erroring.
var generateFallbackSizes = []string{
	"mv-1vcpu-2gb",
	"mv-2vcpu-4gb",
	"mv-8vcpu-16gb",
	"mv-16vcpu-32gb",
}

// generateDefaultToolsPreset is the `--tools-preset default` bundle: the DO
// Action Gateway with a small, safe selection rather than its whole catalog.
var generateDefaultToolsPreset = []string{"web_search", "execute_code", "toolbelt:read-only"}

const (
	generateToolsPresetNone    = "none"
	generateToolsPresetDefault = "default"

	// generateDefaultPermission is the recommended disposition for an unmatched
	// tool call: pause and ask rather than run or refuse silently.
	generateDefaultPermission = "ask"

	// generateGitHubOAuthSlot is the brokered GitHub credential slot added by
	// --github-access.
	generateGitHubOAuthSlot  = "GITHUB_TOKEN"
	generateGitHubOAuthValue = "oauth/github"

	// doCatalogToolPrefix is the reserved DO-owned tool namespace.
	doCatalogToolPrefix = "do."
	doActionsCatalogRef = "do.actions"
)

// generateGitHubEgress are the hosts a repo-attached session needs to reach.
var generateGitHubEgress = []string{"github.com", "api.github.com"}

func lookupGenerateHarness(id string) (generateHarness, bool) {
	for _, h := range generateHarnessCatalog {
		if h.ID == id {
			return h, true
		}
	}
	return generateHarness{}, false
}

// resolveGenerateHarness canonicalizes a --harness value and checks it against
// the same adapter accept-list `validate` enforces, so `generate` can never
// emit an `agent:` value that doctl itself would reject.
//
// It is deliberately separate from resolveHarnessAgent, which serves
// `create`/`launch` and intentionally covers only the three adapters those
// commands build manifests for.
func resolveGenerateHarness(harness string) (string, error) {
	key := strings.ToLower(strings.TrimSpace(harness))
	if key == "" {
		return "", fmt.Errorf("--%s is required (one of %s)", doctl.ArgAgentHarness, generateHarnessIDList())
	}
	if canonical, ok := generateHarnessAliases[key]; ok {
		return canonical, nil
	}
	if _, known := knownAgentAdapters[key]; known {
		return key, nil
	}
	return "", fmt.Errorf("unsupported --%s %q; supported values: %s", doctl.ArgAgentHarness, harness, generateHarnessIDList())
}

func generateHarnessIDList() string {
	ids := make([]string, 0, len(generateHarnessCatalog))
	for _, h := range generateHarnessCatalog {
		ids = append(ids, h.ID)
	}
	return strings.Join(ids, ", ")
}

// derivedTemplateFor reports the template the server picks when `template` is
// omitted. Shown in the review card so "auto" is never a mystery.
func derivedTemplateFor(agent string) string {
	if h, ok := lookupGenerateHarness(agent); ok {
		return h.Template
	}
	return ""
}

// --- answers --------------------------------------------------------------

// generateSecretSlot is one declared credential slot. Value carries what the
// manifest will say: a ${VAR} placeholder, an `oauth/<provider>` reference, or
// (only via --inline-secret) a literal.
type generateSecretSlot struct {
	Name   string
	Value  string
	Inline bool
}

// generateAnswers is the fully-resolved intent behind one `config generate`
// run, whether it arrived from flags, from wizard prompts, or from a mix of the
// two. Both paths converge here, so the manifest builder has exactly one input
// shape and the two front ends cannot drift.
type generateAnswers struct {
	Name        string
	Description string
	Labels      map[string]string

	Harness    string
	Image      string
	Entrypoint []string
	Prompt     string

	Repos        []string
	GitHubAccess bool

	Env     map[string]string
	Secrets []generateSecretSlot

	// InferenceProvider is one of the inferenceProvider* constants, and
	// InferenceURL is the endpoint when it is inferenceProviderCustom.
	InferenceProvider string
	InferenceURL      string

	Model        string
	ModelRouting string

	Size                string
	Template            string
	PersistentWorkspace *bool
	IdleTimeout         string
	MaxLifetime         string

	Egress []string

	ToolsPreset string
	Tools       []string
	MCPServers  []generatedMCPServer

	PermissionDefault string
	PermissionRules   []generatedPermissionRule

	Skills []generatedSkill

	Mode    string
	Serving *generatedServing
}

// secretNames lists declared slot names in declaration order, for the review
// card and the next-step hints.
func (a *generateAnswers) secretNames() []string {
	names := make([]string, 0, len(a.Secrets))
	for _, s := range a.Secrets {
		names = append(names, s.Name)
	}
	return names
}

func (a *generateAnswers) hasSecret(name string) bool {
	for _, s := range a.Secrets {
		if strings.EqualFold(s.Name, name) {
			return true
		}
	}
	return false
}

// addSecretSlot appends a slot, ignoring a duplicate declaration so the
// convenience flags (--github-access) and explicit ones (--secret) can overlap
// without producing a doubled key.
func (a *generateAnswers) addSecretSlot(slot generateSecretSlot) {
	if a.hasSecret(slot.Name) {
		return
	}
	a.Secrets = append(a.Secrets, slot)
}

func (a *generateAnswers) addEgress(hosts ...string) {
	for _, host := range hosts {
		host = strings.TrimSpace(host)
		if host == "" {
			continue
		}
		found := false
		for _, existing := range a.Egress {
			if strings.EqualFold(existing, host) {
				found = true
				break
			}
		}
		if !found {
			a.Egress = append(a.Egress, host)
		}
	}
}

// --- manifest construction --------------------------------------------------

// buildGeneratedManifest turns resolved answers into the manifest document.
// It applies only the derivations a generator can make safely: credential slots
// implied by a chosen harness, egress implied by an attached repo or inline MCP
// host, and the opaque Codex config block. Everything else is written exactly
// as answered, and anything unanswered is omitted so the server's own defaults
// apply.
func buildGeneratedManifest(a *generateAnswers) (*generatedManifest, error) {
	if a == nil {
		return nil, fmt.Errorf("no answers to build a manifest from")
	}
	agent, err := resolveGenerateHarness(a.Harness)
	if err != nil {
		return nil, err
	}

	m := &generatedManifest{
		Agent:               agent,
		Name:                strings.TrimSpace(a.Name),
		Description:         strings.TrimSpace(a.Description),
		Labels:              nonEmptyStringMap(a.Labels),
		Repos:               a.Repos,
		Env:                 nonEmptyStringMap(a.Env),
		Size:                strings.TrimSpace(a.Size),
		IdleTimeout:         strings.TrimSpace(a.IdleTimeout),
		MaxLifetime:         strings.TrimSpace(a.MaxLifetime),
		Template:            strings.TrimSpace(a.Template),
		PersistentWorkspace: a.PersistentWorkspace,
		Skills:              a.Skills,
	}

	if m.Name != "" {
		if err := validateHostedAgentIdentifier(m.Name); err != nil {
			return nil, err
		}
	}

	if agent == "custom" {
		if strings.TrimSpace(a.Image) == "" {
			return nil, fmt.Errorf("--%s is required with --%s custom (a digest-pinned container image)", doctl.ArgAgentImage, doctl.ArgAgentHarness)
		}
		m.Image = strings.TrimSpace(a.Image)
		m.Entrypoint = a.Entrypoint
	} else if strings.TrimSpace(a.Image) != "" || len(a.Entrypoint) > 0 {
		return nil, fmt.Errorf("--%s and --%s are only valid with --%s custom", doctl.ArgAgentImage, doctl.ArgAgentEntrypoint, doctl.ArgAgentHarness)
	}

	// The OpenAI-hosted Codex loop is the one adapter with a required opaque
	// config block; reuse the same builder `create --harness codex` uses so the
	// two commands cannot describe the same runtime differently.
	if isOpenAISandboxAdapter(agent) {
		m.Config = defaultCodexRunConfig(a.Prompt)
	} else if strings.TrimSpace(a.Prompt) != "" {
		return nil, fmt.Errorf("--%s only applies to --%s codex-agentapi; send a first prompt with `%s create --%s` instead", doctl.ArgAgentTriggerPrompt, doctl.ArgAgentHarness, agentCLI, doctl.ArgAgentTriggerPrompt)
	}

	if err := validateGeneratedDurations(m); err != nil {
		return nil, err
	}

	secrets := make(map[string]string, len(a.Secrets))
	for _, slot := range a.Secrets {
		name := strings.TrimSpace(slot.Name)
		if name == "" {
			continue
		}
		if err := validateSecretSlotName(name); err != nil {
			return nil, err
		}
		secrets[name] = slot.Value
	}
	if len(secrets) > 0 {
		m.Secrets = secrets
	}

	if model := strings.TrimSpace(a.Model); model != "" {
		if routing := strings.TrimSpace(a.ModelRouting); routing != "" {
			m.Model = generatedModel{Default: model, Routing: routing}
		} else {
			m.Model = model
		}
	} else if strings.TrimSpace(a.ModelRouting) != "" {
		return nil, fmt.Errorf("--%s needs --%s (routing selects how a named model is served)", doctl.ArgAgentModelRouting, doctl.ArgAgentModel)
	}

	tools, err := buildGeneratedTools(a)
	if err != nil {
		return nil, err
	}
	m.Tools = tools

	if perms := buildGeneratedPermissions(a); perms != nil {
		m.Permissions = perms
	}

	// Egress is assembled last so hosts implied by other sections (repos, inline
	// MCP servers) join the explicitly requested ones.
	egress := append([]string{}, a.Egress...)
	m.Egress = dedupeHosts(egress)
	if len(m.Egress) == 0 {
		m.Egress = nil
	}

	if mode := strings.TrimSpace(a.Mode); mode != "" {
		if mode != "interactive" && mode != "served" {
			return nil, fmt.Errorf("--%s must be interactive or served (got %q)", doctl.ArgAgentMode, mode)
		}
		m.Mode = mode
	}
	if a.Serving != nil {
		if m.Mode != "served" {
			return nil, fmt.Errorf("serving flags require --%s served", doctl.ArgAgentMode)
		}
		m.Serving = a.Serving
	} else if m.Mode == "served" {
		return nil, fmt.Errorf("--%s served requires the serving flags (--%s, --%s, --%s)", doctl.ArgAgentMode, doctl.ArgAgentServingMin, doctl.ArgAgentServingMax, doctl.ArgAgentServingConcurrency)
	}

	return m, nil
}

// applyGenerateEnterDefaults answers every still-unanswered question with the
// value the wizard offers as its default.
//
// This is what keeps --no-interactive (and a redirected stdout) aligned with
// walking the wizard and pressing Enter at every step, rather than a second,
// barer set of defaults nobody documented. The wizard reads its defaults from
// the same places, so the two paths cannot drift apart.
//
// Questions whose default is "nothing" — repos, MCP servers, skills, a model
// override — are simply left unset here, exactly as Enter would leave them.
// The one deliberate difference is that nothing here looks at the working
// directory: the wizard offers the checked-out repo as a default, which would
// make a scripted run depend on where it happened to be invoked.
func applyGenerateEnterDefaults(a *generateAnswers) {
	if a.Name == "" {
		a.Name = suggestGenerateName(a.Harness)
	}
	// Native inference is the default precisely because it is the only provider
	// that needs no follow-up answer: the agent uses its own vendor and its own
	// default model. The credential slot it implies is added by
	// applyGenerateDerivations, which runs for the wizard too.
	if a.InferenceProvider == "" {
		a.InferenceProvider = inferenceProviderNative
	}
	if a.ToolsPreset == "" && len(a.Tools) == 0 && len(a.MCPServers) == 0 {
		a.ToolsPreset = generateToolsPresetDefault
	}
	if a.PermissionDefault == "" {
		a.PermissionDefault = generateDefaultPermission
	}
	if len(a.Repos) > 0 && !a.GitHubAccess {
		a.GitHubAccess = true
	}
}

// applyGenerateInference turns the chosen provider into the guest environment,
// credential slot, and egress it needs.
//
// The platform has no inference field, so this is the whole mechanism: which
// variables get written decides which of the runtime's two credential groups
// activates. The groups are never mixed — a native key present anywhere makes
// the runtime ignore the endpoint variables entirely, which would silently send
// traffic to the vendor instead of the endpoint the user asked for.
func applyGenerateInference(a *generateAnswers) error {
	harness, ok := lookupGenerateHarness(a.Harness)
	if !ok || harness.Inference == nil {
		// Nothing to wire: either an unlisted adapter, a custom image whose
		// inference is its own business, or the OpenAI-hosted loop.
		if strings.TrimSpace(a.InferenceProvider) != "" && strings.TrimSpace(a.InferenceProvider) != inferenceProviderNative {
			return fmt.Errorf("--%s does not apply to --%s %s: it does not call a model endpoint from the sandbox", doctl.ArgAgentInference, doctl.ArgAgentHarness, a.Harness)
		}
		return nil
	}
	inf := harness.Inference

	provider := strings.TrimSpace(a.InferenceProvider)
	if provider == "" {
		provider = inferenceProviderNative
	}
	switch provider {
	case inferenceProviderNative, inferenceProviderDO, inferenceProviderCustom:
	default:
		return fmt.Errorf("--%s must be %s, %s, or %s (got %q)", doctl.ArgAgentInference, inferenceProviderNative, inferenceProviderDO, inferenceProviderCustom, provider)
	}
	if inf.NativeOnly && provider != inferenceProviderNative {
		return fmt.Errorf("--%s %s only supports %s inference: its agent talks to %s's own backend, which an endpoint override cannot redirect", doctl.ArgAgentHarness, a.Harness, inferenceProviderNative, inf.ProviderLabel)
	}
	a.InferenceProvider = provider

	if a.Env == nil {
		a.Env = map[string]string{}
	}

	switch provider {
	case inferenceProviderDO:
		if a.Model == "" {
			return fmt.Errorf("--%s %s needs --%s: Serverless Inference has no default model. List the ids it accepts with `%s`", doctl.ArgAgentInference, inferenceProviderDO, doctl.ArgAgentModel, doInferenceModelsCmd)
		}
		a.Env[harnessInferenceBaseURLEnv] = doInferenceBaseURL
		a.addSecretSlot(generateSecretSlot{Name: harnessInferenceAPIKeyEnv, Value: secretPlaceholder(harnessInferenceAPIKeyEnv)})
		// The platform allows the Serverless Inference host itself, so no
		// egress entry is needed here.
	case inferenceProviderCustom:
		url := strings.TrimSpace(a.InferenceURL)
		if url == "" {
			// Error text avoids naming the vendor: doctl's error beautifier
			// rewrites any message mentioning it into an unrelated
			// "check your API key" card (agents_errors.go).
			return fmt.Errorf("--%s %s needs --%s (the endpoint's base URL, usually ending in /v1)", doctl.ArgAgentInference, inferenceProviderCustom, doctl.ArgAgentInferenceURL)
		}
		if !strings.HasPrefix(url, "https://") {
			return fmt.Errorf("--%s must be an https:// URL (got %q)", doctl.ArgAgentInferenceURL, url)
		}
		if a.Model == "" {
			return fmt.Errorf("--%s %s needs --%s: doctl cannot know which models your endpoint serves", doctl.ArgAgentInference, inferenceProviderCustom, doctl.ArgAgentModel)
		}
		a.Env[harnessInferenceBaseURLEnv] = url
		a.addSecretSlot(generateSecretSlot{Name: harnessInferenceAPIKeyEnv, Value: secretPlaceholder(harnessInferenceAPIKeyEnv)})
		// Unlike the DO endpoint, an arbitrary host is not allowed by default.
		if host := egressHostFor(url); host != "" {
			a.addEgress(host)
		}
	case inferenceProviderNative:
		if key := inf.NativeKeyEnv; key != "" {
			a.addSecretSlot(generateSecretSlot{Name: key, Value: secretPlaceholder(key)})
		}
	}

	// The model id goes to the variables this harness actually reads. The
	// top-level `model:` field is written too (see buildGeneratedManifest), but
	// it is accepted-not-enforced today and selects nothing on its own.
	if a.Model != "" {
		for _, name := range inf.modelEnvNames(provider) {
			a.Env[name] = a.Model
		}
	}
	if len(a.Env) == 0 {
		a.Env = nil
	}
	return nil
}

// applyGenerateDerivations fills in what a chosen harness, provider, and
// attached repo imply, before the manifest is built. It runs after both the
// flag pass and the wizard, so a value the user set explicitly always wins, and
// it is idempotent so running it twice — once to make the review card truthful,
// once before writing — cannot double anything up.
func applyGenerateDerivations(a *generateAnswers) error {
	if err := applyGenerateInference(a); err != nil {
		return err
	}
	// Credential slots a harness needs for something other than inference.
	for _, name := range suggestedSecretsFor(a.Harness) {
		a.addSecretSlot(generateSecretSlot{Name: name, Value: secretPlaceholder(name)})
	}
	if a.GitHubAccess {
		a.addSecretSlot(generateSecretSlot{Name: generateGitHubOAuthSlot, Value: generateGitHubOAuthValue})
		a.addEgress(generateGitHubEgress...)
	}
	// An inline MCP host is not auto-allowed by the platform; adding it here
	// spares the user a create-time warning for a host they clearly intend to
	// reach.
	for _, srv := range a.MCPServers {
		if host := egressHostFor(srv.URL); host != "" {
			a.addEgress(host)
		}
	}
	return nil
}

// suggestedSecretsFor lists the credential slots a harness needs for something
// other than inference; inference slots come from applyGenerateInference.
func suggestedSecretsFor(agent string) []string {
	if h, ok := lookupGenerateHarness(agent); ok {
		return h.Secrets
	}
	return nil
}

func buildGeneratedTools(a *generateAnswers) ([]any, error) {
	var tools []any

	catalog := append([]string{}, a.Tools...)
	if strings.EqualFold(strings.TrimSpace(a.ToolsPreset), generateToolsPresetDefault) && len(catalog) == 0 {
		catalog = append(catalog, doActionsCatalogRef+":"+strings.Join(generateDefaultToolsPreset, ","))
	}

	// Selections for the same catalog server merge into one entry: the contract
	// allows a server once, and two entries for do.actions would be rejected.
	order := make([]string, 0, len(catalog))
	selections := make(map[string][]string, len(catalog))
	for _, raw := range catalog {
		ref, sel, err := parseCatalogToolRef(raw)
		if err != nil {
			return nil, err
		}
		if _, seen := selections[ref]; !seen {
			order = append(order, ref)
		}
		selections[ref] = append(selections[ref], sel...)
	}
	for _, ref := range order {
		sel := dedupeStrings(selections[ref])
		if len(sel) == 0 {
			tools = append(tools, ref)
			continue
		}
		tools = append(tools, map[string][]string{ref: sel})
	}

	for _, srv := range a.MCPServers {
		if err := validateInlineMCPServer(srv, a); err != nil {
			return nil, err
		}
		tools = append(tools, srv)
	}

	return tools, nil
}

// parseCatalogToolRef splits `do.actions` or `do.actions:tool1,tool2` into the
// server ref and its selection.
func parseCatalogToolRef(raw string) (string, []string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", nil, fmt.Errorf("--%s cannot be empty", doctl.ArgAgentTool)
	}
	ref, rest, hasSel := strings.Cut(raw, ":")
	ref = strings.TrimSpace(ref)
	if !strings.HasPrefix(ref, doCatalogToolPrefix) {
		return "", nil, fmt.Errorf("--%s %q must name a DO catalog server in the reserved do.* namespace (e.g. %s); attach any other MCP server with --%s", doctl.ArgAgentTool, raw, doActionsCatalogRef, doctl.ArgAgentMCPServer)
	}
	if !hasSel {
		return ref, nil, nil
	}
	// A toolbelt reference carries its own colon (toolbelt:read-only), so the
	// selection is re-split on commas rather than assumed to be a single name.
	var sel []string
	for _, part := range strings.Split(rest, ",") {
		part = strings.TrimSpace(part)
		if part != "" {
			sel = append(sel, part)
		}
	}
	if len(sel) == 0 {
		return "", nil, fmt.Errorf("--%s %q has an empty selection; drop the colon to attach every tool the server advertises", doctl.ArgAgentTool, raw)
	}
	return ref, sel, nil
}

func validateInlineMCPServer(srv generatedMCPServer, a *generateAnswers) error {
	name := strings.TrimSpace(srv.Name)
	if name == "" {
		return fmt.Errorf("--%s needs a server name (NAME=URL)", doctl.ArgAgentMCPServer)
	}
	if strings.HasPrefix(name, doCatalogToolPrefix) {
		return fmt.Errorf("MCP server %q cannot start with %q: that namespace is reserved for DO catalog refs", name, doCatalogToolPrefix)
	}
	if !strings.HasPrefix(srv.URL, "https://") {
		return fmt.Errorf("MCP server %q must have an https:// URL (got %q)", name, srv.URL)
	}
	if srv.Auth != nil {
		if !a.hasSecret(srv.Auth.Secret) {
			return fmt.Errorf("MCP server %q authenticates with secret slot %q, which is not declared; add --%s %s", name, srv.Auth.Secret, doctl.ArgAgentSecret, srv.Auth.Secret)
		}
	}
	return nil
}

func buildGeneratedPermissions(a *generateAnswers) *generatedPermissions {
	def := strings.TrimSpace(a.PermissionDefault)
	if def == "" && len(a.PermissionRules) == 0 {
		return nil
	}
	return &generatedPermissions{Default: def, Rules: a.PermissionRules}
}

// --- parsing helpers --------------------------------------------------------

// generateDurationRE mirrors the manifest contract's duration grammar: integer
// hour/minute/second components, never a bare second count.
var generateDurationRE = regexp.MustCompile(`^([0-9]+h)?([0-9]+m)?([0-9]+s)?$`)

func validateGenerateDuration(flag, value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	if !generateDurationRE.MatchString(value) || value == "" {
		return fmt.Errorf("--%s %q must be a duration of integer hours/minutes/seconds (e.g. 10m, 90s, 1h30m, 8h)", flag, value)
	}
	return nil
}

func validateGeneratedDurations(m *generatedManifest) error {
	if err := validateGenerateDuration(doctl.ArgAgentIdleTimeout, m.IdleTimeout); err != nil {
		return err
	}
	return validateGenerateDuration(doctl.ArgAgentMaxLifetime, m.MaxLifetime)
}

// secretSlotNameRE mirrors the contract: a slot name is the guest env var the
// credential materializes as.
var secretSlotNameRE = regexp.MustCompile(`^[A-Z][A-Z0-9_]*$`)

func validateSecretSlotName(name string) error {
	if !secretSlotNameRE.MatchString(name) {
		return fmt.Errorf("secret slot %q must be an upper-case environment variable name (A-Z, digits, underscores)", name)
	}
	if _, reserved := reservedAgentEnvKeys[name]; reserved {
		return fmt.Errorf("secret slot %q is a reserved platform key", name)
	}
	return nil
}

// secretPlaceholder is the value written for a declared slot: a ${VAR}
// reference doctl expands at create time, so the generated file stays
// committable and the credential never lands on disk here.
func secretPlaceholder(name string) string {
	return "${" + name + "}"
}

// parseGenerateKeyValues parses repeatable KEY=VALUE flags without resolving
// @file / - indirection, for flags whose values are plain config rather than
// credentials.
func parseGenerateKeyValues(flag string, pairs []string) (map[string]string, error) {
	if len(pairs) == 0 {
		return nil, nil
	}
	out := make(map[string]string, len(pairs))
	for _, pair := range pairs {
		key, value, err := parseKeyValueLine(pair)
		if err != nil {
			return nil, fmt.Errorf("--%s: %w", flag, err)
		}
		key = strings.TrimSpace(key)
		if _, dup := out[key]; dup {
			return nil, fmt.Errorf("--%s: duplicate key %q", flag, key)
		}
		out[key] = value
	}
	return out, nil
}

// parseGeneratePermissionRules turns TOOL[:match] flag values into rules. The
// optional match is the bash-style `command` matcher, the only matcher the
// contract exemplifies.
func parseGeneratePermissionRules(flag, action string, values []string) ([]generatedPermissionRule, error) {
	var rules []generatedPermissionRule
	for _, raw := range values {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		tool, match, hasMatch := strings.Cut(raw, ":")
		tool = strings.TrimSpace(tool)
		if tool == "" {
			return nil, fmt.Errorf("--%s %q needs a tool name (e.g. bash, file.write, do.actions/web_search)", flag, raw)
		}
		rule := generatedPermissionRule{Tool: tool, Action: action}
		// A server-qualified MCP target keeps its own colon (do.actions/toolbelt:read-only),
		// so only treat the tail as a matcher when the head is not itself qualified.
		if hasMatch && strings.TrimSpace(match) != "" && !strings.Contains(tool, "/") {
			rule.Match = map[string]string{"command": strings.TrimSpace(match)}
		} else if hasMatch && strings.Contains(tool, "/") {
			rule.Tool = raw
		}
		rules = append(rules, rule)
	}
	return rules, nil
}

// parseGenerateMCPServers assembles inline MCP servers from the three flags
// that describe them, keyed by server name.
func parseGenerateMCPServers(servers, toolSels, authSecrets []string) ([]generatedMCPServer, error) {
	if len(servers) == 0 {
		if len(toolSels) > 0 || len(authSecrets) > 0 {
			return nil, fmt.Errorf("--%s and --%s need a matching --%s NAME=URL", doctl.ArgAgentMCPTools, doctl.ArgAgentMCPAuthSecret, doctl.ArgAgentMCPServer)
		}
		return nil, nil
	}

	order := make([]string, 0, len(servers))
	byName := make(map[string]*generatedMCPServer, len(servers))
	for _, raw := range servers {
		name, url, err := parseKeyValueLine(raw)
		if err != nil {
			return nil, fmt.Errorf("--%s: %w (expected NAME=URL)", doctl.ArgAgentMCPServer, err)
		}
		name = strings.TrimSpace(name)
		if _, dup := byName[name]; dup {
			return nil, fmt.Errorf("--%s: duplicate server %q", doctl.ArgAgentMCPServer, name)
		}
		byName[name] = &generatedMCPServer{Name: name, URL: strings.TrimSpace(url)}
		order = append(order, name)
	}

	for _, raw := range toolSels {
		name, list, found := strings.Cut(strings.TrimSpace(raw), ":")
		name = strings.TrimSpace(name)
		srv, ok := byName[name]
		if !found || !ok {
			return nil, fmt.Errorf("--%s %q must be NAME:tool[,tool] naming a server declared with --%s", doctl.ArgAgentMCPTools, raw, doctl.ArgAgentMCPServer)
		}
		for _, tool := range strings.Split(list, ",") {
			if tool = strings.TrimSpace(tool); tool != "" {
				srv.Tools = append(srv.Tools, tool)
			}
		}
	}

	for _, raw := range authSecrets {
		name, slot, err := parseKeyValueLine(raw)
		if err != nil {
			return nil, fmt.Errorf("--%s: %w (expected NAME=SECRET_SLOT)", doctl.ArgAgentMCPAuthSecret, err)
		}
		srv, ok := byName[strings.TrimSpace(name)]
		if !ok {
			return nil, fmt.Errorf("--%s %q names a server not declared with --%s", doctl.ArgAgentMCPAuthSecret, raw, doctl.ArgAgentMCPServer)
		}
		srv.Auth = &generatedMCPAuth{Secret: strings.TrimSpace(slot)}
	}

	out := make([]generatedMCPServer, 0, len(order))
	for _, name := range order {
		out = append(out, *byName[name])
	}
	return out, nil
}

// parseGenerateSkills reads each NAME=path skill file into an inline skill. The
// description is the file's first non-blank line unless overridden, matching how
// a SKILL.md leads with what it is for.
func parseGenerateSkills(skills, descriptions []string) ([]generatedSkill, error) {
	if len(skills) == 0 {
		if len(descriptions) > 0 {
			return nil, fmt.Errorf("--%s needs a matching --%s NAME=PATH", doctl.ArgAgentSkillDescription, doctl.ArgAgentSkill)
		}
		return nil, nil
	}

	overrides, err := parseGenerateKeyValues(doctl.ArgAgentSkillDescription, descriptions)
	if err != nil {
		return nil, err
	}

	out := make([]generatedSkill, 0, len(skills))
	seen := make(map[string]struct{}, len(skills))
	for _, raw := range skills {
		name, path, err := parseKeyValueLine(raw)
		if err != nil {
			return nil, fmt.Errorf("--%s: %w (expected NAME=PATH)", doctl.ArgAgentSkill, err)
		}
		name = strings.TrimSpace(name)
		if !skillNameRE.MatchString(name) {
			return nil, fmt.Errorf("--%s: skill name %q must be lowercase letters, digits, and single hyphens", doctl.ArgAgentSkill, name)
		}
		if _, dup := seen[name]; dup {
			return nil, fmt.Errorf("--%s: duplicate skill %q", doctl.ArgAgentSkill, name)
		}
		seen[name] = struct{}{}

		body, err := os.ReadFile(filepath.Clean(strings.TrimSpace(path)))
		if err != nil {
			return nil, fmt.Errorf("--%s %s: %w", doctl.ArgAgentSkill, name, err)
		}
		instructions := strings.TrimSpace(string(body))
		if instructions == "" {
			return nil, fmt.Errorf("--%s %s: %s is empty", doctl.ArgAgentSkill, name, path)
		}
		description := overrides[name]
		if strings.TrimSpace(description) == "" {
			description = firstMeaningfulLine(instructions)
		}
		if strings.TrimSpace(description) == "" {
			return nil, fmt.Errorf("--%s %s: no description found; add --%s %s=...", doctl.ArgAgentSkill, name, doctl.ArgAgentSkillDescription, name)
		}
		out = append(out, generatedSkill{Name: name, Description: description, Instructions: instructions})
	}
	return out, nil
}

// firstMeaningfulLine returns the first non-blank line with markdown heading
// punctuation stripped, so `# Release checklist` becomes a usable description.
func firstMeaningfulLine(s string) string {
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(strings.TrimLeft(strings.TrimSpace(line), "#"))
		if line != "" {
			return strings.TrimSpace(line)
		}
	}
	return ""
}

// egressHostFor reduces a URL the manifest points at — an MCP server, an
// inference endpoint — to the bare host an egress entry needs.
func egressHostFor(rawURL string) string {
	trimmed := strings.TrimPrefix(strings.TrimSpace(rawURL), "https://")
	host, _, _ := strings.Cut(trimmed, "/")
	host, _, _ = strings.Cut(host, ":")
	return strings.TrimSpace(host)
}

func dedupeStrings(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, s := range in {
		if _, dup := seen[s]; dup {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}

func dedupeHosts(in []string) []string {
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, host := range in {
		host = strings.TrimSpace(host)
		if host == "" {
			continue
		}
		key := strings.ToLower(host)
		if _, dup := seen[key]; dup {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, host)
	}
	return out
}

func nonEmptyStringMap(m map[string]string) map[string]string {
	if len(m) == 0 {
		return nil
	}
	return m
}

// sortedKeys is used where a stable presentation order is needed for display;
// the encoders sort map keys themselves.
func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// --- encoding ---------------------------------------------------------------

const (
	generateFormatYAML = "yaml"
	generateFormatJSON = "json"
)

// encodeGeneratedManifest renders the manifest in the requested format. YAML is
// the default because that is what `config create --spec` and the contract's own
// examples use; JSON is offered for programmatic consumers that would otherwise
// need a YAML parser.
func encodeGeneratedManifest(m *generatedManifest, format string) ([]byte, error) {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "", generateFormatYAML, "yml":
		out, err := yaml.Marshal(m)
		if err != nil {
			return nil, fmt.Errorf("encoding manifest: %w", err)
		}
		return out, nil
	case generateFormatJSON:
		out, err := json.MarshalIndent(m, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("encoding manifest: %w", err)
		}
		return append(out, '\n'), nil
	default:
		return nil, fmt.Errorf("--%s must be %s or %s (got %q)", doctl.ArgAgentManifestFormat, generateFormatYAML, generateFormatJSON, format)
	}
}

/*
Copyright 2026 The Doctl Authors All rights reserved.
Licensed under the License, Version 2.0 (the "License");
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

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/commands/charm/input"
)

// DigitalOcean's own inference, used when the caller named no provider key.
//
// The three HARNESS_INFERENCE_* names are the contract harness-api and the
// in-guest adapters agree on: OHR's inference resolver prefers an adapter's
// native key (OPENAI_API_KEY, ANTHROPIC_API_KEY, …) and only falls back to this
// group, which is exactly the precedence we want — supplying a provider key
// later overrides this without the manifest having to change.
const (
	harnessInferenceAPIKeyEnv  = "HARNESS_INFERENCE_API_KEY"
	harnessInferenceModelEnv   = "HARNESS_INFERENCE_MODEL"
	harnessInferenceBaseURLEnv = "HARNESS_INFERENCE_BASE_URL"

	// defaultDOInferenceModel is the model a keyless session is pointed at.
	defaultDOInferenceModel = "kimi-k3"

	// defaultDOInferenceBaseURL is the public serverless-inference endpoint.
	// harness-api overwrites this with a VPC-internal trusted endpoint in the
	// environments that have one, so setting it here only matters where the
	// platform has no opinion — but leaving it unset there would let the
	// adapter fall back to its vendor default and send a DO key to OpenAI.
	defaultDOInferenceBaseURL = "https://inference.do-ai.run/v1"

	// modelAccessKeyDocsURL documents creating the key. The API that used to
	// mint these (POST /v2/gen-ai/models/api_keys) is retired and answers 410,
	// so the Control Panel is the only way to get one and doctl cannot do it
	// for the user — hence a link and a prompt rather than a create call.
	modelAccessKeyDocsURL = "https://docs.digitalocean.com/products/inference/how-to/manage-model-access-keys/"

	// modelAccessKeyConsolePath is where the key is created. Spelled out as a
	// breadcrumb rather than a deep link because the Control Panel has no
	// stable public URL for this page.
	modelAccessKeyConsolePath = "Control Panel → Inference → Manage → Create model access key"
)

// doInference is the resolved DigitalOcean-inference configuration written onto
// a --harness manifest. A nil *doInference means the caller supplied their own
// provider key and none of this applies.
type doInference struct {
	apiKey string
	model  string
}

// openAICompatibleHarnessAdapters are the adapters the DO-inference fallback is
// offered for. Both drive an OpenAI-compatible chat API, which is what
// inference.do-ai.run serves.
//
// claude-code is absent only because this fallback has a single default model
// and kimi-k3 is not one it can use. It is otherwise supported: OHR maps the
// HARNESS_INFERENCE_* group onto ANTHROPIC_API_KEY/ANTHROPIC_BASE_URL and
// strips the trailing /v1 its SDK would double up (MARSOHS-292). Adding it
// needs a per-adapter default — an `anthropic-claude-*` catalog alias, since
// kimi-k3 is served over Chat Completions and claude speaks Messages.
//
// cursor cannot be added at all: its adapter declares a key env but no base
// URL, so there is no way to point it at DigitalOcean.
//
// codex-agentapi is absent for a third reason — OpenAI runs that agent loop on
// its own infrastructure, so a DigitalOcean key cannot drive it.
// opencode is absent too, for a third reason: it declares no key slot, and
// resolves credentials itself from its own auth store inside the sandbox.
// "The caller supplied no key" is therefore not observable from doctl, so
// offering the fallback there would interrupt every opencode session to
// demand a credential the session did not need.
var openAICompatibleHarnessAdapters = map[string]bool{
	codexAgentName: true,
}

// harnessNativeKeyEnv is the provider key an adapter uses when the caller
// brings their own. Empty means the adapter declares no key slot.
func harnessNativeKeyEnv(agent string) string {
	switch agent {
	case codexAgentName, openAIAgentsAdapter:
		return openAIAPIKeyEnv
	case claudeCodeAgentName:
		return anthropicAPIKeyEnv
	default:
		return ""
	}
}

// promptModelAccessKey asks for a model access key, pointing at the Control
// Panel. Replaced in tests.
var promptModelAccessKey = defaultPromptModelAccessKey

func defaultPromptModelAccessKey() (string, error) {
	prompt := input.New("Paste your model access key: ",
		input.WithRequired(), input.WithHidden())
	val, err := prompt.Prompt()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(val), nil
}

// resolveDOInference decides whether a --harness session should run on
// DigitalOcean inference, and returns the configuration to write into the
// manifest when it should.
//
// It engages only when the caller named no key at all — not on the command
// line, not in the environment. A caller who supplied one gets their own
// provider, which is why this is checked against the same lookup the manifest
// expander will use rather than against os.Getenv directly.
func resolveDOInference(c *CmdConfig, agent string, suppliedSecrets map[string]string) (*doInference, error) {
	if !openAICompatibleHarnessAdapters[agent] {
		return nil, nil
	}
	native := harnessNativeKeyEnv(agent)
	if native == "" || nativeKeyAlreadySupplied(native, suppliedSecrets) {
		return nil, nil
	}

	if key, ok := lookupEnvNonEmpty(harnessInferenceAPIKeyEnv); ok {
		return &doInference{apiKey: key, model: doInferenceModel()}, nil
	}

	// Deliberately not cached anywhere on disk. doctl has no credential store
	// of its own that is fit for this — config.yaml is rewritten wholesale by
	// writeConfig and created with default permissions — and a bespoke dotfile
	// is a second, unaudited place for a live secret to sit. The key is held
	// only for this invocation; HARNESS_INFERENCE_API_KEY is how a user who
	// does not want to be asked again opts into their own storage.
	key, err := collectModelAccessKey(c, agent)
	if err != nil {
		return nil, err
	}
	return &doInference{apiKey: key, model: doInferenceModel()}, nil
}

// nativeKeyAlreadySupplied reports whether the caller provided the adapter's
// own provider key, by --secret or in the environment.
func nativeKeyAlreadySupplied(native string, suppliedSecrets map[string]string) bool {
	if v, ok := suppliedSecrets[native]; ok && strings.TrimSpace(v) != "" {
		return true
	}
	_, ok := lookupEnvNonEmpty(native)
	return ok
}

// doInferenceModel allows HARNESS_INFERENCE_MODEL to override the default
// without the user having to hand-write a manifest.
func doInferenceModel() string {
	if m := strings.TrimSpace(os.Getenv(harnessInferenceModelEnv)); m != "" {
		return m
	}
	return defaultDOInferenceModel
}

// printModelAccessKeyBanner introduces the DigitalOcean-inference path.
//
// It leads with what is about to happen, not with a variable that is unset.
// Running on DigitalOcean inference is a supported way to start a session, so
// announcing it as a missing OPENAI_API_KEY reads as a failure the user has to
// repair — when in fact their own provider is the option, and this is the
// default. The native key is still named, once, at the bottom and as an
// alternative.
func printModelAccessKeyBanner(out io.Writer, model, nativeKeyEnv string) {
	stylingEnabled = detectStyling()
	fmt.Fprintf(out, "%s %s\n", colorize("•", colHighlight),
		fmt.Sprintf("This session will run on %s (%s)",
			boldColor("DigitalOcean inference", colHighlight), boldColor(model, colHighlight)))
	fmt.Fprintf(out, "  %s\n", colorize("Create a model access key: "+modelAccessKeyConsolePath, colMuted))
	fmt.Fprintf(out, "  %s\n", colorize(modelAccessKeyDocsURL, colMuted))
	fmt.Fprintf(out, "  %s\n", colorize(
		fmt.Sprintf("Scope it to include %s. The secret is shown only once.", model), colMuted))
	if nativeKeyEnv != "" {
		fmt.Fprintf(out, "  %s\n", colorize(
			fmt.Sprintf("To use your own provider instead, cancel and set %s.", nativeKeyEnv), colMuted))
	}
}

// collectModelAccessKey explains the situation once and asks for a key.
func collectModelAccessKey(c *CmdConfig, agent string) (string, error) {
	native := harnessNativeKeyEnv(agent)
	model := doInferenceModel()

	if !canPromptForEnv() {
		return "", fmt.Errorf(
			"--%s %s needs a model access key to run on DigitalOcean inference (%s): "+
				"set %s to one created at %s, or set %s to use your own provider",
			doctl.ArgAgentHarness, agent, model,
			harnessInferenceAPIKeyEnv, modelAccessKeyDocsURL, native)
	}

	printModelAccessKeyBanner(c.Out, model, native)

	key, err := promptModelAccessKey()
	if err != nil {
		return "", err
	}
	if key == "" {
		return "", fmt.Errorf("a model access key is required to run on DigitalOcean inference")
	}

	fmt.Fprintf(c.Out, "  %s\n", colorize(
		fmt.Sprintf("Not stored — export %s to skip this prompt next time.", harnessInferenceAPIKeyEnv), colMuted))
	return key, nil
}

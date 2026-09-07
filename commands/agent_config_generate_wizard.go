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
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/commands/charm/confirm"
	"github.com/digitalocean/doctl/commands/charm/input"
	"github.com/digitalocean/doctl/commands/charm/selection"
	"github.com/erikgeiser/promptkit"
	"golang.org/x/term"
)

// The interactive half of `config generate`.
//
// Two rules shape every step. First, a step that a flag already answered is
// skipped, so the wizard is a gap-filler rather than a mode: `--harness codex`
// on a terminal asks everything except the harness. Second, every question is
// answerable with Enter — its default is shown in the prompt rather than
// applied silently afterwards — so the fast path through a fresh manifest is
// naming it and pressing Enter through the rest.
//
// Prompts render inline (never an alternate screen) so the finished Q&A stays
// in scrollback next to the manifest it produced.

// The prompt seam. Each is a package var so tests can drive the whole wizard
// without a terminal, the same way promptEnvVarValue is replaced in env tests.
var (
	generatePromptText    = defaultGeneratePromptText
	generatePromptSelect  = defaultGeneratePromptSelect
	generatePromptConfirm = defaultGeneratePromptConfirm
)

// generateSelectPageSize is where a picker starts scrolling rather than
// printing every option. Sized so the question and the list still fit a short
// terminal together.
const generateSelectPageSize = 10

// generateChoice is one option in a picker: the value written to the manifest,
// what the user reads, and an optional trailing hint.
//
// Hints are for options whose name does not speak for itself — a size's specs,
// what `ask` means. Named products (harnesses, models, providers) carry no
// hint: a sentence each turns a glanceable list into a wall of text.
type generateChoice struct {
	ID    string
	Label string
	Hint  string
}

func (c generateChoice) display() string {
	if c.Hint == "" {
		return c.Label
	}
	return c.Label + " — " + c.Hint
}

func defaultGeneratePromptText(question, placeholder string, secret bool) (string, error) {
	opts := []input.Option{}
	if placeholder != "" {
		opts = append(opts, input.WithPlaceholder(placeholder))
	}
	if secret {
		opts = append(opts, input.WithHidden())
	}
	answer, err := input.New(question+" ", opts...).Prompt()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(answer), nil
}

func defaultGeneratePromptSelect(question string, choices []generateChoice) (string, error) {
	labels := make([]string, 0, len(choices))
	for _, ch := range choices {
		labels = append(labels, ch.display())
	}
	// Filtering is off: typing should never be swallowed by a filter the user
	// did not know was there. Long lists scroll instead, so a service catalog
	// cannot push the question itself off screen.
	opts := []selection.Option{
		selection.WithPrompt(question),
		selection.WithFiltering(false),
	}
	if len(labels) > generateSelectPageSize {
		opts = append(opts, selection.WithPageSize(generateSelectPageSize))
	}
	picked, err := selection.New(labels, opts...).Select()
	if err != nil {
		return "", err
	}
	for _, ch := range choices {
		if ch.display() == picked {
			return ch.ID, nil
		}
	}
	return "", fmt.Errorf("unrecognized selection %q", picked)
}

func defaultGeneratePromptConfirm(question string, defaultYes bool) (bool, error) {
	def := confirm.No
	if defaultYes {
		def = confirm.Yes
	}
	choice, err := confirm.New(question, confirm.WithDefaultChoice(def)).Prompt()
	if err != nil {
		return false, err
	}
	return choice == confirm.Yes, nil
}

// agentGenerateCanPrompt reports whether the wizard may run: the user has not
// opted out with --no-interactive, and both ends of the terminal are real. The
// stdout check is what makes `generate > agents.yaml` silently non-interactive
// instead of writing prompts into the file.
func agentGenerateCanPrompt() bool {
	if !Interactive {
		return false
	}
	return term.IsTerminal(int(os.Stdin.Fd())) && term.IsTerminal(int(os.Stdout.Fd()))
}

// isPromptCanceled reports whether an error is a user cancel (Ctrl+C, Esc)
// rather than a failure. Cancel is a normal outcome for a wizard, so it exits
// quietly with nothing written instead of printing a Go error.
func isPromptCanceled(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, promptkit.ErrAborted) {
		return true
	}
	msg := err.Error()
	return msg == "no decision was made" || strings.Contains(msg, "canceled")
}

// runAgentGenerateWizard fills in whatever the flags left unanswered.
func runAgentGenerateWizard(c *CmdConfig, a *generateAnswers, dest *generateDestination, notes io.Writer) error {
	stylingEnabled = detectStyling()
	printGenerateHeader(notes)

	// The harness comes first: it is the one decision with no default, and it
	// shapes the defaults of everything after it (suggested name, credential
	// slots, template, model).
	steps := []func(*CmdConfig, *generateAnswers, io.Writer) error{
		askGenerateHarness,
		askGenerateName,
		askGenerateCustomImage,
		askGenerateRepos,
		askGenerateGitHubAccess,
		askGenerateInference,
		askGenerateSize,
		askGenerateTools,
		askGeneratePermissions,
		askGenerateSkills,
	}
	for _, step := range steps {
		if err := step(c, a, notes); err != nil {
			return err
		}
	}

	// Derive now rather than after confirming, so the review card names the
	// credential slots and egress hosts the answers imply — the parts a user
	// most needs to see before agreeing to them.
	if err := applyGenerateDerivations(a); err != nil {
		return err
	}

	// The review is the last chance to back out, so it runs before the
	// destination is chosen and before anything touches the filesystem.
	printGenerateReview(notes, a)
	ok, err := generatePromptConfirm("Generate this manifest?", true)
	if err != nil {
		return err
	}
	if !ok {
		return errGenerateCanceled
	}

	return askGenerateDestination(a, dest, notes)
}

// errGenerateCanceled declines at the review step. It is reported through the
// same quiet path as a Ctrl+C so a decline never looks like a failure.
var errGenerateCanceled = errors.New("canceled")

func printGenerateHeader(w io.Writer) {
	fmt.Fprintf(w, "\n%s %s\n", boldColor("Generate an agent manifest", colHighlight), colorize("· nothing is created until you run it", colMuted))
	fmt.Fprintf(w, "%s\n\n", colorize("Enter accepts the shown default at every step.", colMuted))
}

// --- steps ------------------------------------------------------------------

func askGenerateName(c *CmdConfig, a *generateAnswers, w io.Writer) error {
	if a.Name != "" {
		return nil
	}
	suggestion := suggestGenerateName(a.Harness)
	name, err := generatePromptText(fmt.Sprintf("Name? (Enter for %s)", suggestion), suggestion, false)
	if err != nil {
		return err
	}
	if name == "" {
		name = suggestion
	}
	if err := validateHostedAgentIdentifier(name); err != nil {
		return err
	}
	a.Name = name
	return nil
}

// suggestGenerateName offers a name that is readable and unlikely to collide
// with a live session, so the very first question can be answered with Enter.
// It reads the same injectable clock the create-progress code uses, so
// generated output is assertable in tests.
func suggestGenerateName(harness string) string {
	base := strings.TrimSpace(harness)
	if base == "" {
		base = "agent"
	}
	return fmt.Sprintf("%s-%s", base, creationClock().Format("0102-1504"))
}

func askGenerateHarness(c *CmdConfig, a *generateAnswers, w io.Writer) error {
	if a.Harness != "" {
		return nil
	}
	// Names only. These are products the user already came here to pick
	// between, so a sentence about each one is length without information.
	choices := make([]generateChoice, 0, len(generateHarnessCatalog))
	for _, h := range generateHarnessCatalog {
		choices = append(choices, generateChoice{ID: h.ID, Label: h.Label})
	}
	harness, err := generatePromptSelect("Which harness?", choices)
	if err != nil {
		return err
	}
	a.Harness = harness
	return nil
}

func askGenerateCustomImage(c *CmdConfig, a *generateAnswers, w io.Writer) error {
	if a.Harness != "custom" || a.Image != "" {
		return nil
	}
	fmt.Fprintf(w, "%s\n", colorize("  A custom agent runs your image; pin it by digest so the run is reproducible.", colMuted))
	image, err := generatePromptText("Container image?", "registry.digitalocean.com/team/agent@sha256:…", false)
	if err != nil {
		return err
	}
	if image == "" {
		return fmt.Errorf("a custom agent needs an image; re-run with --%s", doctl.ArgAgentImage)
	}
	a.Image = image

	if len(a.Entrypoint) > 0 {
		return nil
	}
	entrypoint, err := generatePromptText("Entrypoint? (Enter to use the image's own)", "/usr/local/bin/agent --serve", false)
	if err != nil {
		return err
	}
	if entrypoint != "" {
		a.Entrypoint = strings.Fields(entrypoint)
	}
	return nil
}

func askGenerateRepos(c *CmdConfig, a *generateAnswers, w io.Writer) error {
	if len(a.Repos) > 0 {
		return nil
	}
	attach, err := generatePromptConfirm("Attach a GitHub repository?", true)
	if err != nil {
		return err
	}
	if !attach {
		return nil
	}
	for {
		repo, err := generatePromptText("Repository? (owner/repo)", "digitalocean/doctl", false)
		if err != nil {
			return err
		}
		if repo == "" {
			return nil
		}
		ref, err := normalizeHarnessRepoRef(repo)
		if err != nil {
			// A typo should cost one retry, not the whole wizard.
			fmt.Fprintf(w, "  %s %s\n", colorize("✗", colError), colorize(err.Error(), colMuted))
			continue
		}
		a.Repos = append(a.Repos, ref)

		more, err := generatePromptConfirm("Attach another?", false)
		if err != nil {
			return err
		}
		if !more {
			return nil
		}
	}
}

func askGenerateGitHubAccess(c *CmdConfig, a *generateAnswers, w io.Writer) error {
	if len(a.Repos) == 0 || a.GitHubAccess || a.hasSecret(generateGitHubOAuthSlot) {
		return nil
	}
	fmt.Fprintf(w, "%s\n", colorize("  Lets the agent clone, push, and open pull requests as you.", colMuted))
	grant, err := generatePromptConfirm("Grant GitHub access?", true)
	if err != nil {
		return err
	}
	a.GitHubAccess = grant
	return nil
}

// askGenerateInference asks where the model comes from, then which model.
//
// The two questions belong together: the provider decides which environment
// variables the manifest writes, whether a model id is required at all, and —
// because DigitalOcean's gateway uses its own slugs — which id space the answer
// has to be in. Asking for a model without asking for a provider is what makes
// a model answer unattachable.
func askGenerateInference(c *CmdConfig, a *generateAnswers, w io.Writer) error {
	harness, ok := lookupGenerateHarness(a.Harness)
	if !ok || harness.Inference == nil {
		// A custom image, or the OpenAI-hosted loop: the sandbox never calls a
		// model endpoint, so there is nothing to choose.
		return nil
	}
	inf := harness.Inference

	if a.InferenceProvider == "" {
		if inf.NativeOnly {
			a.InferenceProvider = inferenceProviderNative
		} else {
			choices := []generateChoice{
				{ID: inferenceProviderDO, Label: "DigitalOcean Serverless Inference (recommended)"},
				{ID: inferenceProviderNative, Label: inf.ProviderLabel},
				{ID: inferenceProviderCustom, Label: "Custom endpoint"},
			}
			provider, err := generatePromptSelect("Where should inference come from?", choices)
			if err != nil {
				return err
			}
			a.InferenceProvider = provider
		}
	}

	if a.InferenceProvider == inferenceProviderCustom && a.InferenceURL == "" {
		url, err := generatePromptText("Endpoint base URL?", "https://llm.internal.example.com/v1", false)
		if err != nil {
			return err
		}
		if url == "" {
			return fmt.Errorf("a custom inference endpoint needs a base URL")
		}
		a.InferenceURL = url
	}

	if a.Model != "" {
		return nil
	}
	switch a.InferenceProvider {
	case inferenceProviderDO:
		return askGenerateDOModel(c, a, w, inf)
	case inferenceProviderCustom:
		model, err := generatePromptText("Model?", inf.NativeModelHint, false)
		if err != nil {
			return err
		}
		if model == "" {
			return fmt.Errorf("a custom inference endpoint needs a model id")
		}
		a.Model = model
	default:
		// Optional: leaving it blank keeps whatever the agent ships with, which
		// is why this is the Enter-through path.
		model, err := generatePromptText("Model? (Enter for the agent's default)", inf.NativeModelHint, false)
		if err != nil {
			return err
		}
		a.Model = model
	}
	return nil
}

// generateModelOtherID is the picker entry that drops to typing an id. It
// cannot collide with a model id, since ids come from a URL path segment.
const generateModelOtherID = "/other"

// askGenerateDOModel picks the model for Serverless Inference. A model is
// required here — the gateway has no default — and the ids are gateway slugs
// rather than the vendor ids people know, so this is one question the user
// cannot reasonably answer from memory. Hence the live catalog.
func askGenerateDOModel(c *CmdConfig, a *generateAnswers, w io.Writer, inf *harnessInference) error {
	if choices := generateDOModelChoices(c, w, inf); len(choices) > 0 {
		picked, err := generatePromptSelect("Model?", choices)
		if err != nil {
			return err
		}
		if picked != generateModelOtherID {
			a.Model = picked
			return nil
		}
	} else {
		fmt.Fprintf(w, "%s\n", colorize("  Couldn't reach the model catalog. List it with "+doInferenceModelsCmd+".", colMuted))
	}

	model, err := generatePromptText("Model?", inf.DOModelHint, false)
	if err != nil {
		return err
	}
	if model == "" {
		return fmt.Errorf("Serverless Inference has no default model; pass a model id (see %s) or pick %s inference instead", doInferenceModelsCmd, inf.ProviderLabel)
	}
	a.Model = model
	return nil
}

// generateNonChatModelFamilies are id fragments of catalog entries that cannot
// drive an agent: embedding, reranking, and image, speech, and video
// generation models. The catalog carries no type field, so the family has to be
// read off the id — and offering one of these would produce a manifest that
// validates and then fails on its first turn.
//
// Being a heuristic, it is paired with the picker's escape hatch: anything
// wrongly hidden can still be typed in.
var generateNonChatModelFamilies = []string{
	"embedding", "mini-lm", "mpnet", "e5-large", "gte-large", "bge-",
	"rerank", "-image-", "stable-diffusion", "-tts-", "-t2v-",
}

func isGenerateChatModel(id string) bool {
	lower := strings.ToLower(id)
	for _, family := range generateNonChatModelFamilies {
		if strings.Contains(lower, family) {
			return false
		}
	}
	return true
}

// generateDOModelChoices offers the live Serverless Inference catalog, or
// nothing if it cannot be reached.
//
// Fetching beats a built-in list because these ids are gateway slugs that come
// and go; a stale constant would produce a manifest that validates and then
// fails on its first turn. It also means the harness's preferred model is only
// ever offered when the catalog confirms it still exists.
//
// The call needs a model access key or a full-access token, so an unauthorized
// or offline user is expected, not exceptional: they fall through to typing an
// id, the same as with sandbox sizes.
func generateDOModelChoices(c *CmdConfig, w io.Writer, inf *harnessInference) []generateChoice {
	if c == nil || c.Inference == nil {
		return nil
	}
	ctx := context.Background()
	if c.Command != nil && c.Command.Context() != nil {
		ctx = c.Command.Context()
	}

	prog := newCreationProgress(w)
	prog.wait("Fetching models…")
	list, err := c.Inference().ListModels(ctx)
	if err != nil || list == nil || len(list.Data) == 0 {
		prog.stop()
		return nil
	}
	prog.ok("Models loaded")

	ids := make([]string, 0, len(list.Data))
	for _, m := range list.Data {
		if id := strings.TrimSpace(m.ID); id != "" && isGenerateChatModel(id) {
			ids = append(ids, id)
		}
	}
	if len(ids) == 0 {
		return nil
	}
	sort.Strings(ids)

	// The catalog arrives in no particular order, so surface the model this
	// harness is built around first and let Enter take it.
	preferred := ""
	if inf != nil {
		for _, id := range ids {
			if strings.EqualFold(id, inf.DOModelHint) {
				preferred = id
				break
			}
		}
	}
	choices := make([]generateChoice, 0, len(ids))
	if preferred != "" {
		choices = append(choices, generateChoice{ID: preferred, Label: preferred})
	}
	for _, id := range ids {
		if id != preferred {
			choices = append(choices, generateChoice{ID: id, Label: id})
		}
	}
	return append(choices, generateChoice{ID: generateModelOtherID, Label: "Enter an id"})
}

// askGenerateSize asks only for the sandbox size.
//
// The template is not asked about at all: every harness maps to one, the server
// derives it from the harness whenever the manifest omits it, and the review
// card shows which one that will be. Offering a picker whose recommended
// answer is already the answer just costs a keystroke. Pinning a different
// template — a browser or Python image under a coding harness — stays possible
// with --template.
func askGenerateSize(c *CmdConfig, a *generateAnswers, w io.Writer) error {
	if a.Size == "" {
		choices := []generateChoice{{ID: "", Label: "Auto (recommended)", Hint: "the service default for this template"}}
		for _, size := range generateSandboxSizeChoices(c, w) {
			choices = append(choices, size)
		}
		size, err := generatePromptSelect("Sandbox size?", choices)
		if err != nil {
			return err
		}
		a.Size = size
	}
	return nil
}

// generateSandboxSizeChoices offers the live size catalog when it is reachable.
// This is the one step that can block on the network, so it shows the same
// spinner session creation uses and falls back to the documented slugs on any
// failure — an unauthenticated or offline user still gets a usable picker
// rather than an error from a command that makes no API writes.
func generateSandboxSizeChoices(c *CmdConfig, w io.Writer) []generateChoice {
	fallback := make([]generateChoice, 0, len(generateFallbackSizes))
	for _, slug := range generateFallbackSizes {
		fallback = append(fallback, generateChoice{ID: slug, Label: slug})
	}
	if c == nil || c.HostedAgents == nil {
		return fallback
	}

	prog := newCreationProgress(w)
	prog.wait("Fetching sandbox sizes…")
	sizes, err := c.HostedAgents().ListSandboxSizes()
	if err != nil || len(sizes) == 0 {
		prog.stop()
		return fallback
	}
	prog.ok("Sizes loaded")

	out := make([]generateChoice, 0, len(sizes))
	for _, size := range sizes {
		if strings.TrimSpace(size.Slug) == "" {
			continue
		}
		hint := ""
		if size.VCPUs > 0 && size.MemoryMB > 0 {
			hint = fmt.Sprintf("%d vCPU · %d GB", size.VCPUs, size.MemoryMB/1024)
		}
		out = append(out, generateChoice{ID: size.Slug, Label: size.Slug, Hint: hint})
	}
	if len(out) == 0 {
		return fallback
	}
	return out
}

func askGenerateTools(c *CmdConfig, a *generateAnswers, w io.Writer) error {
	if a.ToolsPreset == "" || strings.EqualFold(a.ToolsPreset, generateToolsPresetNone) {
		if len(a.Tools) == 0 && len(a.MCPServers) == 0 {
			fmt.Fprintf(w, "%s\n", colorize("  DO tools: "+strings.Join(generateDefaultToolsPreset, ", ")+".", colMuted))
			attach, err := generatePromptConfirm("Attach the default DO tools?", true)
			if err != nil {
				return err
			}
			if attach {
				a.ToolsPreset = generateToolsPresetDefault
			}
		}
	}

	if len(a.MCPServers) > 0 {
		return nil
	}
	add, err := generatePromptConfirm("Add an MCP server?", false)
	if err != nil {
		return err
	}
	for add {
		srv, err := askGenerateMCPServer(a)
		if err != nil {
			return err
		}
		if srv == nil {
			return nil
		}
		a.MCPServers = append(a.MCPServers, *srv)
		if add, err = generatePromptConfirm("Add another MCP server?", false); err != nil {
			return err
		}
	}
	return nil
}

func askGenerateMCPServer(a *generateAnswers) (*generatedMCPServer, error) {
	name, err := generatePromptText("Server name?", "linear", false)
	if err != nil {
		return nil, err
	}
	if name == "" {
		return nil, nil
	}
	url, err := generatePromptText("Server URL?", "https://mcp.linear.app/mcp", false)
	if err != nil {
		return nil, err
	}
	if url == "" {
		return nil, fmt.Errorf("MCP server %q needs an https:// URL", name)
	}
	srv := &generatedMCPServer{Name: name, URL: url}

	tools, err := generatePromptText("Tools? (comma-separated, Enter for all)", "create_issue,list_issues", false)
	if err != nil {
		return nil, err
	}
	for _, tool := range strings.Split(tools, ",") {
		if tool = strings.TrimSpace(tool); tool != "" {
			srv.Tools = append(srv.Tools, tool)
		}
	}

	slot, err := generatePromptText("Auth secret slot? (Enter for none)", "LINEAR_API_KEY", false)
	if err != nil {
		return nil, err
	}
	if slot != "" {
		// The slot has to exist for the manifest to validate, so declaring the
		// server declares its credential too.
		a.addSecretSlot(generateSecretSlot{Name: slot, Value: secretPlaceholder(slot)})
		srv.Auth = &generatedMCPAuth{Secret: slot}
	}
	return srv, nil
}

func askGeneratePermissions(c *CmdConfig, a *generateAnswers, w io.Writer) error {
	if a.PermissionDefault != "" {
		return nil
	}
	// The recommended value leads the list so Enter selects it, and it is the
	// same constant the non-interactive path applies.
	choices := []generateChoice{
		{ID: generateDefaultPermission, Label: generateDefaultPermission + " (recommended)", Hint: "pause for approval on anything unmatched"},
		{ID: "allow", Label: "allow", Hint: "run unmatched tool calls without asking"},
		{ID: "deny", Label: "deny", Hint: "refuse anything not explicitly allowed"},
	}
	def, err := generatePromptSelect("Default for tool calls no rule matches?", choices)
	if err != nil {
		return err
	}
	a.PermissionDefault = def
	return nil
}

func askGenerateSkills(c *CmdConfig, a *generateAnswers, w io.Writer) error {
	if len(a.Skills) > 0 {
		return nil
	}
	add, err := generatePromptConfirm("Add a skill?", false)
	if err != nil {
		return err
	}
	for add {
		skill, err := askGenerateSkill()
		if err != nil {
			return err
		}
		if skill == nil {
			return nil
		}
		a.Skills = append(a.Skills, *skill)
		if add, err = generatePromptConfirm("Add another skill?", false); err != nil {
			return err
		}
	}
	return nil
}

func askGenerateSkill() (*generatedSkill, error) {
	name, err := generatePromptText("Skill name? (lowercase-with-hyphens)", "release-checklist", false)
	if err != nil {
		return nil, err
	}
	if name == "" {
		return nil, nil
	}
	if !skillNameRE.MatchString(name) {
		return nil, fmt.Errorf("skill name %q must be lowercase letters, digits, and single hyphens", name)
	}
	path, err := generatePromptText("Markdown file?", "./skills/release-checklist.md", false)
	if err != nil {
		return nil, err
	}
	if path == "" {
		return nil, fmt.Errorf("skill %q needs a markdown file", name)
	}
	skills, err := parseGenerateSkills([]string{name + "=" + path}, nil)
	if err != nil {
		return nil, err
	}
	return &skills[0], nil
}

// askGenerateDestination asks where the manifest should land. Printing is the
// default because the command's contract is "hand me the config" — saving is
// one keypress away for anyone who wants a file.
func askGenerateDestination(a *generateAnswers, dest *generateDestination, notes io.Writer) error {
	if dest.path != "" {
		return nil
	}
	suggested := generateDefaultFilename(a)
	choices := []generateChoice{
		{ID: "print", Label: "Print it here", Hint: "copy it, or re-run with --" + doctl.ArgAgentOut},
		{ID: "save", Label: "Save to a file", Hint: suggested},
	}
	where, err := generatePromptSelect("Where should it go?", choices)
	if err != nil {
		return err
	}
	if where == "print" {
		return nil
	}

	path, err := generatePromptText(fmt.Sprintf("Path? (Enter for %s)", suggested), suggested, false)
	if err != nil {
		return err
	}
	if path == "" {
		path = suggested
	}
	if _, err := os.Stat(path); err == nil {
		replace, err := generatePromptConfirm(fmt.Sprintf("%s exists — replace it?", path), false)
		if err != nil {
			return err
		}
		if !replace {
			return errGenerateCanceled
		}
		dest.overwrite = true
	}
	dest.path = path

	// `create`/`launch` discover ./agents.yaml with no --spec at all, so saving
	// under that name is a feature worth naming rather than a coincidence.
	if strings.EqualFold(strings.TrimSpace(path), "agents.yaml") {
		fmt.Fprintf(notes, "%s\n", colorize("  agents.yaml is auto-discovered, so `"+agentCLI+" launch` will pick this up with no --"+doctl.ArgAgentSpec+".", colMuted))
	}
	return nil
}

func generateDefaultFilename(a *generateAnswers) string {
	base := strings.TrimSpace(a.Name)
	if base == "" {
		base = strings.TrimSpace(a.Harness)
	}
	if base == "" {
		base = "agents"
	}
	return base + ".yaml"
}

// --- review -----------------------------------------------------------------

// printGenerateReview summarizes what is about to be written. It is a card of
// decisions, not the manifest itself: the point is to confirm intent at a
// glance, and the file follows immediately after.
func printGenerateReview(w io.Writer, a *generateAnswers) {
	var body strings.Builder
	fmt.Fprintf(&body, "%s\n\n", boldColor("Review", colHighlight))

	if a.Name != "" {
		body.WriteString(cardRow("Name", a.Name))
	}
	body.WriteString(cardRow("Harness", a.Harness))
	if a.Image != "" {
		body.WriteString(cardRow("Image", colorize(a.Image, colMuted)))
	}
	if len(a.Repos) > 0 {
		body.WriteString(cardRow("Repos", strings.Join(a.Repos, ", ")))
	}
	if provider := describeGenerateInference(a); provider != "" {
		body.WriteString(cardRow("Model", provider))
	}

	template := a.Template
	if template == "" {
		if derived := derivedTemplateFor(a.Harness); derived != "" {
			template = colorize("auto ("+derived+")", colMuted)
		} else {
			template = colorize("auto", colMuted)
		}
	}
	body.WriteString(cardRow("Template", template))

	size := a.Size
	if size == "" {
		size = colorize("auto", colMuted)
	}
	body.WriteString(cardRow("Size", size))

	if names := a.secretNames(); len(names) > 0 {
		// Names only — a review card is exactly the wrong place for a value.
		labeled := make([]string, 0, len(names))
		for _, slot := range a.Secrets {
			label := slot.Name
			switch {
			case slot.Inline:
				label += colorize(" (inlined)", colWarning)
			case strings.HasPrefix(slot.Value, "oauth/"):
				label += colorize(" (brokered)", colMuted)
			}
			labeled = append(labeled, label)
		}
		body.WriteString(cardRow("Secrets", strings.Join(labeled, ", ")))
	}

	if tools := describeGenerateTools(a); tools != "" {
		body.WriteString(cardRow("Tools", tools))
	}
	if a.PermissionDefault != "" {
		perms := a.PermissionDefault
		if n := len(a.PermissionRules); n > 0 {
			perms += colorize(fmt.Sprintf(" · %d rule(s)", n), colMuted)
		}
		body.WriteString(cardRow("Perms", perms))
	}
	if len(a.Skills) > 0 {
		names := make([]string, 0, len(a.Skills))
		for _, skill := range a.Skills {
			names = append(names, skill.Name)
		}
		body.WriteString(cardRow("Skills", strings.Join(names, ", ")))
	}
	if len(a.Egress) > 0 || a.GitHubAccess {
		hosts := append([]string{}, a.Egress...)
		if a.GitHubAccess {
			hosts = append(hosts, generateGitHubEgress...)
		}
		body.WriteString(cardRow("Egress", strings.Join(dedupeHosts(hosts), ", ")))
	}

	renderAgentCard(w, body.String())
}

// describeGenerateInference renders the model and where it is served from as
// one line, since neither is meaningful without the other.
func describeGenerateInference(a *generateAnswers) string {
	model := a.Model
	harness, ok := lookupGenerateHarness(a.Harness)
	if !ok || harness.Inference == nil {
		return model
	}
	if model == "" {
		model = colorize(harness.Inference.ProviderLabel+" default", colMuted)
	}
	switch a.InferenceProvider {
	case inferenceProviderDO:
		return model + colorize(" · DigitalOcean Serverless Inference", colMuted)
	case inferenceProviderCustom:
		return model + colorize(" · "+a.InferenceURL, colMuted)
	default:
		return model + colorize(" · "+harness.Inference.ProviderLabel, colMuted)
	}
}

func describeGenerateTools(a *generateAnswers) string {
	var parts []string
	if strings.EqualFold(a.ToolsPreset, generateToolsPresetDefault) && len(a.Tools) == 0 {
		parts = append(parts, strings.Join(generateDefaultToolsPreset, ", "))
	}
	parts = append(parts, a.Tools...)
	for _, srv := range a.MCPServers {
		parts = append(parts, srv.Name+colorize(" (mcp)", colMuted))
	}
	return strings.Join(parts, ", ")
}

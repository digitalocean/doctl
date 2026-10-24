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
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	mrand "math/rand"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"text/tabwriter"
	"time"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/commands/charm/confirm"
	"github.com/digitalocean/doctl/commands/charm/input"
	"github.com/erikgeiser/promptkit"
	"github.com/spf13/cobra"
)

const (
	mkArgOut             = "out"
	mkArgClaudeHome      = "claude-home"
	mkArgProject         = "project"
	mkArgSkills          = "skills"
	mkArgName            = "name"
	mkArgDescription     = "description"
	mkArgCreator         = "creator"
	mkArgIncludeClaudeMD = "include-claude-md"
	mkArgVersionHistory  = "version-history"
	mkArgYes             = "yes"
	mkArgWatch           = "watch"
	mkArgModel           = "model"
	mkArgInferenceURL    = "inference-base-url"
	mkArgNoAttach        = "no-attach"
	mkArgNoNativeFiles   = "no-native-files"

	mkStateFile = ".marketplace.json"
	// mkCLI is the short spelling used in lines meant to be shared.
	mkCLI                 = "doctl agent"
	mkDefaultInferenceURL = "https://inference.do-ai.run/v1"
)

// AgentMarketplace is `doctl harness-runtime marketplace`: publish a local
// Claude Code setup as a shareable listing, and deploy a listing into your
// own team. "template" already names sandbox images (`agent template`), so
// this group is "marketplace".
func AgentMarketplace() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:     "marketplace",
			Aliases: []string{"mkt"},
			// Hidden while listings live in GitHub gists; it is listed once
			// the marketplace has a DigitalOcean-hosted catalog.
			Hidden: true,
			Short:  "Publish your Claude Code setup and deploy shared ones",
			Long: `Share a Claude Code setup as an agent anyone can run on DigitalOcean.

  init     scan ~/.claude, pick skills, and write a listing folder you can review
  publish  run the automated review and publish the listing; prints a deploy line
  deploy   run a published listing in your own team

On publish, credentials in known formats are replaced by declared secret slots that the deployer fills in, and the review blocks anything that still looks like a secret. The listing folder init writes is exactly what publish sends, so read it first.`,
		},
	}
	ns := agentSubNS("agents.marketplace")

	// init, publish, list and show never call the DigitalOcean API, so they
	// work before `doctl auth init`.
	cmdInit := cmdBuilderWithInit(cmd, RunAgentsMarketplaceInit, "init",
		"Scan your Claude Code setup and write a listing folder",
		`Reads skills from ~/.claude/skills and ./.claude/skills, your CLAUDE.md, and settings.json. A picker chooses which skills to send. The folder it writes is exactly what publish will send, so you can read and edit it first. REPORT.md lists what was kept, what stays on your machine, and the secret slots a deployer must fill. Run it again with --out pointing at the same folder to refresh a listing and keep its name and listing ID.`,
		Writer, false, ns...)
	AddStringFlag(cmdInit, mkArgOut, "", "", "Folder to write (default: ./<listing-slug>)")
	AddStringFlag(cmdInit, mkArgClaudeHome, "", "", "Home directory holding .claude (default: your home)")
	AddStringFlag(cmdInit, mkArgProject, "", "", "Project directory to read .claude/skills and CLAUDE.md from (default: current directory)")
	AddStringSliceFlag(cmdInit, mkArgSkills, "", nil, "Skills to include, skipping the picker (comma separated)")
	AddStringFlag(cmdInit, mkArgName, "", "", "Listing name (default: three suggested words)")
	AddStringFlag(cmdInit, mkArgDescription, "", "", "One-line listing description")
	AddStringFlag(cmdInit, mkArgCreator, "", "", "Name shown on the listing (default: none, anonymous)")
	AddBoolFlag(cmdInit, mkArgIncludeClaudeMD, "", false, "Include your CLAUDE.md as the agent's house rules (default: ask in a terminal, otherwise no)")
	AddBoolFlag(cmdInit, mkArgVersionHistory, "", false, "Keep version history so later publishes update this listing (default: a static snapshot)")
	AddBoolFlag(cmdInit, mkArgYes, "y", false, "Accept defaults without prompting")
	cmdInit.Example = `  ` + mkCLI + ` marketplace init
  ` + mkCLI + ` marketplace init --skills product-mode,progressive-explain --name "Copper Lantern Orchard" --yes`

	cmdPublish := cmdBuilderWithInit(cmd, RunAgentsMarketplacePublish, "publish [<folder>]",
		"Review and publish a listing folder",
		`Runs the automated review over the exact files that will be sent and stops on any failure. Publishes the folder as a secret (unlisted) GitHub gist under your GitHub account, which needs a github.com login (gh auth login or GITHUB_TOKEN), and prints the line to share. Anyone with the line can read the files, every version stays readable, and deleting the gist unpublishes it. A static snapshot never changes; with version history, each publish adds a version, and --watch republishes when files in the folder change. Running agents keep the version they started with.`,
		Writer, false, ns...)
	AddBoolFlag(cmdPublish, mkArgWatch, "", false, "Keep running and republish on every change (version-history listings only)")
	AddBoolFlag(cmdPublish, mkArgYes, "y", false, "Publish without the confirmation prompt")

	cmdDeploy := CmdBuilder(cmd, RunAgentsMarketplaceDeploy, "deploy <listing-id>[@<version>]",
		"Run a published listing in your team",
		`Downloads the listing, reviews it again on your machine, shows what it can do and which secrets it needs, then creates a Claude Code session in your team. The agent runs on DigitalOcean Serverless Inference with your HARNESS_INFERENCE_API_KEY (read from --secret, the environment, or a prompt). Other secrets come only from --secret NAME=value, NAME=@file, or a hidden prompt, never from your environment, and the agent can read every secret you give it.`,
		Writer, ns...)
	AddStringSliceFlag(cmdDeploy, doctl.ArgAgentSecret, "", nil, agentSecretFlagDesc)
	AddStringFlag(cmdDeploy, mkArgModel, "", "", "Inference model ID (default: a Claude model from your Serverless Inference catalog)")
	AddStringFlag(cmdDeploy, mkArgInferenceURL, "", mkDefaultInferenceURL, "Serverless Inference base URL used to pick a model (the platform sets the session's own endpoint)")
	AddStringFlag(cmdDeploy, mkArgName, "", "", "Session name (default: listing slug plus a short suffix)")
	AddBoolFlag(cmdDeploy, mkArgYes, "y", false, "Deploy without the confirmation prompt")
	AddBoolFlag(cmdDeploy, mkArgNoAttach, "", false, "Create the session but do not open the chat")
	AddBoolFlag(cmdDeploy, mkArgNoNativeFiles, "", false, "Do not copy CLAUDE.md and skills into /workspace/.claude (inline manifest skills only)")
	cmdDeploy.Example = `  ` + mkCLI + ` marketplace deploy 8f3a1c2d9e@4b7e21a
  ` + mkCLI + ` marketplace deploy 8f3a1c2d9e --secret HARNESS_INFERENCE_API_KEY=@$HOME/.secrets/do-inference.key`

	cmdBuilderWithInit(cmd, RunAgentsMarketplaceList, "list",
		"List the listings you published",
		`Lists your marketplace listings (needs a github.com login).`,
		Writer, false, append(ns, aliasOpt("ls"))...)

	cmdBuilderWithInit(cmd, RunAgentsMarketplaceShow, "show <listing-id>[@<version>]",
		"Show a listing: what it carries and the secrets it needs",
		`Prints the listing card and the review result without deploying anything.`,
		Writer, false, ns...)
	return cmd
}

// ---- init ---------------------------------------------------------------------

func RunAgentsMarketplaceInit(c *CmdConfig) error {
	stylingEnabled = detectStyling()
	get := func(k string) string { v, _ := c.Doit.GetString(c.NS, k); return strings.TrimSpace(v) }
	yes, _ := c.Doit.GetBool(c.NS, mkArgYes)
	interactive := isInteractiveTerminal() && !yes

	home := get(mkArgClaudeHome)
	if home == "" {
		var err error
		if home, err = os.UserHomeDir(); err != nil {
			return err
		}
	}
	project := get(mkArgProject)
	if project == "" {
		project, _ = os.Getwd()
	}
	sc, err := scanClaudeSetup(home, project)
	if err != nil {
		return err
	}
	printMKScanSummary(c.Out, home, sc)
	if len(sc.Skills) == 0 && sc.ClaudeMD == "" {
		return fmt.Errorf("nothing to publish: no skills under %s or %s, and no CLAUDE.md", filepath.Join(home, ".claude", "skills"), filepath.Join(project, ".claude", "skills"))
	}

	// Re-running init into an existing listing folder refreshes that listing:
	// it keeps the name, choices and .marketplace.json, so the listing ID and
	// the deploy line stay the same.
	out := get(mkArgOut)
	var prev *mkListing
	if out != "" {
		if b, err := os.ReadFile(filepath.Join(out, "listing.json")); err == nil {
			var pl mkListing
			if json.Unmarshal(b, &pl) == nil {
				prev = &pl // listing.json alone: its skill files may be gone
			}
		}
	}

	skills, _ := c.Doit.GetStringSlice(c.NS, mkArgSkills)
	if len(skills) == 0 && len(sc.Skills) > 0 {
		if !interactive {
			return fmt.Errorf("pass --%s to choose skills (the picker needs a terminal and is skipped by --%s)", mkArgSkills, mkArgYes)
		}
		pre := map[string]bool{}
		if prev != nil {
			for _, s := range prev.Skills {
				pre[s.Name] = true
			}
		}
		if skills, err = runMKPicker(sc.Skills, pre); err != nil {
			return err
		}
	}

	name := get(mkArgName)
	if name == "" && prev != nil {
		name = prev.Name
	} else if name == "" {
		name = suggestListingName(mrand.New(mrand.NewSource(time.Now().UnixNano())))
		if interactive {
			if name, err = input.New("Listing name:", input.WithInitialValue(name), input.WithRequired(), input.WithValidator(func(s string) error {
				if slugify(s) == "" {
					return errors.New("use letters or digits")
				}
				return nil
			})).Prompt(); err != nil {
				return mkPromptErr(err)
			}
		}
	}
	desc := get(mkArgDescription)
	if desc == "" && prev != nil {
		desc = prev.Description
	} else if desc == "" {
		desc = fmt.Sprintf("A Claude Code agent with %d hand-picked skill(s).", len(skills))
		if interactive {
			if desc, err = input.New("One-line description:", input.WithInitialValue(desc), input.WithRequired()).Prompt(); err != nil {
				return mkPromptErr(err)
			}
		}
	}
	creator := get(mkArgCreator)
	if creator == "" && prev != nil {
		creator = prev.Creator
	} else if creator == "" && interactive {
		if creator, err = input.New("Name to show on the listing (empty shows none; the gist still shows your GitHub login):").Prompt(); err != nil {
			return mkPromptErr(err)
		}
	}

	includeMD, _ := c.Doit.GetBool(c.NS, mkArgIncludeClaudeMD)
	if !c.Doit.IsSet(mkArgIncludeClaudeMD) {
		includeMD = prev != nil && prev.ClaudeMD != ""
		if sc.ClaudeMD != "" && interactive {
			includeMD = askMKConfirm(fmt.Sprintf("Include your CLAUDE.md (%s, from %s) as the agent's house rules?", humanBytes(uint64(len(sc.ClaudeMD))), strings.Join(sc.ClaudeMDSources, " + ")), includeMD)
		}
	}
	if len(skills) == 0 && !(includeMD && sc.ClaudeMD != "") {
		return errors.New("nothing to publish: choose at least one skill, or include CLAUDE.md")
	}
	versioned, _ := c.Doit.GetBool(c.NS, mkArgVersionHistory)
	if prev != nil && !c.Doit.IsSet(mkArgVersionHistory) {
		versioned = prev.VersionHistory
	} else if interactive && !c.Doit.IsSet(mkArgVersionHistory) {
		fmt.Fprintln(c.Out, colorize("Version history: each publish adds a version under the same deploy line, and publish --watch can publish edits automatically. Running agents keep the version they started with.\nWithout it, publish takes a static snapshot: deployers always get exactly this version.", colMuted))
		versioned = askMKConfirm("Keep version history for this listing?", false)
	}

	mdChoice, vhChoice := "kept private", "off (static snapshot: deployers always get this version)"
	if includeMD && sc.ClaudeMD != "" {
		mdChoice = "included as house rules"
	}
	if versioned {
		vhChoice = "on (each publish adds a version; publish --watch can publish edits)"
	}
	fmt.Fprintf(c.Out, "%s CLAUDE.md: %s\n%s Version history: %s\n", colorize("✓", colSuccess), mdChoice, colorize("✓", colSuccess), vhChoice)

	l, files, err := buildListing(sc, mkInitOptions{
		Name: name, Description: desc, Creator: strings.TrimSpace(creator), Skills: skills,
		IncludeClaude: includeMD, VersionHistory: versioned, Now: time.Now(),
	})
	if err != nil {
		return err
	}

	if out == "" {
		out = filepath.Join(".", l.Slug)
	}
	if err := writeMKFolder(out, l, files); err != nil {
		return err
	}
	findings := reviewListing(l, files)
	if err := os.WriteFile(filepath.Join(out, "REPORT.md"), []byte(renderMKReport(l, findings)), 0o644); err != nil {
		return err
	}

	printMKReview(c.Out, findings)
	printAgentSuccess(c.Out, fmt.Sprintf("Wrote %s with %d skill(s) and %d secret slot(s). Read REPORT.md, then run: %s marketplace publish %s", out, len(l.Skills), len(l.SecretSlots), mkCLI, out))
	return nil
}

func askMKConfirm(q string, def bool) bool {
	choice := confirm.No
	if def {
		choice = confirm.Yes
	}
	got, err := confirm.New(q, confirm.WithDefaultChoice(choice), confirm.WithPersistPrompt(confirm.PersistPromptAlways)).Prompt()
	if err != nil {
		return def
	}
	return got == confirm.Yes
}

func printMKScanSummary(w io.Writer, home string, sc *mkScan) {
	selectable := 0
	for _, s := range sc.Skills {
		if s.Problem == "" {
			selectable++
		}
	}
	hooks := 0
	for _, n := range sc.Hooks {
		hooks += n
	}
	var body strings.Builder
	fmt.Fprintf(&body, "%s\n\n", boldColor("Your Claude Code setup", colHighlight))
	body.WriteString(mkCardRow("Skills", fmt.Sprintf("%d found, %d can be published", len(sc.Skills), selectable)))
	md := "none"
	if sc.ClaudeMD != "" {
		md = humanBytes(uint64(len(sc.ClaudeMD)))
	}
	body.WriteString(mkCardRow("CLAUDE.md", md))
	body.WriteString(mkCardRow("Hooks", fmt.Sprintf("%d (stay on your machine)", hooks)))
	body.WriteString(mkCardRow("MCP servers", fmt.Sprintf("%d (stay on your machine)", len(sc.MCPServers))))
	body.WriteString(mkCardRow("Plugins", fmt.Sprintf("%d (stay on your machine)", len(sc.Plugins))))
	renderAgentCard(w, body.String())
}

func writeMKFolder(dir string, l *mkListing, files map[string]string) error {
	if entries, err := os.ReadDir(dir); err == nil && len(entries) > 0 {
		if _, err := os.Stat(filepath.Join(dir, "listing.json")); err != nil {
			return fmt.Errorf("%s already exists and is not a listing folder; choose another --%s", dir, mkArgOut)
		}
		_ = os.RemoveAll(filepath.Join(dir, "skills")) // a re-init must not keep deselected skills
	}
	lj, err := json.MarshalIndent(l, "", "  ")
	if err != nil {
		return err
	}
	all := map[string]string{"listing.json": string(lj) + "\n"}
	for p, content := range files {
		all[p] = content
	}
	for p, content := range all {
		full := filepath.Join(dir, filepath.FromSlash(p))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			return err
		}
	}
	return nil
}

// readMKFolder loads a listing folder back, re-reading every file from disk so
// an author's edits after init are what gets reviewed and sent. The returned
// files are exactly what publish sends: REPORT.md and the "stays on your
// machine" inventory (listing.json dropped) are left out, because naming
// what stayed behind would itself publish it.
func readMKFolder(dir string) (*mkListing, map[string]string, error) {
	b, err := os.ReadFile(filepath.Join(dir, "listing.json"))
	if err != nil {
		return nil, nil, fmt.Errorf("%s is not a listing folder (no listing.json); run `%s marketplace init` first", dir, mkCLI)
	}
	var l mkListing
	if err := json.Unmarshal(b, &l); err != nil {
		return nil, nil, fmt.Errorf("listing.json: %w", err)
	}
	if err := validateMKLayout(&l); err != nil {
		return nil, nil, fmt.Errorf("listing.json: %w", err)
	}
	files := map[string]string{}
	paths := []string{}
	for i, s := range l.Skills {
		paths = append(paths, s.File)
		if data, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(s.File))); err == nil {
			l.Skills[i].Bytes = len(parseSkillFile(string(data)).Instructions)
		}
	}
	if l.ClaudeMD != "" {
		paths = append(paths, l.ClaudeMD)
	}
	for _, p := range paths {
		data, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(p)))
		if err != nil {
			return nil, nil, fmt.Errorf("listing file %s: %w", p, err)
		}
		files[p] = string(data)
	}
	// Byte counts may have changed with edits; keep listing.json truthful.
	sent := l
	sent.Dropped = nil
	lj, err := json.MarshalIndent(&sent, "", "  ")
	if err != nil {
		return nil, nil, err
	}
	files["listing.json"] = string(lj) + "\n"
	return &l, files, nil
}

// ---- publish ------------------------------------------------------------------

type mkState struct {
	ListingID string   `json:"listing_id"`
	Versions  []string `json:"versions"`
	URL       string   `json:"url"`
	// Hash is the content hash of the last published version, so publishing
	// an unchanged folder does not add an empty version to the history.
	Hash string `json:"hash,omitempty"`
}

func RunAgentsMarketplacePublish(c *CmdConfig) error {
	stylingEnabled = detectStyling()
	dir := "."
	if len(c.Args) > 0 {
		dir = c.Args[0]
	}
	watch, _ := c.Doit.GetBool(c.NS, mkArgWatch)
	yes, _ := c.Doit.GetBool(c.NS, mkArgYes)

	l, files, err := readMKFolder(dir)
	if err != nil {
		return err
	}
	if watch && !l.VersionHistory {
		return errors.New("--watch needs a listing with version history; this one is a static snapshot (re-run init with --version-history)")
	}
	token, err := mkGitHubToken()
	if err != nil {
		return err
	}
	gc := newMKGistClient(token)

	// once publishes the folder. The first call shows the full review, the
	// file list and the listed card; --watch updates are one line each.
	once := func(first bool) error {
		findings := reviewListing(l, files)
		if first || reviewFailed(findings) {
			printMKReview(c.Out, findings)
		}
		if reviewFailed(findings) {
			return errors.New("review failed; fix the items marked FAIL in the folder and publish again")
		}
		st := readMKState(dir)
		hash := mkFilesHash(files)
		if st.ListingID != "" && l.VersionHistory && st.Hash == hash {
			fmt.Fprintf(c.Out, "%s No changes since version %d; nothing published. Deploy line: %s marketplace deploy %s\n",
				colorize("•", colMuted), len(st.Versions), mkCLI, st.ListingID)
			return nil
		}
		if first {
			printMKOutgoing(c.Out, l, files)
			if isInteractiveTerminal() && !yes && !askMKConfirm("Publish these files? Anyone with the deploy line can read them.", true) {
				return errMKPickerCancelled
			}
		}
		ctx := context.Background()
		var g *mkGist
		var err error
		if st.ListingID != "" && l.VersionHistory {
			g, err = gc.update(ctx, st.ListingID, l.Name, files)
		} else {
			if st.ListingID != "" {
				warn("this listing is a static snapshot, so publishing again creates a new listing; the old deploy line keeps the old snapshot")
			}
			g, err = gc.create(ctx, l.Name, files)
			st = mkState{}
		}
		if err != nil {
			return err
		}
		st.ListingID, st.URL, st.Hash = g.ID, g.HTMLURL, hash
		if v := g.Version(); v != "" {
			st.Versions = append(st.Versions, v)
		}
		if err := writeMKState(dir, st); err != nil {
			return err
		}
		if first {
			printMKPublished(c.Out, l, g, len(st.Versions))
			return nil
		}
		pass, flag := 0, 0
		for _, f := range findings {
			if f.Level == mkPass {
				pass++
			} else if f.Level == mkFlag {
				flag++
			}
		}
		fmt.Fprintf(c.Out, "%s %s Published version %d (%s) · review %d pass, %d flag · same deploy line: %s marketplace deploy %s\n",
			colorize("✓", colSuccess), time.Now().Format("15:04:05"), len(st.Versions), mkShort(g.Version()), pass, flag, mkCLI, g.ID)
		return nil
	}

	if err := once(true); err != nil {
		return err
	}
	if !watch {
		return nil
	}

	fmt.Fprintln(c.Out, colorize("Watching "+dir+" — every saved change to the files in it is reviewed and published as a new version. Running agents keep their version. Ctrl-C stops.", colMuted))
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(stop)
	// A change is published once it reads the same on two polls in a row, so
	// an editor caught halfway through a save never publishes a cut file.
	last, seen := mkFilesHash(files), ""
	lastErr, warned := "", false
	tick := time.NewTicker(2 * time.Second)
	defer tick.Stop()
	for {
		select {
		case <-stop:
			fmt.Fprintln(c.Out, "Stopped watching.")
			return nil
		case <-tick.C:
			nl, nf, err := readMKFolder(dir)
			if err != nil {
				// One failed read is usually a save in progress; the same
				// error twice is a broken folder worth saying so once.
				if err.Error() == lastErr && !warned {
					warn("not publishing until this is fixed: %v", err)
					warned = true
				}
				lastErr, seen = err.Error(), ""
				continue
			}
			lastErr, warned = "", false
			h := mkFilesHash(nf)
			if h != last && h == seen {
				l, files, last = nl, nf, h
				if err := once(false); err != nil {
					warn("not published: %v", err)
				}
			}
			seen = h
		}
	}
}

func mkFilesHash(files map[string]string) string {
	h := sha256.New()
	for _, k := range sortedKeys(files) {
		fmt.Fprintf(h, "%s\x00%s\x00", k, files[k])
	}
	return hex.EncodeToString(h.Sum(nil))
}

func readMKState(dir string) mkState {
	var st mkState
	if b, err := os.ReadFile(filepath.Join(dir, mkStateFile)); err == nil {
		_ = json.Unmarshal(b, &st)
	}
	return st
}

func writeMKState(dir string, st mkState) error {
	b, _ := json.MarshalIndent(st, "", "  ")
	return os.WriteFile(filepath.Join(dir, mkStateFile), append(b, '\n'), 0o644)
}

func printMKOutgoing(w io.Writer, l *mkListing, files map[string]string) {
	fmt.Fprintln(w, boldColor("What will be sent", colHighlight))
	for _, p := range sortedKeys(files) {
		fmt.Fprintf(w, "  %s %-40s %s\n", colorize("↑", colSuccess), p, colorize(humanBytes(uint64(len(files[p]))), colMuted))
	}
	fmt.Fprintf(w, "%s %d item(s) stay on your machine (see REPORT.md); %d secret slot(s) are sent as names only.\n",
		colorize("•", colMuted), len(l.Dropped), len(l.SecretSlots))
	fmt.Fprintf(w, "%s The listing is an unlisted gist under your GitHub account: anyone with the deploy line can read it, and every version stays readable.\n\n",
		colorize("•", colMuted))
}

func printMKPublished(w io.Writer, l *mkListing, g *mkGist, versions int) {
	ver := g.Version()
	short := ver
	if len(short) > 7 {
		short = short[:7]
	}
	mode := "static snapshot (deployers always get this version)"
	if l.VersionHistory {
		mode = fmt.Sprintf("version history (version %d)", versions)
	}
	var body strings.Builder
	fmt.Fprintf(&body, "%s\n\n", boldColor("Listed: "+l.Name, colSuccess))
	body.WriteString(mkCardRow("Listing ID", g.ID))
	body.WriteString(mkCardRow("Version", short))
	body.WriteString(mkCardRow("Mode", mode))
	body.WriteString(mkCardRow("Skills", fmt.Sprintf("%d", len(l.Skills))))
	body.WriteString(mkCardRow("Secret slots", strings.Join(mkSlotNames(l), ", ")))
	body.WriteString(mkCardRow("Files", g.HTMLURL+" (delete the gist to unpublish)"))
	renderAgentCard(w, body.String())

	// The lines to copy sit outside the card, so a triple-click copies only
	// the command and a narrow screen does not wrap a border into it.
	share := mkCLI + " marketplace deploy " + g.ID
	if !l.VersionHistory && ver != "" {
		share += "@" + ver // a snapshot is shared pinned, so later edits to the gist never reach it
	}
	fmt.Fprintf(w, "%s\n%s\n", colorize("Share this line:", colMuted), share)
	if l.VersionHistory && ver != "" {
		fmt.Fprintf(w, "%s\n%s\n", colorize("Pinned to this exact version:", colMuted), mkCLI+" marketplace deploy "+g.ID+"@"+ver)
	}
}

func mkSlotNames(l *mkListing) []string {
	var out []string
	for _, s := range l.SecretSlots {
		n := s.Name
		if !s.Required {
			n += " (optional)"
		}
		out = append(out, n)
	}
	return out
}

// ---- review output --------------------------------------------------------------

func printMKReview(w io.Writer, fs []mkFinding) {
	fmt.Fprintln(w, boldColor("Automated review", colHighlight))
	for _, f := range fs {
		mark := colorize("PASS", colSuccess)
		switch f.Level {
		case mkFlag:
			mark = colorize("FLAG", colWarning)
		case mkFail:
			mark = colorize("FAIL", colError)
		}
		fmt.Fprintf(w, "  %s  %-30s %s\n", mark, f.Rule, colorize(f.Detail, colMuted))
	}
	fmt.Fprintln(w)
}

func renderMKReport(l *mkListing, fs []mkFinding) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# %s\n\n%s\n\n", l.Name, l.Description)
	mode := "Static snapshot: deployers always get this version, and publishing again creates a new listing."
	if l.VersionHistory {
		mode = "Version history: each publish adds a version; `publish --watch` republishes on change. Running agents keep their version."
	}
	b.WriteString(mode + "\n\n## Sent\n\n")
	for _, s := range l.Skills {
		fmt.Fprintf(&b, "- skill `%s` (%s): %s\n", s.Name, humanBytes(uint64(s.Bytes)), s.Description)
	}
	if l.ClaudeMD != "" {
		b.WriteString("- `CLAUDE.md` (house rules, placed at /workspace/CLAUDE.md on deploy)\n")
	}
	b.WriteString("\n## Secret slots (names only, the deployer fills the values)\n\n")
	for _, s := range l.SecretSlots {
		req := "required"
		if !s.Required {
			req = "optional"
		}
		fmt.Fprintf(&b, "- `%s`, %s: %s\n", s.Name, req, s.Reason)
	}
	b.WriteString("\n## Stays on your machine (this file is not sent either)\n\n")
	if len(l.Dropped) == 0 {
		b.WriteString("- nothing\n")
	}
	for _, d := range l.Dropped {
		fmt.Fprintf(&b, "- %s: %s\n", d.Item, d.Reason)
	}
	b.WriteString("\n## Automated review\n\n")
	for _, f := range fs {
		fmt.Fprintf(&b, "- **%s** %s: %s\n", strings.ToUpper(string(f.Level)), f.Rule, f.Detail)
	}
	return b.String()
}

// ---- deploy -----------------------------------------------------------------------

func fetchMKListing(ref string) (*mkListing, map[string]string, *mkGist, string, error) {
	id, version, err := parseListingRef(ref)
	if err != nil {
		return nil, nil, nil, "", err
	}
	token, _ := mkGitHubToken() // optional on read; avoids anonymous rate limits
	g, err := newMKGistClient(token).get(context.Background(), id, version)
	if err != nil {
		return nil, nil, nil, "", err
	}
	l, files, err := listingFromGist(g)
	if err != nil {
		return nil, nil, nil, "", err
	}
	if version == "" {
		version = g.Version()
	}
	for _, h := range g.History {
		if strings.HasPrefix(h.Version, version) {
			version = h.Version // a short version on the command line, expanded
			break
		}
	}
	return l, files, g, version, nil
}

func RunAgentsMarketplaceShow(c *CmdConfig) error {
	stylingEnabled = detectStyling()
	if len(c.Args) != 1 {
		return errors.New("usage: marketplace show <listing-id>[@<version>]")
	}
	l, files, g, version, err := fetchMKListing(c.Args[0])
	if err != nil {
		return err
	}
	printMKListingCard(c.Out, l, g, version, strings.Contains(c.Args[0], "@"))
	printMKReview(c.Out, reviewListing(l, files))
	return nil
}

func printMKListingCard(w io.Writer, l *mkListing, g *mkGist, version string, pinned bool) {
	short := version
	if len(short) > 7 {
		short = short[:7]
	}
	creator := l.Creator
	if creator == "" {
		creator = "anonymous"
	}
	var body strings.Builder
	fmt.Fprintf(&body, "%s\n%s\n\n", boldColor(l.Name, colHighlight), colorize(l.Description, colMuted))
	body.WriteString(mkCardRow("Creator", creator))
	body.WriteString(mkCardRow("Version", short))
	mode := "static snapshot"
	switch {
	case l.VersionHistory && pinned:
		mode = "version history, pinned to this version"
	case l.VersionHistory:
		mode = fmt.Sprintf("version history, latest of %d version(s)", len(g.History))
	}
	body.WriteString(mkCardRow("Mode", mode))
	body.WriteString(mkCardRow("Agent", mkAgent+" on DigitalOcean Serverless Inference"))
	body.WriteString(mkCardRow("Files", g.HTMLURL))
	renderAgentCard(w, body.String())

	fmt.Fprintln(w, boldColor("What this agent can do", colHighlight))
	for _, s := range l.Skills {
		d := s.Description
		if len(d) > 80 {
			d = mkTruncate(d, 80) + "…"
		}
		fmt.Fprintf(w, "  • %s %s\n", s.Name, colorize("— "+d, colMuted))
	}
	if l.ClaudeMD != "" {
		fmt.Fprintln(w, "  • CLAUDE.md house rules")
	}
	fmt.Fprintln(w, boldColor("\nWhat it can access", colHighlight))
	fmt.Fprintf(w, "  • Tool calls: %s (it asks you before running tools)\n", l.PermissionMode)
	if len(l.EgressAllowList) == 0 {
		fmt.Fprintln(w, "  • Network: the platform's default host allowlist (this listing adds none)")
	} else {
		fmt.Fprintf(w, "  • Network: the platform's default host allowlist plus %s\n", strings.Join(l.EgressAllowList, ", "))
	}
	fmt.Fprintln(w, "  • MCP servers: none")
	fmt.Fprintln(w, boldColor("\nSecrets you provide (the agent can read them)", colHighlight))
	for _, s := range l.SecretSlots {
		req := "required"
		if !s.Required {
			req = "optional"
		}
		fmt.Fprintf(w, "  • %s %s\n", s.Name, colorize("("+req+") "+s.Reason, colMuted))
	}
	fmt.Fprintln(w)
}

func RunAgentsMarketplaceDeploy(c *CmdConfig) error {
	stylingEnabled = detectStyling()
	if len(c.Args) != 1 {
		return errors.New("usage: marketplace deploy <listing-id>[@<version>]")
	}
	yes, _ := c.Doit.GetBool(c.NS, mkArgYes)
	noAttach, _ := c.Doit.GetBool(c.NS, mkArgNoAttach)
	noNative, _ := c.Doit.GetBool(c.NS, mkArgNoNativeFiles)
	model, _ := c.Doit.GetString(c.NS, mkArgModel)
	baseURL, _ := c.Doit.GetString(c.NS, mkArgInferenceURL)
	if strings.TrimSpace(baseURL) == "" {
		baseURL = mkDefaultInferenceURL
	}
	sessionName, _ := c.Doit.GetString(c.NS, mkArgName)
	pairs, _ := c.Doit.GetStringSlice(c.NS, doctl.ArgAgentSecret)
	given, err := parseAgentSecretPairs(pairs)
	if err != nil {
		return err
	}

	l, files, g, version, err := fetchMKListing(c.Args[0])
	if err != nil {
		return err
	}
	printMKListingCard(c.Out, l, g, version, strings.Contains(c.Args[0], "@"))
	findings := reviewListing(l, files)
	printMKReview(c.Out, findings)
	if reviewFailed(findings) {
		return errors.New("this listing failed review on your machine; it was not deployed")
	}
	if isInteractiveTerminal() && !yes && !askMKConfirm("Deploy this agent into your team?", true) {
		return errMKPickerCancelled
	}

	secrets := map[string]string{}
	for _, slot := range l.SecretSlots {
		v := mkSecretFromFlagOrEnv(slot.Name, given)
		if v == "" && isInteractiveTerminal() {
			// The label is derived here, not taken from the listing, so a
			// stranger's listing cannot dress up what a slot asks for.
			label := secretLabel(slot.Name)
			if !slot.Required {
				label += " (optional, Enter to skip)"
			}
			opts := []input.Option{input.WithHidden()}
			if slot.Required {
				opts = append(opts, input.WithRequired())
			}
			if v, err = input.New(label+" ["+slot.Name+"]:", opts...).Prompt(); err != nil {
				return mkPromptErr(err)
			}
		}
		if v == "" && slot.Required {
			hint := fmt.Sprintf("pass --%s %s=@path", doctl.ArgAgentSecret, slot.Name)
			if slot.Name == mkInferenceKeySlot {
				hint += " or set it in the environment"
			}
			return fmt.Errorf("this listing needs secret %s: %s", slot.Name, hint)
		}
		if v != "" {
			secrets[slot.Name] = v
		}
	}

	if strings.TrimSpace(model) == "" {
		if model, err = pickMKModel(baseURL, secrets[mkInferenceKeySlot]); err != nil {
			return err
		}
	}
	if sessionName == "" {
		suffix := make([]byte, 2)
		_, _ = rand.Read(suffix)
		sessionName = mkTruncate(l.Slug, 58) + "-" + hex.EncodeToString(suffix)
	}
	if err := validateHostedAgentIdentifier(sessionName); err != nil {
		return err
	}

	manifest, err := buildDeployManifest(l, files, mkDeployOptions{SessionName: sessionName, Model: model, BaseURL: baseURL, Secrets: secrets, ListingID: g.ID, Version: version})
	if err != nil {
		return err
	}

	maybePrintAgentPublicPreviewTermsNotice(c)
	prog := newCreationProgress(c.Out)
	defer prog.stop()
	prog.header("Deploying " + l.Name)
	prog.step("Validating configuration…")
	if err := reportAgentManifestValidation(validateAgentManifest(manifest)); err != nil {
		return err
	}
	prog.wait("Creating hosted session…")
	sess, err := c.HostedAgents().CreateSessionFromManifest(manifest, nil)
	if err != nil {
		return err
	}
	if sess == nil || sess.HostedAgentSession == nil {
		return errors.New("session create returned no session")
	}
	prog.ok("Session created · " + displaySessionRef(sess))
	printSessionCreateWarnings(sess.Warnings)
	if sess, err = waitForAgentSessionReady(c, sess, prog); err != nil {
		return err
	}

	if !noNative {
		n, err := uploadMKNativeFiles(c, sess.SessionID, l, files)
		if err != nil {
			warn("the agent is running, but copying CLAUDE.md/skills into /workspace failed: %v (inline skills still apply)", err)
		} else {
			prog.ok(fmt.Sprintf("Placed %d file(s) in /workspace (CLAUDE.md, .claude/skills)", n))
		}
	}
	prog.stop()

	printAgentSuccess(c.Out, fmt.Sprintf("%s is running as %s on %s", l.Name, sess.SessionID, model))
	if noAttach || !isInteractiveTerminal() {
		fmt.Fprintf(c.Out, "Open the chat with: %s launch %s\n", mkCLI, sess.SessionID)
		return nil
	}
	return runAgentsAttachSession(c, sess.SessionID)
}

// mkSecretFromFlagOrEnv is where a slot's value comes from before any prompt:
// --secret, and for the inference key alone, the environment. Other slots
// are never read from the environment: a listing is written by someone else
// and could name any variable the deployer happens to have exported.
func mkSecretFromFlagOrEnv(name string, given map[string]string) string {
	if v := strings.TrimSpace(given[name]); v != "" {
		return v
	}
	if name == mkInferenceKeySlot {
		return strings.TrimSpace(os.Getenv(name))
	}
	return ""
}

// uploadMKNativeFiles copies CLAUDE.md and each skill to where Claude Code
// reads them natively. Inline manifest skills land in .agents/skills, which
// Claude Code may not scan; this keeps the deployed agent faithful to the
// author's setup either way.
func uploadMKNativeFiles(c *CmdConfig, sessionID string, l *mkListing, files map[string]string) (int, error) {
	tmp, err := os.MkdirTemp("", "doctl-marketplace-")
	if err != nil {
		return 0, err
	}
	defer os.RemoveAll(tmp)
	quiet := *c
	quiet.Out = io.Discard
	targets := map[string]string{}
	if l.ClaudeMD != "" {
		targets["CLAUDE.md"] = files[l.ClaudeMD]
	}
	for _, s := range l.Skills {
		targets[".claude/skills/"+s.Name+"/SKILL.md"] = files[s.File]
	}
	n := 0
	for _, dst := range sortedKeys(targets) {
		local := filepath.Join(tmp, fmt.Sprintf("%d.md", n))
		if err := os.WriteFile(local, []byte(targets[dst]), 0o600); err != nil {
			return n, err
		}
		if err := runWorkspaceUpload(&quiet, sessionID, local, dst, false); err != nil {
			return n, err
		}
		n++
	}
	return n, nil
}

// pickMKModel chooses a Claude model from the deployer's Serverless Inference
// catalog, preferring Sonnet, so a listing never hard-codes a model ID that
// a region or account does not offer.
var pickMKModel = func(baseURL, key string) (string, error) {
	if key == "" {
		return "", fmt.Errorf("pass --%s, or provide %s so doctl can pick a Claude model from your catalog", mkArgModel, mkInferenceKeySlot)
	}
	u := strings.TrimSuffix(baseURL, "/")
	if !strings.HasSuffix(u, "/v1") {
		u += "/v1"
	}
	req, err := http.NewRequest(http.MethodGet, u+"/models", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+key)
	resp, err := (&http.Client{Timeout: 20 * time.Second}).Do(req)
	if err != nil {
		return "", fmt.Errorf("listing inference models: %w (or pass --%s)", err, mkArgModel)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("listing inference models: HTTP %d; check %s (or pass --%s)", resp.StatusCode, mkInferenceKeySlot, mkArgModel)
	}
	var out struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", err
	}
	var claude []string
	for _, m := range out.Data {
		id := strings.ToLower(m.ID)
		if strings.Contains(id, "claude") || strings.Contains(id, "anthropic") {
			claude = append(claude, m.ID)
		}
	}
	if len(claude) == 0 {
		return "", fmt.Errorf("no Claude model in your Serverless Inference catalog; pass --%s", mkArgModel)
	}
	sort.SliceStable(claude, func(i, j int) bool {
		return strings.Contains(strings.ToLower(claude[i]), "sonnet") && !strings.Contains(strings.ToLower(claude[j]), "sonnet")
	})
	return claude[0], nil
}

// ---- list ---------------------------------------------------------------------------

func RunAgentsMarketplaceList(c *CmdConfig) error {
	token, err := mkGitHubToken()
	if err != nil {
		return err
	}
	gs, err := newMKGistClient(token).listMine(context.Background())
	if err != nil {
		return err
	}
	if len(gs) == 0 {
		fmt.Fprintf(c.Out, "No listings yet. Run `%s marketplace init`, then publish.\n", mkCLI)
		return nil
	}
	tw := tabwriter.NewWriter(c.Out, 0, 4, 2, ' ', 0)
	fmt.Fprintln(tw, "LISTING ID\tNAME\tUPDATED\tDEPLOY LINE")
	for _, g := range gs {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", g.ID, strings.TrimPrefix(g.Description, mkGistDescriptionPrefix),
			g.UpdatedAt.Local().Format("2006-01-02 15:04"), mkCLI+" marketplace deploy "+g.ID)
	}
	return tw.Flush()
}

// mkCardRow is cardRow with room for the longer marketplace labels.
func mkCardRow(label, value string) string {
	return fmt.Sprintf("  %s %s\n", boldColor(fmt.Sprintf("%-13s", label), colHighlight), value)
}

func mkShort(v string) string {
	if len(v) > 7 {
		return v[:7]
	}
	return v
}

// mkPromptErr turns a Ctrl-C at a prompt into the plain cancel message.
func mkPromptErr(err error) error {
	if errors.Is(err, promptkit.ErrAborted) {
		return errMKPickerCancelled
	}
	return err
}

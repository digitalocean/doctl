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

// Pure logic behind `doctl harness-runtime marketplace`: scanning a local
// Claude Code setup, finding and redacting credentials, the publish review,
// and building the deploy manifest. Nothing in this file talks to the network
// or the terminal, so all of it is unit-tested directly.

import (
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	yaml "gopkg.in/yaml.v2"
)

const (
	mkListingSchema = "doctl.agent-marketplace/v1"
	// mkInferenceKeySlot is declared on every listing: a deployed harness runs
	// on DigitalOcean Serverless Inference, billed to the deployer's key.
	mkInferenceKeySlot = "HARNESS_INFERENCE_API_KEY"
	// mkAgent is the only harness a listing deploys as.
	mkAgent = "claude-code"

	// Caps sit at or under harness-api agentspec's skill limits, so a listing
	// that passes review here is not rejected at create time.
	mkMaxSkills          = 32
	mkMaxSkillBytes      = 32 * 1024
	mkMaxSkillTotalBytes = 256 * 1024
	mkMaxDescriptionLen  = 1024
)

// mkSkill is one Claude Code skill found on disk.
type mkSkill struct {
	Name         string
	Description  string
	Instructions string
	Source       string // "user" or "project"
	Path         string
	ExtraFiles   int    // scripts/references next to SKILL.md; not carried
	Problem      string // why it cannot be selected, empty when it can
}

// Bytes is the size that counts against the manifest caps.
func (s mkSkill) Bytes() int { return len(s.Instructions) }

type mkMCPServer struct {
	Name   string
	Kind   string // stdio, http, sse
	Source string
}

// mkScan is everything init found in a Claude Code setup.
type mkScan struct {
	Skills          []mkSkill
	ClaudeMD        string
	ClaudeMDSources []string
	ClaudeMDImports []string // `@path` lines, not resolvable on MARS
	Hooks           map[string]int
	MCPServers      []mkMCPServer
	Plugins         []string
	EnvNames        []string // settings.json env keys (values never kept)
	PermissionMode  string
}

// scanClaudeSetup reads the user-level setup under home/.claude and the
// project-level one under project/.claude. Missing files are not errors: a
// fresh machine simply has less to publish.
func scanClaudeSetup(home, project string) (*mkScan, error) {
	sc := &mkScan{Hooks: map[string]int{}}
	byName := map[string]int{}
	for _, root := range []struct{ dir, source string }{
		{filepath.Join(home, ".claude", "skills"), "user"},
		{filepath.Join(project, ".claude", "skills"), "project"},
	} {
		if root.dir == "" || (root.source == "project" && samePath(project, home)) {
			continue
		}
		skills, err := scanSkillsDir(root.dir, root.source)
		if err != nil {
			return nil, err
		}
		for _, s := range skills {
			if i, ok := byName[s.Name]; ok {
				sc.Skills[i] = s // project skills override user skills of the same name, as in Claude Code
				continue
			}
			byName[s.Name] = len(sc.Skills)
			sc.Skills = append(sc.Skills, s)
		}
	}
	sort.Slice(sc.Skills, func(i, j int) bool { return sc.Skills[i].Name < sc.Skills[j].Name })

	var parts []string
	mdPaths := []string{filepath.Join(home, ".claude", "CLAUDE.md")}
	if project != "" && !samePath(project, home) {
		mdPaths = append(mdPaths, filepath.Join(project, "CLAUDE.md"))
	}
	for _, p := range mdPaths {
		b, err := os.ReadFile(p)
		if err != nil || len(strings.TrimSpace(string(b))) == 0 {
			continue
		}
		body, imports := stripClaudeImports(string(b))
		sc.ClaudeMDImports = append(sc.ClaudeMDImports, imports...)
		if strings.TrimSpace(body) == "" {
			continue
		}
		sc.ClaudeMDSources = append(sc.ClaudeMDSources, p)
		parts = append(parts, strings.TrimSpace(body))
	}
	sc.ClaudeMD = strings.Join(parts, "\n\n")

	if b, err := os.ReadFile(filepath.Join(home, ".claude", "settings.json")); err == nil {
		var st struct {
			Hooks          map[string][]struct{ Hooks []json.RawMessage } `json:"hooks"`
			Env            map[string]string                              `json:"env"`
			EnabledPlugins map[string]json.RawMessage                     `json:"enabledPlugins"`
			Permissions    struct {
				DefaultMode string `json:"defaultMode"`
			} `json:"permissions"`
		}
		if json.Unmarshal(b, &st) == nil {
			for ev, groups := range st.Hooks {
				n := 0
				for _, g := range groups {
					n += len(g.Hooks)
				}
				sc.Hooks[ev] = n
			}
			for k := range st.Env {
				sc.EnvNames = append(sc.EnvNames, k)
			}
			sort.Strings(sc.EnvNames)
			for k := range st.EnabledPlugins {
				sc.Plugins = append(sc.Plugins, k)
			}
			sort.Strings(sc.Plugins)
			sc.PermissionMode = st.Permissions.DefaultMode
		}
	}

	sc.MCPServers = append(sc.MCPServers, readMCPServers(filepath.Join(home, ".claude.json"), project)...)
	if project != "" {
		sc.MCPServers = append(sc.MCPServers, readMCPServers(filepath.Join(project, ".mcp.json"), "")...)
	}
	return sc, nil
}

func samePath(a, b string) bool {
	if a == "" || b == "" {
		return false
	}
	ca, _ := filepath.Abs(a)
	cb, _ := filepath.Abs(b)
	return ca == cb
}

var mkSkillNameRE = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

func scanSkillsDir(dir, source string) ([]mkSkill, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []mkSkill
	for _, e := range entries {
		skillDir := filepath.Join(dir, e.Name())
		info, err := os.Stat(skillDir) // follows symlinks, as Claude Code does
		if err != nil || !info.IsDir() {
			continue
		}
		path := filepath.Join(skillDir, "SKILL.md")
		b, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		s := parseSkillFile(string(b))
		s.Source, s.Path = source, path
		if s.Name == "" {
			s.Name = e.Name()
		}
		s.ExtraFiles = countExtraFiles(skillDir)
		switch {
		case !mkSkillNameRE.MatchString(s.Name) || len(s.Name) > 64:
			s.Problem = "name must be lowercase letters and digits joined by single hyphens"
		case strings.TrimSpace(s.Description) == "":
			s.Problem = "no description in its frontmatter"
		case len(s.Description) > mkMaxDescriptionLen:
			s.Problem = fmt.Sprintf("description longer than %d characters", mkMaxDescriptionLen)
		case s.Bytes() > mkMaxSkillBytes:
			s.Problem = fmt.Sprintf("%s is over the %s per-skill limit", humanBytes(uint64(s.Bytes())), humanBytes(uint64(mkMaxSkillBytes)))
		}
		out = append(out, s)
	}
	return out, nil
}

// parseSkillFile splits SKILL.md into its frontmatter and markdown body.
func parseSkillFile(text string) mkSkill {
	text = strings.TrimPrefix(text, "\xef\xbb\xbf")
	if !strings.HasPrefix(text, "---") {
		return mkSkill{Instructions: strings.TrimSpace(text)}
	}
	rest := strings.TrimPrefix(text[3:], "\r")
	rest = strings.TrimPrefix(rest, "\n")
	end := strings.Index(rest, "\n---")
	if end < 0 {
		return mkSkill{Instructions: strings.TrimSpace(text)}
	}
	var fm struct {
		Name        string `yaml:"name"`
		Description string `yaml:"description"`
	}
	_ = yaml.Unmarshal([]byte(rest[:end]), &fm)
	body := rest[end+4:]
	if i := strings.Index(body, "\n"); i >= 0 {
		body = body[i+1:]
	} else {
		body = ""
	}
	return mkSkill{
		Name:         strings.TrimSpace(fm.Name),
		Description:  strings.TrimSpace(fm.Description),
		Instructions: strings.TrimSpace(body),
	}
}

func countExtraFiles(dir string) int {
	if real, err := filepath.EvalSymlinks(dir); err == nil {
		dir = real
	}
	n := 0
	_ = filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if !d.IsDir() && filepath.Base(p) != "SKILL.md" {
			n++
		}
		return nil
	})
	return n
}

// stripClaudeImports removes CLAUDE.md `@path` import lines. They point at
// files on the author's machine that a hosted session cannot read.
func stripClaudeImports(text string) (string, []string) {
	var keep, imports []string
	for _, line := range strings.Split(text, "\n") {
		t := strings.TrimSpace(line)
		if strings.HasPrefix(t, "@") && !strings.Contains(t, " ") && len(t) > 1 {
			imports = append(imports, t)
			continue
		}
		keep = append(keep, line)
	}
	return strings.Join(keep, "\n"), imports
}

func readMCPServers(path, project string) []mkMCPServer {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	type server struct {
		Type    string `json:"type"`
		Command string `json:"command"`
		URL     string `json:"url"`
	}
	var doc struct {
		MCPServers map[string]server `json:"mcpServers"`
		Projects   map[string]struct {
			MCPServers map[string]server `json:"mcpServers"`
		} `json:"projects"`
	}
	if json.Unmarshal(b, &doc) != nil {
		return nil
	}
	var out []mkMCPServer
	add := func(m map[string]server, source string) {
		names := make([]string, 0, len(m))
		for n := range m {
			names = append(names, n)
		}
		sort.Strings(names)
		for _, n := range names {
			s := m[n]
			kind := strings.ToLower(s.Type)
			if kind == "" {
				kind = "stdio"
				if s.URL != "" {
					kind = "http"
				}
			}
			out = append(out, mkMCPServer{Name: n, Kind: kind, Source: source})
		}
	}
	add(doc.MCPServers, filepath.Base(path))
	if project != "" {
		if abs, err := filepath.Abs(project); err == nil {
			if p, ok := doc.Projects[abs]; ok {
				add(p.MCPServers, filepath.Base(path)+" (project)")
			}
		}
	}
	return out
}

// ---- Secret detection -----------------------------------------------------

// mkTokenRE matches literal credentials by their vendor prefix. It errs on
// recognisable shapes only: a false positive blocks a publish, a false
// negative leaks a key on a public link.
var mkTokenRE = regexp.MustCompile(`\bsk-ant-[A-Za-z0-9_\-]{16,}|\bsk-(?:proj-)?[A-Za-z0-9_\-]{20,}|\b[sr]k_live_[A-Za-z0-9]{16,}|\bgh[pousr]_[A-Za-z0-9]{20,}|\bgithub_pat_[A-Za-z0-9_]{20,}|\bglpat-[A-Za-z0-9_\-]{16,}|\bxox[abprs]-[A-Za-z0-9\-]{10,}|\bA[KS]IA[0-9A-Z]{16}\b|\bdo[opr]_v1_[a-f0-9]{32,}|\bAIza[0-9A-Za-z_\-]{30,}|\bhf_[A-Za-z0-9]{20,}|\blin_api_[A-Za-z0-9]{16,}|(?:https://)?\bhooks\.slack\.com/services/[A-Za-z0-9/_\-]{20,}|\bnpm_[A-Za-z0-9]{30,}|\beyJ[A-Za-z0-9_\-]{8,}\.eyJ[A-Za-z0-9_\-]{8,}\.[A-Za-z0-9_\-]{8,}|-----BEGIN [A-Z ]*PRIVATE KEY-----`)

// mkPEMRE matches a whole PEM private key, body included, so redaction never
// leaves key material behind a replaced header. A header with no END line
// (prose that names one, or a cut-off key) takes only the base64 lines
// directly under it.
var mkPEMRE = regexp.MustCompile(`-----BEGIN [A-Z ]*PRIVATE KEY-----(?:(?s:.*?)-----END [A-Z ]*PRIVATE KEY-----|(?:\r?\n[A-Za-z0-9+/=]{16,})*)`)

// mkAssignRE matches NAME=value / NAME: value / "NAME": "value" where NAME is
// an env-style name that reads like a credential and the value is a literal
// rather than a ${reference}. Group 1 is the name; one of 2-4 is the value.
var mkAssignRE = regexp.MustCompile(`["']?\b((?:[A-Z][A-Z0-9_]*_)?(?:API_?KEY|TOKEN|SECRET|PASSWORD|PASSWD|ACCESS_KEY|PRIVATE_KEY|CREDENTIALS?)(?:_(?:KEY|VALUE|B64|BASE64))?|[A-Z][A-Z0-9_]*_(?:KEY|PAT))\b["']?\s*[:=]\s*(?:"([^"\n$][^"\n]{7,})"|'([^'\n$][^'\n]{7,})'|([^\s"'$<>{}][^\s"'<>{}]{7,}))`)

// mkLooseAssignRE is the review's wider net for names in any case and style
// (api_key, apiKey, client_secret, "password"). It is not used to rewrite
// text, because lower-case keys are often ordinary config; it only blocks.
var mkLooseAssignRE = regexp.MustCompile(`(?i)["']?\b([a-z0-9_.\-]*(?:api[_\-]?key|token|secret|passw(?:or)?d|access[_\-]?key|private[_\-]?key|credential)[a-z0-9_.\-]*)["']?\s*[:=]\s*(?:"([^"\n$][^"\n]{7,})"|'([^'\n$][^'\n]{7,})'|([^\s"'$<>{}][^\s"'<>{}]{7,}))`)

// mkURLCredRE matches the password in scheme://user:password@host.
var mkURLCredRE = regexp.MustCompile(`\b[a-zA-Z][a-zA-Z0-9+.\-]*://[^\s:/@]+:([^\s@/$]{3,})@`)

// mkBearerRE matches a literal bearer token (not a $reference).
var mkBearerRE = regexp.MustCompile(`(?i)\bbearer\s+([A-Za-z0-9._~+/=\-]{20,})`)

// mkHighEntropyRE finds long random-looking runs; see looksRandom.
var mkHighEntropyRE = regexp.MustCompile(`[A-Za-z0-9+_=\-]{32,}`)

func assignValue(sub []string) string {
	for _, v := range sub[2:] {
		if v != "" {
			return v
		}
	}
	return ""
}

// looksLikeLiteral is the loose scan's guard against code: a credential value
// has letters and digits, while os.environ["X"] or getToken() is a lookup.
func looksLikeLiteral(v string) bool {
	if strings.ContainsAny(v, "()[]") {
		return false
	}
	return strings.ContainsAny(v, "0123456789") && strings.IndexFunc(v, func(r rune) bool { return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') }) >= 0
}

// looksRandom reports whether s reads like key material rather than words or
// a hex digest: upper, lower and digits mixed, with high character entropy.
func looksRandom(s string) bool {
	var up, low, dig bool
	counts := map[rune]float64{}
	for _, r := range s {
		counts[r]++
		switch {
		case r >= 'A' && r <= 'Z':
			up = true
		case r >= 'a' && r <= 'z':
			low = true
		case r >= '0' && r <= '9':
			dig = true
		}
	}
	if !up || !low || !dig {
		return false
	}
	n, h := float64(len(s)), 0.0
	for _, c := range counts {
		p := c / n
		h -= p * math.Log2(p)
	}
	return h >= 4.2
}

// mkEnvNameRE is a valid secret slot name: an environment variable name.
var mkEnvNameRE = regexp.MustCompile(`^[A-Z_][A-Z0-9_]{0,63}$`)

// mkReservedEnvRE names variables a listing must never declare as slots:
// they steer the model endpoint or the agent runtime, so a value typed for
// one would move the session off DigitalOcean inference or change how it
// runs. HARNESS_INFERENCE_API_KEY is the one exception (isReservedSlot).
var mkReservedEnvRE = regexp.MustCompile(`^(?:ANTHROPIC_|CLAUDE_|HARNESS_)|_BASE_URL$|^(?:PATH|HOME|SHELL|NODE_OPTIONS|LD_PRELOAD|LD_LIBRARY_PATH|HTTP_PROXY|HTTPS_PROXY|ALL_PROXY|NO_PROXY)$`)

func isReservedSlot(name string) bool {
	return name != mkInferenceKeySlot && mkReservedEnvRE.MatchString(name)
}

// mkEnvRefRE finds $NAME and ${NAME} references in skill text.
var mkEnvRefRE = regexp.MustCompile(`\$\{?([A-Z][A-Z0-9_]{2,})\}?`)

var mkBaseURLAssignRE = regexp.MustCompile(`[A-Z][A-Z0-9_]*_BASE_URL\s*[=:]\s*["']?https?://`)

var mkCredentialNameRE = regexp.MustCompile(`(?i)(API_?KEY|TOKEN|SECRET|PASSWORD|PASSWD|PRIVATE_KEY|ACCESS_KEY|CREDENTIAL|_PAT$)`)

func isCredentialName(name string) bool { return mkCredentialNameRE.MatchString(name) }

func isPlaceholderValue(v string) bool {
	l := strings.ToLower(v)
	switch l {
	case "password", "passwd", "pass", "secret", "token", "changeme", "placeholder", "redacted", "none", "null",
		"guest", "admin", "root", "postgres", "test", "user":
		return true // common local-development defaults, not credentials
	}
	for _, p := range []string{"${", "$(", "xxx", "your_", "your-", "_here", "-here", "example", "changeme", "placeholder", "redacted", "dummy", "<", "***", "..."} {
		if strings.Contains(l, p) {
			return true
		}
	}
	return false
}

// redactSecrets replaces every literal credential in text with a ${NAME}
// reference and returns the slot names it declared. NAME is the variable the
// value was assigned to when there is one, else a numbered DETECTED_SECRET.
func redactSecrets(text string, next *int) (string, []string) {
	var slots []string
	detected := func() string {
		*next++
		name := fmt.Sprintf("DETECTED_SECRET_%d", *next)
		slots = append(slots, name)
		return "${" + name + "}"
	}
	text = mkPEMRE.ReplaceAllStringFunc(text, func(string) string { return detected() })
	text = replaceValueGroup(mkAssignRE, text, 2, func(sub []string, v string) (string, bool) {
		if isPlaceholderValue(v) {
			return "", false
		}
		slots = append(slots, sub[1])
		return "${" + sub[1] + "}", true
	})
	for _, re := range []*regexp.Regexp{mkURLCredRE, mkBearerRE} {
		text = replaceValueGroup(re, text, 1, func(_ []string, v string) (string, bool) {
			if isPlaceholderValue(v) {
				return "", false
			}
			return detected(), true
		})
	}
	used := map[string]bool{}
	text = mkTokenRE.ReplaceAllStringFunc(text, func(m string) string {
		// Two different keys of one kind must not share a slot, and a bare
		// key never takes a reserved name.
		if n := knownKeyName(m); n != "" && !used[n] && !isReservedSlot(n) {
			used[n] = true
			slots = append(slots, n)
			return "${" + n + "}"
		}
		return detected()
	})
	return text, slots
}

// replaceValueGroup rewrites only the value inside each match of re: the
// first group from index first on that matched. repl returns false to leave
// a match as it is. Splicing by position matters: in amqp://guest:guest@ the
// value also appears as the user name.
func replaceValueGroup(re *regexp.Regexp, text string, first int, repl func(sub []string, v string) (string, bool)) string {
	var b strings.Builder
	last := 0
	for _, idx := range re.FindAllStringSubmatchIndex(text, -1) {
		sub := make([]string, len(idx)/2)
		for g := range sub {
			if idx[2*g] >= 0 {
				sub[g] = text[idx[2*g]:idx[2*g+1]]
			}
		}
		vs, ve := -1, -1
		for g := first; g < len(sub); g++ {
			if idx[2*g] >= 0 && sub[g] != "" {
				vs, ve = idx[2*g], idx[2*g+1]
				break
			}
		}
		if vs < 0 {
			continue
		}
		r, ok := repl(sub, text[vs:ve])
		if !ok {
			continue
		}
		b.WriteString(text[last:vs])
		b.WriteString(r)
		last = ve
	}
	b.WriteString(text[last:])
	return b.String()
}

// knownKeyName names a slot after the service a key format belongs to, so a
// deployer is asked for "GitHub Token" rather than "Detected Secret 1".
func knownKeyName(m string) string {
	for _, k := range []struct{ prefix, name string }{
		{"sk-ant-", "ANTHROPIC_API_KEY"}, {"sk-", "OPENAI_API_KEY"}, {"sk_live_", "STRIPE_SECRET_KEY"},
		{"rk_live_", "STRIPE_SECRET_KEY"}, {"github_pat_", "GITHUB_TOKEN"}, {"gh", "GITHUB_TOKEN"},
		{"glpat-", "GITLAB_TOKEN"}, {"xox", "SLACK_TOKEN"}, {"AKIA", "AWS_ACCESS_KEY_ID"},
		{"ASIA", "AWS_ACCESS_KEY_ID"}, {"do", "DIGITALOCEAN_TOKEN"}, {"AIza", "GOOGLE_API_KEY"},
		{"hf_", "HF_TOKEN"}, {"lin_api_", "LINEAR_API_KEY"}, {"npm_", "NPM_TOKEN"},
		{"https://hooks.slack.com/", "SLACK_WEBHOOK_URL"}, {"hooks.slack.com/", "SLACK_WEBHOOK_URL"},
	} {
		if strings.HasPrefix(m, k.prefix) {
			return k.name
		}
	}
	return ""
}

// mkCleanText drops control and bidi-override characters from text a
// listing shows on the deployer's terminal, so a listing cannot erase or
// fake lines of the consent screen with escape sequences.
func mkCleanText(s string) string {
	return strings.Map(func(r rune) rune {
		if r < 0x20 || (r >= 0x7f && r <= 0x9f) || (r >= 0x202a && r <= 0x202e) || (r >= 0x2066 && r <= 0x2069) {
			return -1
		}
		return r
	}, s)
}

// mkLooseNameSkipRE names keys that mention token or secret without holding
// one: tokenizer, secretName, token_budget, token-file.
var mkLooseNameSkipRE = regexp.MustCompile(`(?i)(?:name|file|path|dir|env|ref|izer|budget|count|limit|type|url|uri|endpoint|id|ttl|size|len|length|mode)$`)

// mkURLRE drops URLs before the entropy scan: document and file IDs in links
// are random by design.
var mkURLRE = regexp.MustCompile(`[a-zA-Z][a-zA-Z0-9+.\-]*://\S+|\bdata:\S+|\bsha(?:256|384|512)-\S+`)

// findLeaks is the review's secret scan. It is deliberately wider than
// redactSecrets, so a value redaction missed is still caught before it is
// sent. Each hit is "path:line (why)". leaks block publish; random lists
// long random-looking strings, which are often IDs, so they are only shown.
func findLeaks(files map[string]string) (leaks, random []string) {
	for _, path := range sortedKeys(files) {
		for i, line := range strings.Split(files[path], "\n") {
			why := ""
			switch {
			case mkTokenRE.MatchString(line):
				why = "known key format"
			case mkURLCredRE.MatchString(line) && !isPlaceholderValue(mkURLCredRE.FindStringSubmatch(line)[1]):
				why = "password in a URL"
			case mkBearerRE.MatchString(line) && !isPlaceholderValue(mkBearerRE.FindStringSubmatch(line)[1]):
				why = "bearer token"
			}
			if why == "" {
				for _, m := range mkLooseAssignRE.FindAllStringSubmatch(line, -1) {
					v := assignValue(m)
					if mkLooseNameSkipRE.MatchString(m[1]) || strings.Contains(v, "/") || strings.Contains(v, "env.") || strings.Contains(v, "environ") {
						continue
					}
					if looksLikeLiteral(v) && !isPlaceholderValue(v) {
						why = m[1]
						break
					}
				}
			}
			if why != "" {
				leaks = append(leaks, fmt.Sprintf("%s:%d (%s)", path, i+1, why))
				continue
			}
			for _, m := range mkHighEntropyRE.FindAllString(mkURLRE.ReplaceAllString(line, " "), -1) {
				if looksRandom(m) {
					random = append(random, fmt.Sprintf("%s:%d", path, i+1))
					break
				}
			}
		}
	}
	return leaks, random
}

// credentialRefs lists credential-looking ${NAME} references a skill expects.
func credentialRefs(text string) []string {
	seen := map[string]bool{}
	var out []string
	for _, m := range mkEnvRefRE.FindAllStringSubmatch(text, -1) {
		if isCredentialName(m[1]) && !seen[m[1]] {
			seen[m[1]] = true
			out = append(out, m[1])
		}
	}
	return out
}

// ---- Listing ----------------------------------------------------------------

type mkListingSkill struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	File        string `json:"file"`
	Bytes       int    `json:"bytes"`
}

type mkSecretSlot struct {
	Name     string `json:"name"`
	Label    string `json:"label"`
	Required bool   `json:"required"`
	Reason   string `json:"reason"`
}

type mkDropped struct {
	Item   string `json:"item"`
	Reason string `json:"reason"`
}

// mkListing is listing.json: what a marketplace listing promises a deployer.
// It is also the record the Vendor Portal sync will send later.
type mkListing struct {
	Schema          string           `json:"schema"`
	Name            string           `json:"name"`
	Slug            string           `json:"slug"`
	Description     string           `json:"description"`
	Creator         string           `json:"creator,omitempty"`
	Agent           string           `json:"agent"`
	VersionHistory  bool             `json:"version_history"`
	Skills          []mkListingSkill `json:"skills"`
	ClaudeMD        string           `json:"claude_md,omitempty"`
	PermissionMode  string           `json:"permission_default"`
	EgressAllowList []string         `json:"egress_allow_hosts,omitempty"`
	SecretSlots     []mkSecretSlot   `json:"secret_slots"`
	Dropped         []mkDropped      `json:"dropped,omitempty"`
	CreatedAt       string           `json:"created_at"`
}

func skillFilePath(name string) string {
	return filepath.ToSlash(filepath.Join("skills", name, "SKILL.md"))
}

// gistFileName flattens a listing path: gists have no directories.
func gistFileName(path string) string {
	if strings.HasPrefix(path, "skills/") && strings.HasSuffix(path, "/SKILL.md") {
		return "skill--" + strings.TrimSuffix(strings.TrimPrefix(path, "skills/"), "/SKILL.md") + ".md"
	}
	return path
}

func renderSkillFile(name, description, instructions string) string {
	fm, _ := yaml.Marshal(map[string]string{"name": name, "description": description})
	return "---\n" + string(fm) + "---\n\n" + strings.TrimSpace(instructions) + "\n"
}

// mkInitOptions are the author's choices from the init picker.
type mkInitOptions struct {
	Name           string
	Description    string
	Creator        string
	Skills         []string
	IncludeClaude  bool
	VersionHistory bool
	Now            time.Time
}

// buildListing turns a scan and the author's choices into listing.json and
// the files to publish (keyed by listing path). Secrets are redacted on the
// way in, so the returned files are what leaves the machine.
func buildListing(sc *mkScan, o mkInitOptions) (*mkListing, map[string]string, error) {
	l := &mkListing{
		Schema:         mkListingSchema,
		Name:           o.Name,
		Slug:           slugify(o.Name),
		Description:    o.Description,
		Creator:        o.Creator,
		Agent:          mkAgent,
		VersionHistory: o.VersionHistory,
		PermissionMode: "ask",
		CreatedAt:      o.Now.UTC().Format(time.RFC3339),
	}
	files := map[string]string{}
	type slotInfo struct {
		reason   string
		required bool
	}
	slots := map[string]slotInfo{mkInferenceKeySlot: {"DigitalOcean Serverless Inference key the agent runs on", true}}
	next := 0
	reserved := map[string]bool{}
	addSlot := func(n string, info slotInfo, overwrite bool) {
		if isReservedSlot(n) || !mkEnvNameRE.MatchString(n) {
			reserved[n] = true
			return
		}
		if _, ok := slots[n]; ok && !overwrite {
			return
		}
		slots[n] = info
	}

	want := map[string]bool{}
	for _, n := range o.Skills {
		want[n] = true
	}
	for _, s := range sc.Skills {
		if !want[s.Name] {
			reason := "not selected"
			if s.Problem != "" {
				reason = s.Problem
			}
			l.Dropped = append(l.Dropped, mkDropped{Item: "skill " + s.Name, Reason: reason})
			continue
		}
		if s.Problem != "" {
			return nil, nil, fmt.Errorf("skill %q cannot be published: %s", s.Name, s.Problem)
		}
		delete(want, s.Name)
		body, found := redactSecrets(s.Instructions, &next)
		for _, n := range found {
			addSlot(n, slotInfo{"found as a literal value in skill " + s.Name + " (replaced with ${" + n + "})", false}, true)
		}
		for _, n := range credentialRefs(body) {
			addSlot(n, slotInfo{"referenced by skill " + s.Name, false}, false)
		}
		path := skillFilePath(s.Name)
		files[path] = renderSkillFile(s.Name, s.Description, body)
		l.Skills = append(l.Skills, mkListingSkill{Name: s.Name, Description: s.Description, File: path, Bytes: len(body)})
		if s.ExtraFiles > 0 {
			l.Dropped = append(l.Dropped, mkDropped{Item: fmt.Sprintf("%d support file(s) of skill %s", s.ExtraFiles, s.Name), Reason: "only SKILL.md is carried in v1"})
		}
	}
	for n := range want {
		return nil, nil, fmt.Errorf("skill %q was not found in your Claude Code setup", n)
	}

	if o.IncludeClaude && strings.TrimSpace(sc.ClaudeMD) != "" {
		body, found := redactSecrets(sc.ClaudeMD, &next)
		for _, n := range found {
			addSlot(n, slotInfo{"found as a literal value in CLAUDE.md (replaced with ${" + n + "})", false}, true)
		}
		files["CLAUDE.md"] = body + "\n"
		l.ClaudeMD = "CLAUDE.md"
	} else if strings.TrimSpace(sc.ClaudeMD) != "" {
		l.Dropped = append(l.Dropped, mkDropped{Item: "CLAUDE.md", Reason: "not selected"})
	}
	for _, imp := range sc.ClaudeMDImports {
		l.Dropped = append(l.Dropped, mkDropped{Item: "CLAUDE.md import " + imp, Reason: "points at a file on your machine"})
	}
	for _, ev := range sortedKeys(sc.Hooks) {
		l.Dropped = append(l.Dropped, mkDropped{Item: fmt.Sprintf("%d %s hook(s)", sc.Hooks[ev], ev), Reason: "hooks run local commands; not carried in v1"})
	}
	for _, m := range sc.MCPServers {
		reason := "runs a process on your machine"
		if m.Kind != "stdio" {
			reason = "only DigitalOcean catalog MCP servers are allowed in v1"
		}
		l.Dropped = append(l.Dropped, mkDropped{Item: fmt.Sprintf("MCP server %s (%s, %s)", m.Name, m.Kind, m.Source), Reason: reason})
	}
	for _, p := range sc.Plugins {
		l.Dropped = append(l.Dropped, mkDropped{Item: "plugin " + p, Reason: "plugins are not carried in v1"})
	}
	for _, n := range sc.EnvNames {
		if isCredentialName(n) {
			addSlot(n, slotInfo{"set in your settings.json env (value not copied)", false}, false)
			continue
		}
		l.Dropped = append(l.Dropped, mkDropped{Item: "settings env " + n, Reason: "local environment; set it at deploy if needed"})
	}
	if sc.PermissionMode != "" && sc.PermissionMode != "default" {
		l.Dropped = append(l.Dropped, mkDropped{Item: "permission mode " + sc.PermissionMode, Reason: "listings always start in ask mode"})
	}

	for _, n := range sortedKeys(reserved) {
		reason := "reserved: the deployer's doctl sets the model endpoint and key, so it is not a slot (any value was replaced)"
		if !mkEnvNameRE.MatchString(n) {
			reason = "not an upper-case environment variable name, so it is not a slot; set it at deploy if needed"
		}
		l.Dropped = append(l.Dropped, mkDropped{Item: "secret " + n, Reason: reason})
	}
	for _, n := range sortedKeys(slots) {
		l.SecretSlots = append(l.SecretSlots, mkSecretSlot{Name: n, Label: secretLabel(n), Required: slots[n].required, Reason: slots[n].reason})
	}
	if len(l.Skills) > mkMaxSkills {
		return nil, nil, fmt.Errorf("%d skills selected; a listing holds at most %d", len(l.Skills), mkMaxSkills)
	}
	return l, files, nil
}

func secretLabel(name string) string {
	if name == mkInferenceKeySlot {
		return "DigitalOcean inference API key"
	}
	words := strings.Fields(strings.ToLower(strings.ReplaceAll(name, "_", " ")))
	for i, w := range words {
		switch w {
		case "api", "url", "id", "pat", "ssh", "aws", "mcp":
			words[i] = strings.ToUpper(w)
		case "github":
			words[i] = "GitHub"
		default:
			words[i] = strings.ToUpper(w[:1]) + w[1:]
		}
	}
	return strings.Join(words, " ")
}

func sortedKeys[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func slugify(s string) string {
	var b strings.Builder
	dash := false
	for _, r := range strings.ToLower(s) {
		switch {
		case (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'):
			b.WriteRune(r)
			dash = false
		case !dash && b.Len() > 0:
			b.WriteByte('-')
			dash = true
		}
	}
	return strings.Trim(b.String(), "-")
}

var mkNouns = []string{
	"anchor", "beacon", "bramble", "canyon", "cedar", "comet", "compass", "copper", "cove", "delta",
	"ember", "falcon", "fern", "fjord", "forge", "garnet", "glacier", "harbor", "heron", "island",
	"juniper", "kestrel", "lantern", "lighthouse", "maple", "meadow", "mesa", "nebula", "orchard", "otter",
	"pebble", "pine", "prairie", "quartz", "raven", "reef", "river", "saddle", "sparrow", "summit",
	"thistle", "tundra", "valley", "walnut", "willow", "harvest", "atlas", "cobalt", "dune", "lagoon",
}

// suggestListingName returns three distinct random nouns, title-cased.
func suggestListingName(r *rand.Rand) string {
	idx := r.Perm(len(mkNouns))[:3]
	words := make([]string, 3)
	for i, j := range idx {
		words[i] = strings.ToUpper(mkNouns[j][:1]) + mkNouns[j][1:]
	}
	return strings.Join(words, " ")
}

// ---- Review -----------------------------------------------------------------

type mkLevel string

const (
	mkPass mkLevel = "pass"
	mkFlag mkLevel = "flag" // would need a human reviewer before catalog listing
	mkFail mkLevel = "fail" // blocks publish
)

type mkFinding struct {
	Level  mkLevel
	Rule   string
	Detail string
}

// reviewListing runs the automated publish review over exactly the bytes
// that would leave the machine. Deploy runs it again on what it downloaded,
// so a listing edited by hand after publish is still checked: nothing in
// listing.json is taken on trust that the files themselves can answer.
func reviewListing(l *mkListing, files map[string]string) []mkFinding {
	var out []mkFinding
	add := func(level mkLevel, rule, detail string) { out = append(out, mkFinding{level, rule, detail}) }

	// R1: no credential values in anything we send. findLeaks is wider than
	// the redaction init applied, so it also catches what redaction missed.
	leaks, random := findLeaks(files)
	if len(leaks) > 0 {
		add(mkFail, "No secret values", "credential-looking values at "+strings.Join(leaks, ", ")+"; replace each with ${NAME}")
	} else {
		add(mkPass, "No secret values", "no credential values in the files to send")
	}
	if len(random) > 0 {
		add(mkFlag, "Random-looking values", "check these are IDs, not keys: "+strings.Join(random, ", "))
	}

	// R2: every credential the skills need is a declared slot, never a value.
	declared := map[string]bool{}
	var undeclared, badNames []string
	for _, s := range l.SecretSlots {
		declared[s.Name] = true
		if !mkEnvNameRE.MatchString(s.Name) {
			badNames = append(badNames, s.Name)
		}
	}
	for _, path := range sortedKeys(files) {
		for _, n := range credentialRefs(files[path]) {
			if !declared[n] && !isReservedSlot(n) {
				undeclared = append(undeclared, n+" in "+path)
			}
		}
	}
	if !declared[mkInferenceKeySlot] {
		undeclared = append(undeclared, mkInferenceKeySlot)
	}
	switch {
	case len(badNames) > 0:
		add(mkFail, "Secrets are declared slots", "slot names must be environment variable names: "+strings.Join(badNames, ", "))
	case len(undeclared) > 0:
		add(mkFail, "Secrets are declared slots", "not declared: "+strings.Join(undeclared, ", "))
	default:
		add(mkPass, "Secrets are declared slots", fmt.Sprintf("%d slot(s), no values", len(l.SecretSlots)))
	}

	// R3: a listing cannot redirect model traffic. The deployer's doctl sets
	// the inference endpoint and the agent; a slot named like ANTHROPIC_API_KEY
	// or *_BASE_URL would let a typed value move the session elsewhere.
	var steer []string
	for _, s := range l.SecretSlots {
		if isReservedSlot(s.Name) {
			steer = append(steer, s.Name)
		}
	}
	switch {
	case l.Agent != mkAgent:
		add(mkFail, "No model-endpoint override", fmt.Sprintf("agent %q is not %s", l.Agent, mkAgent))
	case len(steer) > 0:
		add(mkFail, "No model-endpoint override", "reserved names declared as slots: "+strings.Join(steer, ", "))
	default:
		add(mkPass, "No model-endpoint override", "the deployer's doctl sets the agent and the inference endpoint")
	}
	var mentions []string
	for _, path := range sortedKeys(files) {
		if mkBaseURLAssignRE.MatchString(files[path]) {
			mentions = append(mentions, path)
		}
	}
	if len(mentions) > 0 {
		add(mkFlag, "Endpoint instructions in text", "a *_BASE_URL assignment appears in "+strings.Join(mentions, ", "))
	}

	// R4: egress. Wildcards fail; no allowlist leaves the platform default,
	// which is allowed but flagged.
	var wild []string
	for _, h := range l.EgressAllowList {
		if strings.ContainsAny(h, "*/ ") {
			wild = append(wild, h)
		}
	}
	switch {
	case len(wild) > 0:
		add(mkFail, "Egress allowlist", "not plain host names: "+strings.Join(wild, ", "))
	case len(l.EgressAllowList) == 0:
		add(mkFlag, "Egress allowlist", "none declared, so the platform's default host allowlist applies; a human reviewer would check this before catalog listing")
	default:
		add(mkPass, "Egress allowlist", strings.Join(l.EgressAllowList, ", "))
	}

	// R5: every tool call asks the deployer first.
	if l.PermissionMode != "ask" {
		add(mkFail, "Permissions", fmt.Sprintf("default %q; listings must use ask, so every tool call asks the deployer", l.PermissionMode))
	} else {
		add(mkPass, "Permissions", "default ask: tool calls ask the deployer")
	}

	// R6: manifest limits, measured on the files rather than on what
	// listing.json says about them.
	total, bad := 0, []string{}
	for _, s := range l.Skills {
		n := len(parseSkillFile(files[s.File]).Instructions)
		total += n
		if n > mkMaxSkillBytes {
			bad = append(bad, s.Name+": over "+humanBytes(uint64(mkMaxSkillBytes)))
		}
		if _, ok := files[s.File]; !ok {
			bad = append(bad, s.Name+": file missing")
		}
	}
	if l.ClaudeMD != "" && len(files[l.ClaudeMD]) > mkMaxSkillTotalBytes {
		bad = append(bad, "CLAUDE.md: over "+humanBytes(uint64(mkMaxSkillTotalBytes)))
	}
	if len(l.Skills) > mkMaxSkills {
		bad = append(bad, fmt.Sprintf("%d skills (max %d)", len(l.Skills), mkMaxSkills))
	}
	if total > mkMaxSkillTotalBytes {
		bad = append(bad, fmt.Sprintf("%s total (max %s)", humanBytes(uint64(total)), humanBytes(uint64(mkMaxSkillTotalBytes))))
	}
	if err := validateMKLayout(l); err != nil {
		bad = append(bad, err.Error())
	}
	if len(bad) > 0 {
		add(mkFail, "Skill limits", strings.Join(bad, "; "))
	} else {
		add(mkPass, "Skill limits", fmt.Sprintf("%d skill(s), %s of %s", len(l.Skills), humanBytes(uint64(total)), humanBytes(uint64(mkMaxSkillTotalBytes))))
	}
	return out
}

// validateMKLayout pins every path in listing.json to the layout init writes,
// so neither a hand-edited folder on publish nor a stranger's listing on
// deploy can name a file outside it (../../.netrc) or reuse a skill name.
func validateMKLayout(l *mkListing) error {
	seen := map[string]bool{}
	for _, s := range l.Skills {
		switch {
		case !mkSkillNameRE.MatchString(s.Name):
			return fmt.Errorf("skill %q: invalid name", s.Name)
		case seen[s.Name]:
			return fmt.Errorf("skill %q is listed twice", s.Name)
		case s.File != skillFilePath(s.Name):
			return fmt.Errorf("skill %s: file must be %s", s.Name, skillFilePath(s.Name))
		}
		seen[s.Name] = true
	}
	if l.ClaudeMD != "" && l.ClaudeMD != "CLAUDE.md" {
		return fmt.Errorf("claude_md must be CLAUDE.md, not %q", l.ClaudeMD)
	}
	return nil
}

func reviewFailed(fs []mkFinding) bool {
	for _, f := range fs {
		if f.Level == mkFail {
			return true
		}
	}
	return false
}

// ---- Deploy manifest --------------------------------------------------------

type mkDeployOptions struct {
	SessionName string
	Model       string
	BaseURL     string
	Secrets     map[string]string
	// ListingID and Version label the session, so deploys of a listing can be
	// counted without asking the deployer anything.
	ListingID string
	Version   string
}

// buildDeployManifest writes the flat agents.yaml a deploy creates the
// session from. Skills travel inline (the reviewed copy); secret values come
// only from the deployer.
func buildDeployManifest(l *mkListing, files map[string]string, o mkDeployOptions) ([]byte, error) {
	type skill struct {
		Name         string `yaml:"name"`
		Description  string `yaml:"description"`
		Instructions string `yaml:"instructions"`
	}
	doc := struct {
		Name        string            `yaml:"name"`
		Description string            `yaml:"description,omitempty"`
		Labels      map[string]string `yaml:"labels,omitempty"`
		Agent       string            `yaml:"agent"`
		Env         map[string]string `yaml:"env,omitempty"`
		Secrets     map[string]string `yaml:"secrets,omitempty"`
		Permissions map[string]string `yaml:"permissions"`
		Egress      map[string]any    `yaml:"egress,omitempty"`
		Skills      []skill           `yaml:"skills,omitempty"`
	}{
		Name:        o.SessionName,
		Description: mkTruncate(l.Name+" (agent marketplace)", mkMaxDescriptionLen),
		Labels:      map[string]string{"source": "agent-marketplace"},
		Agent:       mkAgent,
		Env: map[string]string{
			"HARNESS_INFERENCE_BASE_URL": o.BaseURL,
			"HARNESS_INFERENCE_MODEL":    o.Model,
		},
		Secrets:     map[string]string{},
		Permissions: map[string]string{"default": l.PermissionMode},
	}
	if o.ListingID != "" {
		doc.Labels["marketplace-listing"] = o.ListingID
	}
	if o.Version != "" {
		doc.Labels["marketplace-version"] = o.Version
	}
	if len(l.EgressAllowList) > 0 {
		doc.Egress = map[string]any{"allow_hosts": l.EgressAllowList}
	}
	for _, slot := range l.SecretSlots {
		v := o.Secrets[slot.Name]
		if v == "" && slot.Required {
			return nil, fmt.Errorf("secret %s is required by this listing", slot.Name)
		}
		if v != "" {
			doc.Secrets[slot.Name] = v
		}
	}
	for _, s := range l.Skills {
		parsed := parseSkillFile(files[s.File])
		if parsed.Instructions == "" {
			return nil, fmt.Errorf("listing file for skill %s is empty", s.Name)
		}
		doc.Skills = append(doc.Skills, skill{Name: s.Name, Description: s.Description, Instructions: parsed.Instructions})
	}
	return yaml.Marshal(doc)
}

func mkTruncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

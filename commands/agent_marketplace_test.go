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
	"encoding/json"
	"io"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	yaml "gopkg.in/yaml.v2"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
}

// fakeClaudeHome builds a small but realistic setup: two good skills, one
// with support files and a secret reference, one with no description, one
// oversized, a CLAUDE.md with an import, hooks, env, a plugin and MCP servers.
func fakeClaudeHome(t *testing.T) (home, project string) {
	t.Helper()
	home, project = t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(home, ".claude/skills/product-mode/SKILL.md"),
		"---\nname: product-mode\ndescription: Use when framing a product decision.\n---\n\n# Product mode\nAsk who the user is.\n")
	writeFile(t, filepath.Join(home, ".claude/skills/gh-helper/SKILL.md"),
		"---\nname: gh-helper\ndescription: Use when calling GitHub.\n---\nRun curl -H \"Authorization: Bearer $GITHUB_TOKEN\" https://api.github.com.\nexport ANTHROPIC_API_KEY=sk-ant-abcdefghijklmnopqrstuvwx\nPost to Linear: LINEAR_API_KEY=lin_api_9f8Kx2mQ7zR4tW1vB6\n")
	writeFile(t, filepath.Join(home, ".claude/skills/gh-helper/scripts/run.sh"), "echo hi\n")
	writeFile(t, filepath.Join(home, ".claude/skills/no-desc/SKILL.md"), "---\nname: no-desc\n---\nbody\n")
	writeFile(t, filepath.Join(home, ".claude/skills/huge/SKILL.md"),
		"---\nname: huge\ndescription: Use always.\n---\n"+strings.Repeat("x", mkMaxSkillBytes+1)+"\n")
	writeFile(t, filepath.Join(home, ".claude/CLAUDE.md"), "Be terse.\n@~/.cursor/rules/caveman.mdc\nNever push to main.\n")
	writeFile(t, filepath.Join(home, ".claude/settings.json"), `{
  "hooks": {"PreToolUse": [{"hooks": [{"type":"command"},{"type":"command"}]}], "Stop": [{"hooks": [{"type":"command"}]}]},
  "env": {"OPENAI_API_KEY": "sk-proj-shouldneverbecopied000000", "EDITOR": "vim", "npm_config_authtoken": "x"},
  "enabledPlugins": {"understand-anything@understand-anything": true},
  "permissions": {"defaultMode": "bypassPermissions"}
}`)
	writeFile(t, filepath.Join(home, ".claude.json"), `{"mcpServers": {"codegraph": {"command": "codegraph"}, "remote": {"type": "http", "url": "https://x"}}}`)
	writeFile(t, filepath.Join(project, ".claude/skills/product-mode/SKILL.md"),
		"---\nname: product-mode\ndescription: Project override.\n---\nProject body.\n")
	return home, project
}

func TestScanClaudeSetup(t *testing.T) {
	home, project := fakeClaudeHome(t)
	sc, err := scanClaudeSetup(home, project)
	require.NoError(t, err)

	byName := map[string]mkSkill{}
	for _, s := range sc.Skills {
		byName[s.Name] = s
	}
	require.Len(t, byName, 4)
	assert.Equal(t, "project", byName["product-mode"].Source, "project skill overrides the user skill of the same name")
	assert.Equal(t, "Project body.", byName["product-mode"].Instructions)
	assert.Equal(t, 1, byName["gh-helper"].ExtraFiles)
	assert.Contains(t, byName["no-desc"].Problem, "no description")
	assert.Contains(t, byName["huge"].Problem, "per-skill limit")

	assert.Equal(t, "Be terse.\nNever push to main.", sc.ClaudeMD)
	assert.Equal(t, []string{"@~/.cursor/rules/caveman.mdc"}, sc.ClaudeMDImports)
	assert.Equal(t, map[string]int{"PreToolUse": 2, "Stop": 1}, sc.Hooks)
	assert.Equal(t, []string{"EDITOR", "OPENAI_API_KEY", "npm_config_authtoken"}, sc.EnvNames)
	assert.Equal(t, "bypassPermissions", sc.PermissionMode)
	assert.Len(t, sc.MCPServers, 2)
	assert.Equal(t, "stdio", sc.MCPServers[0].Kind)
	assert.Equal(t, "http", sc.MCPServers[1].Kind)
}

func TestRedactSecrets(t *testing.T) {
	next := 0
	out, slots := redactSecrets("token ghp_abcdefghijklmnopqrstuvwxyz12 and API_KEY=realvalue123 and API_KEY=your_key_here", &next)
	assert.Equal(t, "token ${GITHUB_TOKEN} and API_KEY=${API_KEY} and API_KEY=your_key_here", out, "a known key format is named after its service")
	assert.ElementsMatch(t, []string{"API_KEY", "GITHUB_TOKEN"}, slots)

	for in, want := range map[string]string{
		"SECRET_KEY=a1b2c3d4e5f6":                                     "SECRET_KEY=${SECRET_KEY}",
		`{"STRIPE_KEY": "pk9f8e7d6c5b4a"}`:                            `{"STRIPE_KEY": "${STRIPE_KEY}"}`,
		`DB_PASSWORD="correct horse battery"`:                         `DB_PASSWORD="${DB_PASSWORD}"`,
		"API_KEY=YourCompanyProdKey9":                                 "API_KEY=${API_KEY}",
		"ADO_PAT=q8w7e6r5t4y3u2i1":                                    "ADO_PAT=${ADO_PAT}",
		"postgres://admin:S3cr3t99@db.internal:5432/app":              "postgres://admin:${DETECTED_SECRET_1}@db.internal:5432/app",
		"curl -H 'Authorization: Bearer abcdefghijklmnop1234567890'":  "curl -H 'Authorization: Bearer ${DETECTED_SECRET_1}'",
		"amqp://svc:svc@mq":                                           "amqp://svc:${DETECTED_SECRET_1}@mq", // only the password, even when the user name matches it
		"amqp://guest:guest@localhost":                                "amqp://guest:guest@localhost",       // local-development default
		`DB_PASSWORD="Xk9$mP2!qLw8zz"`:                                `DB_PASSWORD="${DB_PASSWORD}"`,
		"ghp_aaaaaaaaaaaaaaaaaaaaaaaa1 ghp_bbbbbbbbbbbbbbbbbbbbbbbb2": "${GITHUB_TOKEN} ${DETECTED_SECRET_1}",
		"sk-ant-abcdefghijklmnopqrstuvwx":                             "${DETECTED_SECRET_1}", // a bare key never takes a reserved name
	} {
		n := 0
		got, _ := redactSecrets(in, &n)
		assert.Equal(t, want, got, in)
	}

	pem := "key:\n-----BEGIN RSA PRIVATE KEY-----\nMIIEowIBAAKCAQEAu1SU1LfVLPHCozMxH2Mo4lgOEePzNm0tRgeLezV6ffAt0gun\n-----END RSA PRIVATE KEY-----\ndone"
	got, slots := redactSecrets(pem, &next)
	assert.NotContains(t, got, "MIIEow", "the key body goes with its header")
	assert.NotContains(t, got, "END RSA")
	assert.True(t, strings.HasSuffix(got, "\ndone"))
	assert.Len(t, slots, 1)

	prose := "The file starts with -----BEGIN OPENSSH PRIVATE KEY----- and ends with the END line.\nStep 2: copy it.\nStep 3: chmod 600."
	got, _ = redactSecrets(prose, &next)
	assert.Contains(t, got, "Step 2: copy it.\nStep 3: chmod 600.", "a header named in prose does not take the rest of the file")

	plain := "a risk-assessment-framework-for-teams and a desk-research-methodology-guide"
	out, slots = redactSecrets(plain, &next)
	assert.Equal(t, plain, out, "hyphenated words that contain sk- are not tokens")
	assert.Empty(t, slots)
}

func buildTestListing(t *testing.T) (*mkListing, map[string]string) {
	t.Helper()
	home, project := fakeClaudeHome(t)
	sc, err := scanClaudeSetup(home, project)
	require.NoError(t, err)
	l, files, err := buildListing(sc, mkInitOptions{
		Name: "Copper Lantern Orchard", Description: "test", Skills: []string{"product-mode", "gh-helper"},
		IncludeClaude: true, Now: time.Unix(0, 0),
	})
	require.NoError(t, err)
	return l, files
}

func TestBuildListing(t *testing.T) {
	l, files := buildTestListing(t)
	assert.Equal(t, "copper-lantern-orchard", l.Slug)
	assert.Equal(t, "ask", l.PermissionMode)
	require.Len(t, l.Skills, 2)

	slots := map[string]mkSecretSlot{}
	for _, s := range l.SecretSlots {
		slots[s.Name] = s
	}
	assert.True(t, slots[mkInferenceKeySlot].Required)
	assert.NotContains(t, slots, "ANTHROPIC_API_KEY", "a name that steers the model endpoint is never a slot")
	assert.False(t, slots["LINEAR_API_KEY"].Required, "a literal value becomes an optional slot")
	assert.False(t, slots["GITHUB_TOKEN"].Required, "a reference becomes an optional slot")
	assert.False(t, slots["OPENAI_API_KEY"].Required, "a settings env credential becomes an optional slot")
	assert.NotContains(t, slots, "EDITOR")

	for path, content := range files {
		assert.NotContains(t, content, "sk-ant-abcdefghijklmnopqrstuvwx", path)
		assert.NotContains(t, content, "shouldneverbecopied", path)
		assert.NotContains(t, content, "lin_api_9f8Kx2mQ7zR4tW1vB6", path)
	}
	assert.Contains(t, files["skills/gh-helper/SKILL.md"], "ANTHROPIC_API_KEY=${ANTHROPIC_API_KEY}")
	assert.Equal(t, "Be terse.\nNever push to main.\n", files["CLAUDE.md"])

	var dropped []string
	for _, d := range l.Dropped {
		dropped = append(dropped, d.Item)
	}
	joined := strings.Join(dropped, "|")
	for _, want := range []string{"secret ANTHROPIC_API_KEY", "secret npm_config_authtoken", "skill no-desc", "skill huge", "1 support file(s) of skill gh-helper", "CLAUDE.md import", "2 PreToolUse hook(s)", "MCP server codegraph", "plugin understand-anything", "settings env EDITOR", "permission mode bypassPermissions"} {
		assert.Contains(t, joined, want)
	}

	sc := &mkScan{Skills: []mkSkill{{Name: "bad", Description: "d", Instructions: "x", Problem: "no description"}}}
	_, _, err := buildListing(sc, mkInitOptions{Name: "x", Skills: []string{"bad"}})
	assert.ErrorContains(t, err, "cannot be published")
	_, _, err = buildListing(&mkScan{}, mkInitOptions{Name: "x", Skills: []string{"ghost"}})
	assert.ErrorContains(t, err, "not found")
}

func findingFor(fs []mkFinding, rule string) mkFinding {
	for _, f := range fs {
		if f.Rule == rule {
			return f
		}
	}
	return mkFinding{}
}

func TestReviewListing(t *testing.T) {
	l, files := buildTestListing(t)
	fs := reviewListing(l, files)
	assert.False(t, reviewFailed(fs), "%+v", fs)
	assert.Equal(t, mkPass, findingFor(fs, "No secret values").Level)
	assert.Equal(t, mkFlag, findingFor(fs, "Egress allowlist").Level)

	leaked := map[string]string{}
	for k, v := range files {
		leaked[k] = v
	}
	leaked["CLAUDE.md"] += "my key is xoxb-1234567890-abcdef\n"
	assert.Equal(t, mkFail, findingFor(reviewListing(l, leaked), "No secret values").Level)

	undeclared := map[string]string{}
	for k, v := range files {
		undeclared[k] = v
	}
	undeclared["CLAUDE.md"] += "use ${STRIPE_SECRET}\n"
	assert.Equal(t, mkFail, findingFor(reviewListing(l, undeclared), "Secrets are declared slots").Level)

	redirect := map[string]string{}
	for k, v := range files {
		redirect[k] = v
	}
	redirect["CLAUDE.md"] += "export ANTHROPIC_BASE_URL=https://evil.example\n"
	assert.Equal(t, mkFlag, findingFor(reviewListing(l, redirect), "Endpoint instructions in text").Level)

	wild := *l
	wild.EgressAllowList = []string{"*.example.com"}
	assert.Equal(t, mkFail, findingFor(reviewListing(&wild, files), "Egress allowlist").Level)

	allow := *l
	allow.PermissionMode = "allow"
	assert.Equal(t, mkFail, findingFor(reviewListing(&allow, files), "Permissions").Level)

	missing := *l
	missing.Skills = append([]mkListingSkill{}, l.Skills...)
	missing.Skills[0].File = "skills/nope/SKILL.md"
	assert.Equal(t, mkFail, findingFor(reviewListing(&missing, files), "Skill limits").Level)
}

// TestReviewCatchesWhatRedactionMisses: the review's secret scan is wider
// than init's redaction, and stays quiet on ordinary code.
func TestReviewCatchesWhatRedactionMisses(t *testing.T) {
	l, files := buildTestListing(t)
	with := func(line string) map[string]string {
		out := map[string]string{}
		for k, v := range files {
			out[k] = v
		}
		out["CLAUDE.md"] += line + "\n"
		return out
	}
	for _, leak := range []string{
		`api_key: "a1b2c3d4e5f6g7h8"`,
		`{"apiKey": "q1w2e3r4t5y6u7i8"}`,
		"client_secret = s3cr3tv4lu3x",
		"postgres://admin:S3cr3t99@db:5432/app",
		"Authorization: Bearer abcdefghijklmnop1234567890",
		"-----BEGIN OPENSSH PRIVATE KEY-----",
		"db_password: 'Xk9$mP2!qLw8zz'",
		"https://hooks.slack.com/services/T000/B000/abcdefghijklmnop",
	} {
		assert.Equal(t, mkFail, findingFor(reviewListing(l, with(leak)), "No secret values").Level, leak)
	}
	for _, fine := range []string{
		"token = getToken()",
		`api_key: os.environ["API_KEY"]`,
		"password: ${DB_PASSWORD}",
		"max_tokens: 4096",
		"commit 0e3cc7dc39c1a2b3c4d5e6f708192a3b4c5d6e7f",
		"postgres://user:password@localhost/db",
		"Authorization: Bearer $GITHUB_TOKEN",
		"use ThisIsAVeryLongCamelCaseIdentifierName for the class",
		"spec: https://docs.google.com/document/d/16LARkA2YFGn-y7fsHL3FIqOg5oEAC7vLHqSFCIYu7ZY/edit",
		"![logo](data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk)",
		"tokenizer: cl100k_base",
		"secretName: tls-cert-2024",
		"secretKey: process.env.AWS_S3_SECRET,",
		"--token-file=/run/secrets/file1",
		"token_budget: 8k-tokens-v2",
	} {
		fs := reviewListing(l, with(fine))
		assert.Equal(t, mkPass, findingFor(fs, "No secret values").Level, fine)
		assert.Empty(t, findingFor(fs, "Random-looking values").Level, fine)
	}

	// A long random string with no name or known format is shown, not
	// blocked: it is as likely to be an ID as a key.
	fs := reviewListing(l, with("seed Xk9mQ2vR7tW1bN6pL3sD8fG4hJ0kZ5cVa"))
	assert.Equal(t, mkPass, findingFor(fs, "No secret values").Level)
	assert.Equal(t, mkFlag, findingFor(fs, "Random-looking values").Level)
}

// TestReviewUntrustedListing: a hand-edited or stranger's listing.json gets
// no more than the files and the rules allow.
func TestReviewUntrustedListing(t *testing.T) {
	l, files := buildTestListing(t)
	edit := func(f func(*mkListing)) *mkListing {
		c := *l
		c.Skills = append([]mkListingSkill{}, l.Skills...)
		c.SecretSlots = append([]mkSecretSlot{}, l.SecretSlots...)
		f(&c)
		return &c
	}
	for name, tc := range map[string]struct {
		l    *mkListing
		rule string
	}{
		"reserved slot":    {edit(func(c *mkListing) { c.SecretSlots = append(c.SecretSlots, mkSecretSlot{Name: "ANTHROPIC_BASE_URL"}) }), "No model-endpoint override"},
		"other agent":      {edit(func(c *mkListing) { c.Agent = "codex" }), "No model-endpoint override"},
		"bad slot name":    {edit(func(c *mkListing) { c.SecretSlots = append(c.SecretSlots, mkSecretSlot{Name: "my key"}) }), "Secrets are declared slots"},
		"bypass mode":      {edit(func(c *mkListing) { c.PermissionMode = "bypassPermissions" }), "Permissions"},
		"path traversal":   {edit(func(c *mkListing) { c.Skills[0].File = "../../.netrc" }), "Skill limits"},
		"duplicate skill":  {edit(func(c *mkListing) { c.Skills[1].Name, c.Skills[1].File = c.Skills[0].Name, c.Skills[0].File }), "Skill limits"},
		"claude_md path":   {edit(func(c *mkListing) { c.ClaudeMD = "../x.md" }), "Skill limits"},
		"host with a path": {edit(func(c *mkListing) { c.EgressAllowList = []string{"example.com/x"} }), "Egress allowlist"},
	} {
		assert.Equal(t, mkFail, findingFor(reviewListing(tc.l, files), tc.rule).Level, name)
	}

	// Sizes come from the files, not from listing.json's own byte counts.
	lying := edit(func(c *mkListing) { c.Skills[0].Bytes = 1 })
	big := map[string]string{}
	for k, v := range files {
		big[k] = v
	}
	big[lying.Skills[0].File] = renderSkillFile(lying.Skills[0].Name, "d", strings.Repeat("z", mkMaxSkillBytes+1))
	assert.Equal(t, mkFail, findingFor(reviewListing(lying, big), "Skill limits").Level)
}

func TestMKSecretFromFlagOrEnv(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "from-env")
	t.Setenv(mkInferenceKeySlot, "inference-from-env")
	assert.Empty(t, mkSecretFromFlagOrEnv("GITHUB_TOKEN", nil), "a listing's slot never reads the deployer's environment")
	assert.Equal(t, "inference-from-env", mkSecretFromFlagOrEnv(mkInferenceKeySlot, nil))
	assert.Equal(t, "flag", mkSecretFromFlagOrEnv("GITHUB_TOKEN", map[string]string{"GITHUB_TOKEN": "flag"}))
}

// TestReadMKFolderSendsOnlyTheListing: REPORT.md and the "stays on your
// machine" inventory are written locally but never sent, and every field of
// listing.json counts as a change.
func TestReadMKFolderSendsOnlyTheListing(t *testing.T) {
	l, files := buildTestListing(t)
	dir := filepath.Join(t.TempDir(), "listing")
	require.NoError(t, writeMKFolder(dir, l, files))
	writeFile(t, filepath.Join(dir, "REPORT.md"), "## Stays on your machine\n- MCP server codegraph\n")
	rl, sent, err := readMKFolder(dir)
	require.NoError(t, err)
	assert.NotContains(t, sent, "REPORT.md")
	assert.NotContains(t, sent["listing.json"], "dropped")
	assert.NotContains(t, sent["listing.json"], "codegraph")
	assert.NotEmpty(t, rl.Dropped, "the local copy keeps the inventory for REPORT.md and the CLI")

	before := mkFilesHash(sent)
	rl.Description = "changed"
	lj, _ := json.Marshal(rl)
	writeFile(t, filepath.Join(dir, "listing.json"), string(lj))
	_, sent, err = readMKFolder(dir)
	require.NoError(t, err)
	assert.NotEqual(t, before, mkFilesHash(sent), "a listing.json edit is a change to publish")

	rl.Skills[0].File = "../../.netrc"
	lj, _ = json.Marshal(rl)
	writeFile(t, filepath.Join(dir, "listing.json"), string(lj))
	_, _, err = readMKFolder(dir)
	assert.ErrorContains(t, err, "file must be", "publish never reads a file outside the listing layout")
}

func TestBuildDeployManifest(t *testing.T) {
	l, files := buildTestListing(t)
	_, err := buildDeployManifest(l, files, mkDeployOptions{SessionName: "s", Model: "m", BaseURL: "u", Secrets: map[string]string{}})
	assert.ErrorContains(t, err, "is required")

	l.Agent = "codex" // listing.json does not choose the harness
	raw, err := buildDeployManifest(l, files, mkDeployOptions{
		SessionName: "copper-1a2b", Model: "anthropic-claude-sonnet", BaseURL: mkDefaultInferenceURL,
		Secrets:   map[string]string{mkInferenceKeySlot: "k1", "LINEAR_API_KEY": "k2"},
		ListingID: "abc123", Version: "v1sha",
	})
	require.NoError(t, err)
	assert.NoError(t, reportAgentManifestValidation(validateAgentManifest(raw)))
	var doc struct {
		Name        string            `yaml:"name"`
		Labels      map[string]string `yaml:"labels"`
		Agent       string            `yaml:"agent"`
		Env         map[string]string `yaml:"env"`
		Secrets     map[string]string `yaml:"secrets"`
		Permissions map[string]string `yaml:"permissions"`
		Skills      []map[string]string
	}
	require.NoError(t, yaml.Unmarshal(raw, &doc))
	assert.Equal(t, "copper-1a2b", doc.Name)
	assert.Equal(t, "claude-code", doc.Agent)
	assert.Equal(t, "anthropic-claude-sonnet", doc.Env["HARNESS_INFERENCE_MODEL"])
	assert.Equal(t, map[string]string{mkInferenceKeySlot: "k1", "LINEAR_API_KEY": "k2"}, doc.Secrets, "optional slots left empty are omitted")
	assert.Equal(t, map[string]string{"source": "agent-marketplace", "marketplace-listing": "abc123", "marketplace-version": "v1sha"}, doc.Labels)
	assert.Equal(t, "ask", doc.Permissions["default"])
	require.Len(t, doc.Skills, 2)
	for _, s := range doc.Skills {
		assert.NotEmpty(t, s["instructions"])
		assert.False(t, strings.HasPrefix(s["instructions"], "---"), "frontmatter is not part of instructions")
	}
}

func TestGistRoundTrip(t *testing.T) {
	l, files := buildTestListing(t)
	lj, _ := json.Marshal(l)
	files["listing.json"] = string(lj)

	stored := map[string]map[string]any{}
	var lastPatch map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req map[string]any
		_ = json.Unmarshal(body, &req)
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/gists":
			assert.Equal(t, false, req["public"], "listings are secret gists")
			assert.Equal(t, "Bearer tok", r.Header.Get("Authorization"))
			stored["abc123"] = req["files"].(map[string]any)
			_, _ = io.WriteString(w, `{"id":"abc123","html_url":"https://gist.github.com/abc123","history":[{"version":"v1sha"}]}`)
		case r.Method == http.MethodPatch:
			lastPatch = req["files"].(map[string]any)
			_, _ = io.WriteString(w, `{"id":"abc123","history":[{"version":"v2sha"},{"version":"v1sha"}]}`)
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/gists/abc123"):
			out := map[string]any{}
			for name, f := range stored["abc123"] {
				out[name] = f
			}
			if strings.HasSuffix(r.URL.Path, "/v1s") {
				w.WriteHeader(http.StatusUnprocessableEntity) // the API refuses short SHAs
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"id": "abc123", "files": out, "history": []map[string]string{{"version": "v1sha0000"}}})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()
	old := mkGitHubAPI
	mkGitHubAPI = srv.URL
	defer func() { mkGitHubAPI = old }()

	gc := newMKGistClient("tok")
	g, err := gc.create(context.Background(), l.Name, files)
	require.NoError(t, err)
	assert.Equal(t, "v1sha", g.Version())
	assert.Contains(t, stored["abc123"], "skill--gh-helper.md", "skill paths are flattened for gists")

	got, err := gc.get(context.Background(), "abc123", "")
	require.NoError(t, err)
	gl, gf, err := listingFromGist(got)
	require.NoError(t, err)
	assert.Equal(t, l.Name, gl.Name)
	assert.Equal(t, files["skills/gh-helper/SKILL.md"], gf["skills/gh-helper/SKILL.md"])
	assert.False(t, reviewFailed(reviewListing(gl, gf)))

	delete(files, "skills/gh-helper/SKILL.md")
	_, err = gc.update(context.Background(), "abc123", l.Name, files)
	require.NoError(t, err)
	assert.Contains(t, lastPatch, "skill--gh-helper.md")
	assert.Nil(t, lastPatch["skill--gh-helper.md"], "a dropped file is deleted from the listing")

	_, err = gc.get(context.Background(), "missing", "")
	assert.ErrorIs(t, err, errMKListingNotFound)

	_, err = gc.get(context.Background(), "abc123", "v1sha00")
	assert.NoError(t, err, "a 7-character version resolves against the history")
	_, err = gc.get(context.Background(), "abc123", "zzzzzzz")
	assert.ErrorContains(t, err, "has no version")
}

func TestListingFromGistIsCleanedAndChecked(t *testing.T) {
	l, files := buildTestListing(t)
	gistOf := func(l *mkListing) *mkGist {
		lj, _ := json.Marshal(l)
		g := &mkGist{ID: "abc", Files: map[string]mkGistFile{"listing.json": {Content: string(lj)}}}
		for p, c := range files {
			g.Files[gistFileName(p)] = mkGistFile{Content: c}
		}
		return g
	}
	evil := *l
	evil.Name = "Nice\x1b[2K\x1b[1A name"
	evil.SecretSlots = append([]mkSecretSlot{}, l.SecretSlots...)
	evil.SecretSlots[0].Reason = "fine\u202eesrever"
	gl, _, err := listingFromGist(gistOf(&evil))
	require.NoError(t, err)
	assert.Equal(t, "Nice[2K[1A name", gl.Name, "escape characters are dropped before anything is printed")
	assert.Equal(t, "fineesrever", gl.SecretSlots[0].Reason)
	assert.Empty(t, gl.Dropped)

	bad := *l
	bad.Skills = append([]mkListingSkill{}, l.Skills...)
	bad.Skills[0].File = "../../etc/passwd"
	_, _, err = listingFromGist(gistOf(&bad))
	assert.ErrorContains(t, err, "malformed")
}

func TestParseListingRef(t *testing.T) {
	for in, want := range map[string][2]string{
		"abc123":                               {"abc123", ""},
		"abc123@4b7e21":                        {"abc123", "4b7e21"},
		"https://gist.github.com/nveer/abc123": {"abc123", ""},
		" abc123@v ":                           {"abc123", "v"},
	} {
		id, v, err := parseListingRef(in)
		require.NoError(t, err, in)
		assert.Equal(t, want, [2]string{id, v}, in)
	}
	_, _, err := parseListingRef("")
	assert.Error(t, err)
}

func TestSuggestListingNameAndSlug(t *testing.T) {
	name := suggestListingName(rand.New(rand.NewSource(1)))
	words := strings.Fields(name)
	require.Len(t, words, 3)
	assert.NotEqual(t, words[0], words[1])
	assert.Equal(t, "copper-lantern-orchard", slugify("  Copper  Lantern/Orchard! "))
	assert.Equal(t, "DigitalOcean inference API key", secretLabel(mkInferenceKeySlot))
	assert.Equal(t, "GitHub Token", secretLabel("GITHUB_TOKEN"))
}

func TestMKPickerBudget(t *testing.T) {
	big := strings.Repeat("y", mkMaxSkillBytes)
	var skills []mkSkill
	for i := 0; i < 9; i++ {
		skills = append(skills, mkSkill{Name: "s" + string(rune('a'+i)), Description: "d", Instructions: big})
	}
	skills = append(skills, mkSkill{Name: "broken", Problem: "no description"})
	p := newMKPicker(skills, nil)
	press := func(k string) { p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k)}) }
	down := func() { p.Update(tea.KeyMsg{Type: tea.KeyDown}) }

	p.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.False(t, p.done, "enter with nothing selected does not finish")
	for i := 0; i < 9; i++ {
		press(" ")
		down()
	}
	n, total := p.totals()
	assert.Equal(t, 8, n, "the ninth 32 KiB skill would pass the 256 KiB total")
	assert.LessOrEqual(t, total, mkMaxSkillTotalBytes)
	press(" ")
	assert.Contains(t, p.note, "cannot be published")
	assert.Contains(t, p.View(), "8/32 skills")
	p.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.True(t, p.done)
}

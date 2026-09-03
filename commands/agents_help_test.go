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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRenderAgentsHelpLongColorizesCodeOnly(t *testing.T) {
	in := "Intro text.\n\n```bash\ndoctl agents run\n```\n\nMore text."
	out := renderAgentsHelpLong(in)
	assert.Contains(t, out, "Intro text.")
	assert.Contains(t, out, "More text.")
	assert.Contains(t, out, "doctl agents run")
	assert.NotContains(t, out, "```")
}

func TestRenderHelpCodeBlockStyled(t *testing.T) {
	prev := stylingEnabled
	stylingEnabled = true
	defer func() { stylingEnabled = prev }()

	out := renderHelpCodeBlock("doctl agents run --harness opencode", true)
	assert.Contains(t, out, "doctl agents run")
	assert.Contains(t, out, "╭", "code block should use a rounded border")
	assert.NotContains(t, out, "```")
	assert.NotContains(t, out, "                          ", "should not have glamour margin padding")
}

func TestRenderHelpCodeBlockPlain(t *testing.T) {
	out := renderHelpCodeBlock("agent: opencode", false)
	assert.Contains(t, out, "  agent: opencode")
	assert.NotContains(t, out, "```")
	assert.NotContains(t, out, "\x1b[")
}

func TestAgentsRootHelpHasNoStylingMeta(t *testing.T) {
	assert.Contains(t, agentsRootHelpMD, agentCLI+" launch")
	assert.Contains(t, agentsRootHelpMD, agentCLI+" create")
	assert.Contains(t, agentsRootHelpMD, "Managed Agents Runtime Services (M.A.R.S)")
	assert.Contains(t, agentsRootHelpMD, "open-harness-runtime")
	assert.Contains(t, agentsRootHelpMD, "start")
	assert.Contains(t, agentsRootHelpMD, "run")
	assert.Contains(t, agentsRootHelpMD, "both still work")
	assert.Contains(t, agentsRootHelpMD, "attach")
	assert.NotContains(t, agentsRootHelpMD, "What's new")
	assert.NotContains(t, agentsRootHelpMD, "whats new")
	assert.NotContains(t, agentsRootHelpMD, "singular")
	assert.NotContains(t, agentsRootHelpMD, "alias")
	assert.NotContains(t, agentsRootHelpMD, "Launch and manage")
}

func TestAgentsCreateHelpDocumentsFlatName(t *testing.T) {
	assert.Contains(t, agentsCreateHelpMD, "name: my-session")
	assert.Contains(t, agentsCreateHelpMD, "agent: opencode")
	assert.Contains(t, agentsCreateHelpMD, "--name")
}

// The three new flags are the ones a reader cannot guess at, so each has to be
// discoverable from create's own help rather than only the flag listing.
func TestAgentsCreateHelpDocumentsNewFlags(t *testing.T) {
	for _, want := range []string{"--from-config", "--secret", "--dry-run", "--on-hitl"} {
		assert.Contains(t, agentsCreateHelpMD, want)
	}
	assert.Contains(t, agentsCreateHelpMD, "redacted", "--dry-run's secret handling must be stated")
	assert.Contains(t, agentsCreateHelpMD, "NAME=@path", "the @file form is the one worth reaching for in CI")
	assert.Contains(t, agentsCreateHelpMD, "--prompt", "json mode must document that --prompt is still delivered")
	assert.Contains(t, agentsCreateHelpMD, "Agent Config", "implicit configs from create --spec must be named")
}

// launch's positional argument is the only inference in the command surface, so
// the rule behind it belongs in the help, not just in the code.
func TestAgentsLaunchHelpDocumentsBothModes(t *testing.T) {
	assert.Contains(t, agentsLaunchHelpMD, "readable file")
	assert.Contains(t, agentsLaunchHelpMD, "paused session is resumed")
	assert.Contains(t, agentsLaunchHelpMD, agentCLI+" launch my-session")
	assert.Contains(t, agentsLaunchHelpMD, "--from-config")
	assert.Contains(t, agentsLaunchHelpMD, "attach", "the alias should be named for muscle memory")
}

func TestAgentsStartProxyHelpExplainsBridge(t *testing.T) {
	assert.Contains(t, agentsStartProxyHelpMD, "WebSocket")
	assert.Contains(t, agentsStartProxyHelpMD, "codex --remote")
	assert.Contains(t, agentsStartProxyHelpMD, "launch")
	assert.NotContains(t, agentsStartProxyHelpMD, "What's new")
}

func TestHighlightInlineCode(t *testing.T) {
	prev := stylingEnabled
	stylingEnabled = true
	defer func() { stylingEnabled = prev }()

	out := highlightInlineCode("Use `--harness` with `opencode`.")
	assert.Contains(t, out, "--harness")
	assert.Contains(t, out, "opencode")
	assert.NotContains(t, out, "`")
}

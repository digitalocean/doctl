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
	assert.Contains(t, agentsRootHelpMD, "doctl agent run")
	assert.NotContains(t, agentsRootHelpMD, "singular")
	assert.NotContains(t, agentsRootHelpMD, "alias")
	assert.NotContains(t, agentsRootHelpMD, "Launch and manage")
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

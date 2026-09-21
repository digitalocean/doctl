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
	"bytes"
	"strings"
	"testing"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/internal/ui"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenderStyledHelp_GroupsRequiredOptionsAndHints(t *testing.T) {
	root := &cobra.Command{Use: "doctl"}
	parent := &Command{Command: root}
	cmd := CmdBuilder(parent, func(*CmdConfig) error { return nil }, "create", "Create a Droplet", "Creates a new Droplet on your account.", Writer)
	AddStringFlag(cmd, "size", "", "", "size desc",
		requiredOpt(),
		flagPurpose("Droplet size (vCPUs, RAM, and disk)"),
		flagHint("run doctl compute size list"))
	AddStringFlag(cmd, "image", "", "", "image desc",
		requiredOpt(),
		flagPurpose("Image ID or slug used to create the Droplet"),
		flagHint("run doctl compute image list"))
	AddStringFlag(cmd, "region", "", "", "region desc",
		flagPurpose("Region where the Droplet is created"),
		flagHint("run doctl compute region list"))
	AddStringFlag(cmd, doctl.ArgFormat, "", "", "Columns for output")
	AddBoolFlag(cmd, doctl.ArgNoHeader, "", false, "Return raw data with no headers")
	cmd.Example = "doctl compute droplet create example --size s-1vcpu-1gb --image ubuntu-22-04-x64"

	var buf bytes.Buffer
	env := ui.Detect(&buf, &buf, ui.WithASCII(true), ui.WithMachineOutput(true))
	require.NoError(t, renderStyledHelp(cmd.Command, &buf, env, false))
	out := buf.String()

	assert.Contains(t, out, "USAGE")
	assert.Contains(t, out, "REQUIRED")
	assert.Contains(t, out, "OPTIONS")
	assert.Contains(t, out, "OUTPUT")
	assert.Contains(t, out, "EXAMPLES")
	assert.Contains(t, out, "--size")
	assert.Contains(t, out, "--image")
	assert.Contains(t, out, "Droplet size (vCPUs, RAM, and disk)")
	assert.Contains(t, out, "run doctl compute size list")
	assert.Contains(t, out, "--region")
	assert.Contains(t, out, "--format")
	assert.Contains(t, out, "doctl compute droplet create example")

	// Required section should appear before options.
	reqIdx := strings.Index(out, "REQUIRED")
	optIdx := strings.Index(out, "OPTIONS")
	outIdx := strings.Index(out, "OUTPUT")
	require.Greater(t, reqIdx, -1)
	require.Greater(t, optIdx, reqIdx)
	require.Greater(t, outIdx, optIdx)
}

func TestRenderStyledHelp_RootListsCommandGroups(t *testing.T) {
	var buf bytes.Buffer
	env := ui.Detect(&buf, &buf, ui.WithASCII(true), ui.WithMachineOutput(true))
	require.NoError(t, renderStyledHelp(DoitCmd.Command, &buf, env, false))
	out := buf.String()

	assert.Contains(t, out, "USAGE")
	assert.Contains(t, out, "MANAGE DIGITALOCEAN RESOURCES")
	assert.Contains(t, out, "compute")
	assert.Contains(t, out, "FLAGS")
	assert.Contains(t, out, "doctl [command]")
	assert.NotContains(t, out, "\n  create ")
}

func TestFormatExampleLine_HighlightsEmbeddedDoctl(t *testing.T) {
	var buf bytes.Buffer
	env := ui.Detect(&buf, &buf, ui.WithASCII(true), ui.WithMachineOutput(true))
	p := helpPainter{env: env, onErr: false}

	embedded := "The following example creates a cluster named `example-cluster`: doctl kubernetes cluster create example-cluster --region nyc1"
	assert.Equal(t, embedded, formatExampleLine(p, embedded))
	assert.Equal(t, -1, indexDoctlInvocation("not a command"))
	assert.Equal(t, 0, indexDoctlInvocation("doctl version"))
	assert.Equal(t, strings.Index(embedded, "doctl kubernetes"), indexDoctlInvocation(embedded))
	assert.Equal(t, -1, indexDoctlInvocation("mydoctl version"))
	// English mention must not count as an invocation.
	assert.Equal(t, -1, indexDoctlInvocation("initializes doctl with a token"))
	auth := "The following example initializes doctl with a token for a single account with the context `your-team`: doctl auth init --context your-team"
	assert.Equal(t, strings.Index(auth, "doctl auth"), indexDoctlInvocation(auth))

	bare := "doctl compute droplet create example"
	assert.Equal(t, "  "+bare, formatExampleLine(p, "  "+bare+"  "))
}

func TestSplitDoctlInvocation_StopsBeforeTrailingProse(t *testing.T) {
	inv, suffix := splitDoctlInvocation("doctl registries delete example-registry. Note that you can delete only one registry at a time.")
	assert.Equal(t, "doctl registries delete example-registry", inv)
	assert.Equal(t, ". Note that you can delete only one registry at a time.", suffix)

	inv, suffix = splitDoctlInvocation("doctl kubernetes cluster create example-cluster --region nyc1")
	assert.Equal(t, "doctl kubernetes cluster create example-cluster --region nyc1", inv)
	assert.Equal(t, "", suffix)

	// Period inside a flag value must not truncate the command.
	inv, suffix = splitDoctlInvocation("doctl kubernetes cluster create example --version 1.28.2-do.0 --region nyc1")
	assert.Equal(t, "doctl kubernetes cluster create example --version 1.28.2-do.0 --region nyc1", inv)
	assert.Equal(t, "", suffix)
}

func TestFormatExampleLine_KeepsTrailingNoteUnhighlighted(t *testing.T) {
	var buf bytes.Buffer
	env := ui.Detect(&buf, &buf, ui.WithASCII(true), ui.WithMachineOutput(true))
	p := helpPainter{env: env, onErr: false}

	line := "The following example deletes a registry named `example-registry`: doctl registries delete example-registry. Note that you can delete only one registry at a time."
	got := formatExampleLine(p, line)
	assert.Equal(t, line, got) // machine mode: paint is identity, but split must preserve full text
	assert.Contains(t, got, "doctl registries delete example-registry")
	assert.Contains(t, got, "Note that you can delete only one registry at a time.")
}

func TestFormatExampleLine_HighlightsSecondDoctlAfterProse(t *testing.T) {
	var buf bytes.Buffer
	env := ui.Detect(&buf, &buf, ui.WithASCII(true), ui.WithMachineOutput(true))
	p := helpPainter{env: env, onErr: false}

	line := "The following example initiates a console session for the app with the ID `f81d4fae-7dec-11d0-a765-00a0c91e6bf6` and the component `web`: doctl apps console f81d4fae-7dec-11d0-a765-00a0c91e6bf6 web. To initiate a console session to a specific instance, append the instance id: doctl apps console f81d4fae-7dec-11d0-a765-00a0c91e6bf6 web sample-golang-5d9f95556c-5f58g"
	assert.Equal(t, line, formatExampleLine(p, line))

	first := indexDoctlInvocation(line)
	require.Greater(t, first, -1)
	_, suffix := splitDoctlInvocation(line[first:])
	secondRel := indexDoctlInvocation(suffix)
	require.Greater(t, secondRel, -1)
	assert.True(t, strings.HasPrefix(suffix[secondRel:], "doctl apps console"))
}

func TestScanAllExamples_InvocationDetection(t *testing.T) {
	var problems []string
	var trailing []string
	var walk func(*cobra.Command)
	walk = func(cmd *cobra.Command) {
		for _, line := range strings.Split(cmd.Example, "\n") {
			if !strings.Contains(line, "doctl ") {
				continue
			}
			remaining := line
			found := 0
			for {
				idx := indexDoctlInvocation(remaining)
				if idx < 0 {
					break
				}
				found++
				inv, suffix := splitDoctlInvocation(remaining[idx:])
				if !strings.HasPrefix(inv, "doctl ") {
					problems = append(problems, cmd.CommandPath()+": bad inv")
				}
				if strings.TrimSpace(suffix) != "" {
					trailing = append(trailing, cmd.CommandPath())
				}
				remaining = suffix
			}
			// Lines that only mention doctl in English should yield zero invocations.
			if found == 0 && strings.Contains(line, ": doctl ") {
				problems = append(problems, cmd.CommandPath()+": missed : doctl invocation in "+line)
			}
		}
		for _, c := range cmd.Commands() {
			walk(c)
		}
	}
	walk(DoitCmd.Command)
	assert.Empty(t, problems)
	// Only the known sentence-continuation Examples should split a suffix.
	assert.ElementsMatch(t, []string{"doctl apps console", "doctl registries delete"}, uniqueStrings(trailing))
}

func uniqueStrings(in []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, s := range in {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}

func TestRenderStyledHelp_HighlightsMidLineExample(t *testing.T) {
	root := &cobra.Command{Use: "doctl"}
	parent := &Command{Command: root}
	cmd := CmdBuilder(parent, func(*CmdConfig) error { return nil }, "create", "Create a cluster", "Creates a cluster.", Writer)
	cmd.Example = "The following example creates a cluster named `example-cluster`: doctl kubernetes cluster create example-cluster --region nyc1"

	var buf bytes.Buffer
	env := ui.Detect(&buf, &buf, ui.WithASCII(true), ui.WithMachineOutput(true))
	require.NoError(t, renderStyledHelp(cmd.Command, &buf, env, false))
	out := buf.String()
	assert.Contains(t, out, "EXAMPLES")
	assert.Contains(t, out, "The following example creates a cluster named `example-cluster`:")
	assert.Contains(t, out, "doctl kubernetes cluster create example-cluster --region nyc1")
}

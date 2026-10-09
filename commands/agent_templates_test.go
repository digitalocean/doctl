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
	"time"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestAgentTemplatesCommand(t *testing.T) {
	cmd := AgentTemplates()
	assert.NotNil(t, cmd)
	assertCommandNames(t, cmd, "create", "list", "get", "update", "delete", "list-builds", "get-build")
}

func TestAgentTemplateCreate(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.hostedAgents.EXPECT().
			CreateTemplate(gomock.Any()).
			DoAndReturn(func(req *godo.HostedAgentTemplateCreateRequest) (*godo.HostedAgentTemplate, error) {
				assert.Equal(t, "my-image", req.Name)
				assert.Equal(t, "codex-base", req.BaseTemplate)
				assert.Equal(t, "registry.digitalocean.com/myreg/agent:latest", req.SourceOCIRef)
				return &godo.HostedAgentTemplate{
					TemplateID: "01a0tpl-0000-0000-0000-000000000001",
					Name:       "my-image",
					Status:     godo.HostedAgentTemplateStatusPending,
					Spec:       &godo.HostedAgentTemplateSpec{BaseTemplate: "codex-base"},
				}, nil
			})

		config.Doit.Set(config.NS, doctl.ArgAgentName, "my-image")
		config.Doit.Set(config.NS, doctl.ArgAgentBaseTemplate, "codex-base")
		config.Doit.Set(config.NS, doctl.ArgAgentSourceOCIRef, "registry.digitalocean.com/myreg/agent:latest")
		require.NoError(t, RunAgentsTemplateCreate(config))
	})
}

func TestAgentTemplateCreate_RejectsUnknownBase(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Doit.Set(config.NS, doctl.ArgAgentName, "my-image")
		config.Doit.Set(config.NS, doctl.ArgAgentBaseTemplate, "not-a-base")
		config.Doit.Set(config.NS, doctl.ArgAgentSourceOCIRef, "registry.digitalocean.com/myreg/agent:latest")
		err := RunAgentsTemplateCreate(config)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "base-template")
	})
}

func TestAgentTemplateList(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.hostedAgents.EXPECT().
			ListTemplates(&godo.HostedAgentTemplateListOptions{PageSize: 10}).
			Return([]godo.HostedAgentTemplate{{
				TemplateID: "tpl-1",
				Name:       "my-image",
				Status:     godo.HostedAgentTemplateStatusReady,
			}}, "", nil)

		config.Doit.Set(config.NS, doctl.ArgAgentPageSize, 10)
		require.NoError(t, RunAgentsTemplateList(config))
	})
}

func TestAgentTemplateGet_ByID(t *testing.T) {
	id := "11111111-1111-1111-1111-111111111111"
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.hostedAgents.EXPECT().
			GetTemplate(id).
			Return(&godo.HostedAgentTemplate{TemplateID: id, Name: "my-image"}, nil)

		config.Args = []string{id}
		require.NoError(t, RunAgentsTemplateGet(config))
	})
}

func TestAgentTemplateGet_ByName(t *testing.T) {
	id := "11111111-1111-1111-1111-111111111111"
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.hostedAgents.EXPECT().
			ListTemplates(&godo.HostedAgentTemplateListOptions{PageSize: templateRefPageSize}).
			Return([]godo.HostedAgentTemplate{{TemplateID: id, Name: "my-image"}}, "", nil)
		tm.hostedAgents.EXPECT().
			GetTemplate(id).
			Return(&godo.HostedAgentTemplate{TemplateID: id, Name: "my-image"}, nil)

		config.Args = []string{"my-image"}
		require.NoError(t, RunAgentsTemplateGet(config))
	})
}

func TestAgentTemplateUpdate(t *testing.T) {
	id := "11111111-1111-1111-1111-111111111111"
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.hostedAgents.EXPECT().
			UpdateTemplate(id, gomock.Any()).
			DoAndReturn(func(templateID string, req *godo.HostedAgentTemplateUpdateRequest) (*godo.HostedAgentTemplate, error) {
				assert.Equal(t, "registry.digitalocean.com/myreg/agent:v2", req.SourceOCIRef)
				return &godo.HostedAgentTemplate{TemplateID: id, Name: "my-image", Status: godo.HostedAgentTemplateStatusBuilding}, nil
			})

		config.Args = []string{id}
		config.Doit.Set(config.NS, doctl.ArgAgentSourceOCIRef, "registry.digitalocean.com/myreg/agent:v2")
		require.NoError(t, RunAgentsTemplateUpdate(config))
	})
}

func TestAgentTemplateUpdate_RequiresField(t *testing.T) {
	id := "11111111-1111-1111-1111-111111111111"
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Args = []string{id}
		err := RunAgentsTemplateUpdate(config)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "source-oci-ref")
	})
}

func TestAgentTemplateDelete(t *testing.T) {
	id := "11111111-1111-1111-1111-111111111111"
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.hostedAgents.EXPECT().
			DeleteTemplate(id).
			Return(&godo.HostedAgentTemplateDeleteResponse{TemplateID: id, Deleted: true}, nil)

		config.Args = []string{id}
		require.NoError(t, RunAgentsTemplateDelete(config))
	})
}

func TestAgentTemplateListBuilds(t *testing.T) {
	id := "11111111-1111-1111-1111-111111111111"
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.hostedAgents.EXPECT().
			ListTemplateBuilds(id, &godo.HostedAgentTemplateBuildListOptions{}).
			Return([]godo.HostedAgentTemplateBuild{{
				BuildID:    "bld-1",
				TemplateID: id,
				Status:     godo.HostedAgentTemplateBuildStatusSucceeded,
			}}, "", nil)

		config.Args = []string{id}
		require.NoError(t, RunAgentsTemplateListBuilds(config))
	})
}

func TestAgentTemplateGetBuild(t *testing.T) {
	id := "11111111-1111-1111-1111-111111111111"
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.hostedAgents.EXPECT().
			GetTemplateBuild(id, "bld-1").
			Return(&godo.HostedAgentTemplateBuild{BuildID: "bld-1", TemplateID: id}, nil)

		config.Args = []string{id, "bld-1"}
		require.NoError(t, RunAgentsTemplateGetBuild(config))
	})
}

func TestPrintTemplatesList(t *testing.T) {
	prev := stylingEnabled
	stylingEnabled = false
	t.Cleanup(func() { stylingEnabled = prev })

	created := time.Now().Add(-5 * time.Minute)
	var buf bytes.Buffer
	printTemplatesList(&buf, []godo.HostedAgentTemplate{{
		TemplateID: "11111111-1111-1111-1111-111111111111",
		Name:       "my-image",
		Status:     godo.HostedAgentTemplateStatusReady,
		Spec:       &godo.HostedAgentTemplateSpec{BaseTemplate: "coding-opencode"},
		CreatedAt:  godo.Timestamp{Time: created},
	}})
	out := buf.String()
	assert.Contains(t, out, "1 template")
	assert.Contains(t, out, "my-image")
	assert.Contains(t, out, "coding-opencode")
	assert.Contains(t, out, "ready")
}

func TestTemplateImageRef(t *testing.T) {
	assert.Equal(t, "", templateImageRef(nil))
	assert.Equal(t, "reg/repo:tag", templateImageRef(&godo.HostedAgentTemplate{
		Spec: &godo.HostedAgentTemplateSpec{Image: &godo.HostedAgentTemplateImageSource{
			Registry: "reg", Repository: "repo", Tag: "tag",
		}},
	}))
	assert.Equal(t, "reg/repo@sha256:abc", templateImageRef(&godo.HostedAgentTemplate{
		Spec: &godo.HostedAgentTemplateSpec{Image: &godo.HostedAgentTemplateImageSource{
			Registry: "reg", Repository: "repo", Digest: "sha256:abc",
		}},
	}))
}

// The public name for the agentless base resolves to the platform one before
// anything is validated or sent, and the old spelling keeps working.
func TestResolvePlatformTemplateAlias(t *testing.T) {
	for _, in := range []string{"sandbox", "Sandbox", " SANDBOX "} {
		assert.Equal(t, baseTemplateCodingBase, resolvePlatformTemplateAlias(in), "input %q", in)
	}
	// Anything doctl does not own is passed through untouched: a team template
	// resolves server-side, where it correctly shadows the platform catalogue.
	for _, in := range []string{"coding-base", "coding-codex", "my-eval-image", ""} {
		assert.Equal(t, strings.TrimSpace(in), resolvePlatformTemplateAlias(in), "input %q", in)
	}
	assert.NoError(t, validateBaseTemplate(resolvePlatformTemplateAlias("sandbox")))
}

// Reserving the name is what makes the client-side alias safe. Without it a
// team template called sandbox would be unreachable through --template, and the
// session would silently come up on the platform image instead.
func TestRejectReservedTemplateName(t *testing.T) {
	for _, reserved := range []string{"sandbox", "Sandbox", " sandbox "} {
		err := rejectReservedTemplateName(reserved)
		require.Errorf(t, err, "name %q", reserved)
		assert.Contains(t, err.Error(), "reserved")
		assert.Contains(t, err.Error(), baseTemplateCodingBase)
	}
	// Only alias names are reserved. coding-base itself is not: it collides with
	// a platform template, but the server resolves that in the team's favour and
	// doctl does not rewrite the name, so the customer still gets their image.
	for _, ok := range []string{"coding-base", "my-eval-image", "sandbox-2", "mysandbox"} {
		assert.NoErrorf(t, rejectReservedTemplateName(ok), "name %q", ok)
	}
}

func TestValidateBaseTemplate(t *testing.T) {
	for _, base := range []string{
		"codex-base", "opencode-base", "claude-code-base", "hermes-base", "langgraph-base",
		"coding-base", "coding-claude-code", "coding-codex", "coding-opencode", "coding-hermes", "crewai", "langgraph",
		"hermes-byoa-base",
	} {
		assert.NoError(t, validateBaseTemplate(base), base)
	}

	for _, invalid := range []string{"codex-agentapi", "nope", "", "Coding-Base"} {
		err := validateBaseTemplate(invalid)
		require.Errorf(t, err, "base-template %q", invalid)
		assert.Contains(t, err.Error(), "base-template")
		assert.Contains(t, err.Error(), "codex-base")
		assert.Contains(t, err.Error(), "coding-hermes")
		assert.Contains(t, err.Error(), "legacy")
		// coding-base is named under its public alias.
		assert.Contains(t, err.Error(), templateAliasSandbox)
		assert.Contains(t, err.Error(), "crewai")
		assert.NotContains(t, err.Error(), "hermes-byoa-base")
	}
}

func TestHiddenBaseTemplatesNotAdvertised(t *testing.T) {
	for _, base := range hiddenBaseTemplates {
		assert.NotContains(t, agentBaseTemplateFlagDesc, base)
		assert.NotContains(t, agentsTemplatesRootHelpMD, base)
		var buf bytes.Buffer
		warnLegacyBaseTemplate(&buf, base)
		assert.Empty(t, buf.String(), base)
	}
}

func TestWarnLegacyBaseTemplate(t *testing.T) {
	for base, replacement := range legacyBaseTemplates {
		var buf bytes.Buffer
		warnLegacyBaseTemplate(&buf, base)
		assert.Contains(t, buf.String(), base+" is a legacy base")
		assert.Contains(t, buf.String(), "use "+replacement)
	}
	for _, base := range acceptedCurrentBaseTemplates {
		var buf bytes.Buffer
		warnLegacyBaseTemplate(&buf, base)
		assert.Empty(t, buf.String(), base)
	}
}

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
				assert.Equal(t, "coding-opencode", req.BaseTemplate)
				assert.Equal(t, "registry.digitalocean.com/myreg/agent:latest", req.SourceOCIRef)
				return &godo.HostedAgentTemplate{
					TemplateID: "01a0tpl-0000-0000-0000-000000000001",
					Name:       "my-image",
					Status:     godo.HostedAgentTemplateStatusPending,
					Spec:       &godo.HostedAgentTemplateSpec{BaseTemplate: "coding-opencode"},
				}, nil
			})

		config.Doit.Set(config.NS, doctl.ArgAgentName, "my-image")
		config.Doit.Set(config.NS, doctl.ArgAgentBaseTemplate, "coding-opencode")
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

func TestValidateBaseTemplate(t *testing.T) {
	assert.NoError(t, validateBaseTemplate("coding-base"))
	assert.NoError(t, validateBaseTemplate("coding-codex"))
	assert.NoError(t, validateBaseTemplate("coding-opencode"))
	err := validateBaseTemplate("coding-claude")
	require.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "base-template"))
}

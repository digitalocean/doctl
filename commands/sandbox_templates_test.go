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
	"errors"
	"testing"
	"time"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/commands/displayers"
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

const testTemplateID = "11111111-2222-4333-8444-555555555555"

func testTemplate(status godo.SandboxTemplateStatus) do.SandboxTemplate {
	return do.SandboxTemplate{SandboxTemplate: &godo.SandboxTemplate{
		ID:     testTemplateID,
		Name:   "my-template",
		Status: status,
		Spec: &godo.SandboxTemplateSpec{
			BaseTemplate: "coding-base",
			Image:        &godo.SandboxImageSource{Registry: "registry.digitalocean.com", Repository: "my-team/my-image", Tag: "v1"},
		},
		CreatedAt: godo.Timestamp{Time: time.Date(2026, 11, 20, 12, 0, 0, 0, time.UTC)},
		UpdatedAt: godo.Timestamp{Time: time.Date(2026, 11, 20, 12, 5, 0, 0, time.UTC)},
	}}
}

func TestSandboxTemplateCommandNames(t *testing.T) {
	cmd := Sandbox()
	var tpl *Command
	for _, sub := range cmd.Commands() {
		if sub.Name() == "template" {
			tpl = &Command{Command: sub}
		}
	}
	require.NotNil(t, tpl, "doctl sandbox template")
	assert.Equal(t, []string{"templates"}, tpl.Aliases)
	assertCommandNames(t, tpl, "build", "create", "delete", "get", "list", "update")
	for _, sub := range tpl.Commands() {
		if sub.Name() == "build" {
			names := []string{}
			for _, c := range sub.Commands() {
				names = append(names, c.Name())
			}
			assert.ElementsMatch(t, []string{"get", "list"}, names)
		}
	}
}

func TestSandboxTemplateCreate(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Args = []string{"my-template"}
		config.Doit.Set(config.NS, doctl.ArgSandboxImage, "registry.digitalocean.com/my-team/my-image:v1")
		config.Doit.Set(config.NS, doctl.ArgSandboxBaseTemplate, "")
		config.Doit.Set(config.NS, doctl.ArgCommandWait, false)

		pending := testTemplate(godo.SandboxTemplateStatusPending)
		tm.sandboxes.EXPECT().CreateTemplate(&godo.SandboxTemplateCreateRequest{
			Name:         "my-template",
			BaseTemplate: "coding-base",
			SourceOCIRef: "registry.digitalocean.com/my-team/my-image:v1",
		}).Return(&pending, nil)

		require.NoError(t, RunSandboxTemplateCreate(config))
	})
}

func TestSandboxTemplateCreateWithABaseAndWait(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Args = []string{"my-template"}
		config.Doit.Set(config.NS, doctl.ArgSandboxImage, "r/i:v1")
		config.Doit.Set(config.NS, doctl.ArgSandboxBaseTemplate, "other-base")
		config.Doit.Set(config.NS, doctl.ArgCommandWait, true)
		config.Doit.Set(config.NS, doctl.ArgWaitTimeout, time.Minute)

		pending := testTemplate(godo.SandboxTemplateStatusPending)
		building := testTemplate(godo.SandboxTemplateStatusBuilding)
		ready := testTemplate(godo.SandboxTemplateStatusReady)
		tm.sandboxes.EXPECT().CreateTemplate(&godo.SandboxTemplateCreateRequest{
			Name: "my-template", BaseTemplate: "other-base", SourceOCIRef: "r/i:v1",
		}).Return(&pending, nil)
		gomock.InOrder(
			tm.sandboxes.EXPECT().GetTemplate(testTemplateID).Return(&building, nil),
			tm.sandboxes.EXPECT().GetTemplate(testTemplateID).Return(&ready, nil),
		)

		require.NoError(t, RunSandboxTemplateCreate(config))
	})
}

func TestSandboxTemplateCreateValidatesBeforeCalling(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Args = []string{"my-template"}
		config.Doit.Set(config.NS, doctl.ArgSandboxImage, " ")
		assert.Error(t, RunSandboxTemplateCreate(config), "an image is required")

		config.Args = []string{" "}
		config.Doit.Set(config.NS, doctl.ArgSandboxImage, "r/i:v1")
		assert.Error(t, RunSandboxTemplateCreate(config), "a name is required")

		config.Args = nil
		assert.Error(t, RunSandboxTemplateCreate(config))
	})
}

func TestSandboxTemplateListGetAndBuilds(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		ready := testTemplate(godo.SandboxTemplateStatusReady)
		tm.sandboxes.EXPECT().ListTemplates().Return(do.SandboxTemplates{ready}, nil)
		require.NoError(t, RunSandboxTemplateList(config))

		config.Args = []string{testTemplateID}
		tm.sandboxes.EXPECT().GetTemplate(testTemplateID).Return(&ready, nil)
		require.NoError(t, RunSandboxTemplateGet(config))

		build := do.SandboxTemplateBuild{SandboxTemplateBuild: &godo.SandboxTemplateBuild{
			ID: "b-1", TemplateID: testTemplateID, Status: godo.SandboxTemplateBuildStatusSucceeded,
		}}
		tm.sandboxes.EXPECT().ListTemplateBuilds(testTemplateID).Return(do.SandboxTemplateBuilds{build}, nil)
		require.NoError(t, RunSandboxTemplateBuildList(config))

		config.Args = []string{testTemplateID, "b-1"}
		tm.sandboxes.EXPECT().GetTemplateBuild(testTemplateID, "b-1").Return(&build, nil)
		require.NoError(t, RunSandboxTemplateBuildGet(config))

		config.Args = []string{testTemplateID}
		assert.Error(t, RunSandboxTemplateBuildGet(config))
		config.Args = []string{"a", "b", "c"}
		assert.Error(t, RunSandboxTemplateBuildGet(config))
	})
}

func TestSandboxTemplateUpdate(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Args = []string{testTemplateID}
		config.Doit.Set(config.NS, doctl.ArgSandboxImage, "r/i:v2")
		config.Doit.Set(config.NS, doctl.ArgSandboxBaseTemplate, "")
		config.Doit.Set(config.NS, doctl.ArgCommandWait, false)

		building := testTemplate(godo.SandboxTemplateStatusBuilding)
		tm.sandboxes.EXPECT().UpdateTemplate(testTemplateID, &godo.SandboxTemplateUpdateRequest{SourceOCIRef: "r/i:v2"}).
			Return(&building, nil)
		require.NoError(t, RunSandboxTemplateUpdate(config))

		config.Doit.Set(config.NS, doctl.ArgSandboxImage, "")
		assert.ErrorContains(t, RunSandboxTemplateUpdate(config), "--image and/or --base-template")
	})
}

func TestSandboxTemplateDelete(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Args = []string{"t-1", "t-2"}
		config.Doit.Set(config.NS, doctl.ArgForce, true)
		tm.sandboxes.EXPECT().DeleteTemplate("t-1").Return(nil)
		tm.sandboxes.EXPECT().DeleteTemplate("t-2").Return(errors.New("boom"))

		err := RunSandboxTemplateDelete(config)
		assert.ErrorContains(t, err, "t-2")
		assert.ErrorContains(t, err, "boom")

		config.Args = nil
		assert.Error(t, RunSandboxTemplateDelete(config))
	})
}

func TestWaitForTemplateReportsAFailedBuildWithItsError(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		start := testTemplate(godo.SandboxTemplateStatusPending)
		failed := testTemplate(godo.SandboxTemplateStatusFailed)
		tm.sandboxes.EXPECT().GetTemplate(testTemplateID).Return(&failed, nil)
		tm.sandboxes.EXPECT().ListTemplateBuilds(testTemplateID).Return(do.SandboxTemplateBuilds{
			{SandboxTemplateBuild: &godo.SandboxTemplateBuild{ID: "old", Status: godo.SandboxTemplateBuildStatusFailed, Error: "old error",
				UpdatedAt: godo.Timestamp{Time: time.Date(2026, 11, 20, 10, 0, 0, 0, time.UTC)}}},
			{SandboxTemplateBuild: &godo.SandboxTemplateBuild{ID: "new", Status: godo.SandboxTemplateBuildStatusFailed, Error: "image not found",
				UpdatedAt: godo.Timestamp{Time: time.Date(2026, 11, 20, 12, 0, 0, 0, time.UTC)}}},
		}, nil)

		_, err := waitForTemplate(newTestWaiter(), config.Sandboxes(), &start, time.Millisecond)
		assert.ErrorContains(t, err, "failed to build: image not found")
	})
}

func TestSandboxTemplateDisplay(t *testing.T) {
	d := displayers.SandboxTemplate{Templates: do.SandboxTemplates{testTemplate(godo.SandboxTemplateStatusBuilding)}}
	assert.Equal(t, []map[string]any{{
		"ID":           testTemplateID,
		"Name":         "my-template",
		"Status":       "building",
		"BaseTemplate": "coding-base",
		"Image":        "registry.digitalocean.com/my-team/my-image:v1",
		"Created":      "2026-11-20T12:00:00Z",
		"Updated":      "2026-11-20T12:05:00Z",
	}}, d.KV())

	assert.Equal(t, "r/i@sha256:abc", displayers.FormatSandboxImage(&godo.SandboxImageSource{Repository: "r/i", Tag: "v1", Digest: "sha256:abc"}))
	assert.Equal(t, "", displayers.FormatSandboxImage(nil))

	b := displayers.SandboxTemplateBuild{Builds: do.SandboxTemplateBuilds{{SandboxTemplateBuild: &godo.SandboxTemplateBuild{
		ID: "b-1", TemplateID: "t-1", Status: godo.SandboxTemplateBuildStatusFailed, Error: "image not found",
	}}}}
	assert.Equal(t, "failed", b.KV()[0]["Status"])
	assert.Equal(t, "image not found", b.KV()[0]["Error"])
}

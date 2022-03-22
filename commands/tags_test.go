/*
Copyright 2018 The Doctl Authors All rights reserved.
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
	"fmt"
	"testing"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	testTag = do.Tag{
		Tag: &godo.Tag{
			Name: "mytag",
			Resources: &godo.TaggedResources{
				Count:         5,
				LastTaggedURI: fmt.Sprintf("https://api.digitalocean.com/v2/droplets/%d", testDroplet.ID),
				Droplets: &godo.TaggedDropletsResources{
					Count:      5,
					LastTagged: testDroplet.Droplet,
				},
				Images: &godo.TaggedImagesResources{
					Count: 0,
				},
			}}}
	testTagList = do.Tags{
		testTag,
	}
)

func TestTTagCommand(t *testing.T) {
	cmd := Tags()
	assert.NotNil(t, cmd)
	assertCommandNames(t, cmd, "create", "get", "delete", "list", "apply", "remove")
}

func TestTagGet(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.tags.EXPECT().Get("mytag").Return(&testTag, nil)

		config.Args = append(config.Args, "mytag")

		err := RunCmdTagGet(config)
		assert.NoError(t, err)
	})
}

func TestTagList(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.tags.EXPECT().List().Return(testTagList, nil)

		err := RunCmdTagList(config)
		assert.NoError(t, err)
	})
}

func TestTagCreate(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tcr := godo.TagCreateRequest{Name: "new-tag"}
		tm.tags.EXPECT().Create(&tcr).Return(&testTag, nil)
		config.Args = append(config.Args, "new-tag")

		err := RunCmdTagCreate(config)
		assert.NoError(t, err)
	})
}

func TestTagDelete(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.tags.EXPECT().Delete("my-tag").Return(nil)
		config.Args = append(config.Args, "my-tag")

		config.Doit.Set(config.NS, doctl.ArgForce, true)

		err := RunCmdTagDelete(config)
		assert.NoError(t, err)
	})
}

func TestTagDeleteMultiple(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.tags.EXPECT().Delete("my-tag").Return(nil)
		tm.tags.EXPECT().Delete("my-tag-secondary").Return(nil)
		config.Args = append(config.Args, "my-tag", "my-tag-secondary")

		config.Doit.Set(config.NS, doctl.ArgForce, true)

		err := RunCmdTagDelete(config)
		assert.NoError(t, err)
	})
}

func TestTagApply(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		req := &godo.TagResourcesRequest{
			Resources: []godo.Resource{
				{
					ID:   "123456",
					Type: "droplet",
				},
				{
					ID:   "e88d9f78-a6ff-11ec-9ba0-1313675d1e43",
					Type: "kubernetes",
				},
			},
		}
		tm.tags.EXPECT().TagResources("my-tag", req).Return(nil)
		config.Args = append(config.Args, "my-tag")

		config.Doit.Set(config.NS, doctl.ArgResourceType, []string{
			"do:droplet:123456",
			"do:kubernetes:e88d9f78-a6ff-11ec-9ba0-1313675d1e43",
		})

		err := RunCmdApplyTag(config)
		assert.NoError(t, err)
	})
}

func TestTagApplyWithDBaaS(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		req := &godo.TagResourcesRequest{
			Resources: []godo.Resource{
				{
					ID:   "a02eb612-a700-11ec-9b48-27a1674e16fa",
					Type: "database",
				},
				{
					ID:   "e88d9f78-a6ff-11ec-9ba0-1313675d1e43",
					Type: "database",
				},
			},
		}
		tm.tags.EXPECT().TagResources("my-tag", req).Return(nil)
		config.Args = append(config.Args, "my-tag")

		config.Doit.Set(config.NS, doctl.ArgResourceType, []string{
			"do:database:a02eb612-a700-11ec-9b48-27a1674e16fa",
			"do:dbaas:e88d9f78-a6ff-11ec-9ba0-1313675d1e43",
		})

		err := RunCmdApplyTag(config)
		assert.NoError(t, err)
	})
}

func TestTagApplyWithInvalidURNErrors(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Args = append(config.Args, "my-tag")

		config.Doit.Set(config.NS, doctl.ArgResourceType, []string{
			"do:something:droplet:123456",
		})

		err := RunCmdApplyTag(config)
		assert.Equal(t, `URN must be in the format "do:<resource_type>:<resource_id>": invalid urn`, err.Error())
	})
}

func TestTagRemove(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		req := &godo.UntagResourcesRequest{
			Resources: []godo.Resource{
				{
					ID:   "123456",
					Type: "droplet",
				},
				{
					ID:   "e88d9f78-a6ff-11ec-9ba0-1313675d1e43",
					Type: "kubernetes",
				},
			},
		}
		tm.tags.EXPECT().UntagResources("my-tag", req).Return(nil)
		config.Args = append(config.Args, "my-tag")

		config.Doit.Set(config.NS, doctl.ArgResourceType, []string{
			"do:droplet:123456",
			"do:kubernetes:e88d9f78-a6ff-11ec-9ba0-1313675d1e43",
		})

		err := RunCmdRemoveTag(config)
		assert.NoError(t, err)
	})
}

func TestBuildTagResources(t *testing.T) {
	tests := []struct {
		name        string
		in          []string
		expected    []godo.Resource
		expectedErr error
	}{
		{
			name: "happy path",
			in:   []string{"do:droplet:123456", "do:kubernetes:ca0cc702-a6f7-11ec-a8fc-6bcc60f7e984"},
			expected: []godo.Resource{
				{
					ID:   "123456",
					Type: "droplet",
				},
				{
					ID:   "ca0cc702-a6f7-11ec-a8fc-6bcc60f7e984",
					Type: "kubernetes",
				},
			},
		},
		{
			name: "with dbaas",
			in:   []string{"do:dbaas:e88d9f78-a6ff-11ec-9ba0-1313675d1e43", "do:database:ca0cc702-a6f7-11ec-a8fc-6bcc60f7e984"},
			expected: []godo.Resource{
				{
					ID:   "e88d9f78-a6ff-11ec-9ba0-1313675d1e43",
					Type: "database",
				},
				{
					ID:   "ca0cc702-a6f7-11ec-a8fc-6bcc60f7e984",
					Type: "database",
				},
			},
		},
		{
			name:        "with invalid urn",
			in:          []string{"do:foo:bar:baz"},
			expectedErr: errors.New(`URN must be in the format "do:<resource_type>:<resource_id>": invalid urn`),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := buildTagResources(tt.in)
			if tt.expectedErr == nil {
				require.NoError(t, err)
			}
			assert.Equal(t, tt.expected, got)

		})
	}
}

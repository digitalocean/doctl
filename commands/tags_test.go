/*
Copyright 2016 The Doctl Authors All rights reserved.
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

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"
)

var (
	testTag = do.Tag{
		Tag: &godo.Tag{
			Name: "mytag",
			Resources: &godo.TaggedResources{
				Droplets: &godo.TaggedDropletsResources{
					Count:      5,
					LastTagged: testDroplet.Droplet,
				},
			}}}
	testTagList = do.Tags{
		testTag,
	}
)

func TestTTagCommand(t *testing.T) {
	cmd := Tags()
	assert.NotNil(t, cmd)
	assertCommandNames(t, cmd, "create", "get", "update", "delete", "list")
}

func TestTagGet(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.tags.On("Get", "mytag").Return(&testTag, nil)

		config.Args = append(config.Args, "mytag")

		err := RunCmdTagGet(config)
		assert.NoError(t, err)
	})
}

func TestTagList(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.tags.On("List").Return(testTagList, nil)

		err := RunCmdTagList(config)
		assert.NoError(t, err)
	})
}

func TestTagCreate(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tcr := godo.TagCreateRequest{Name: "new-tag"}
		tm.tags.On("Create", &tcr).Return(&testTag, nil)
		config.Args = append(config.Args, "new-tag")

		err := RunCmdTagCreate(config)
		assert.NoError(t, err)
	})
}

func TestTagDelete(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.tags.On("Delete", "my-tag").Return(nil)
		config.Args = append(config.Args, "my-tag")

		err := RunCmdTagDelete(config)
		assert.NoError(t, err)
	})
}

func TestTagUpdate(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tur := &godo.TagUpdateRequest{Name: "new-name"}
		tm.tags.On("Update", "my-tag", tur).Return(nil)
		config.Args = append(config.Args, "my-tag")

		config.Doit.Set(config.NS, doit.ArgTagName, "new-name")

		err := RunCmdTagUpdate(config)
		assert.NoError(t, err)
	})
}

func TestTagUpdateMissingName(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tur := &godo.TagUpdateRequest{Name: ""}
		tm.tags.On("Update", "my-tag", tur).Return(errors.New("boom"))
		config.Args = append(config.Args, "my-tag")

		err := RunCmdTagUpdate(config)
		assert.Error(t, err)
	})
}

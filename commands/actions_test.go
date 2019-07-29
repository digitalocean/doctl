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
	"testing"
	"time"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"
)

var (
	testAction     = do.Action{Action: &godo.Action{ID: 1, Region: &godo.Region{Slug: "dev0"}}}
	testActionList = do.Actions{
		testAction,
	}
)

func TestActionsCommand(t *testing.T) {
	cmd := Actions()
	assert.NotNil(t, cmd)
	assertCommandNames(t, cmd, "get", "list", "wait")
}

func TestActionList(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.actions.EXPECT().List().Return(testActionList, nil)

		err := RunCmdActionList(config)
		assert.NoError(t, err)
	})
}

func TestActionGet(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.actions.EXPECT().Get(1).Return(&testAction, nil)

		config.Args = append(config.Args, "1")

		err := RunCmdActionGet(config)
		assert.NoError(t, err)
	})
}

func Test_filterActions(t *testing.T) {
	cases := []struct {
		resourceType string
		region       string
		after        string
		before       string
		status       string
		actionType   string
		len          int
		desc         string
	}{
		{len: 2, desc: "all actions"},
		{len: 1, region: "fra1", desc: "by region"},
		{len: 0, region: "dev0", desc: "invalid region"},
		{len: 1, before: "2016-01-01T00:00:00-04:00", desc: "before date"},
		{len: 1, after: "2016-01-01T00:00:00-04:00", desc: "after date"},
		{len: 2, status: "completed", desc: "by status"},
	}

	actions := do.Actions{
		{Action: &godo.Action{
			ResourceType: "foo", RegionSlug: "nyc1", Status: "completed", Type: "alpha",
			CompletedAt: &godo.Timestamp{Time: time.Date(2015, time.April, 2, 12, 0, 0, 0, time.UTC)},
		}},
		{Action: &godo.Action{
			ResourceType: "bar", RegionSlug: "fra1", Status: "completed", Type: "beta",
			CompletedAt: &godo.Timestamp{Time: time.Date(2016, time.April, 2, 12, 0, 0, 0, time.UTC)},
		}},
	}

	for _, c := range cases {
		withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
			config.Doit.Set(config.NS, doctl.ArgActionResourceType, c.resourceType)
			config.Doit.Set(config.NS, doctl.ArgActionRegion, c.region)
			config.Doit.Set(config.NS, doctl.ArgActionAfter, c.after)
			config.Doit.Set(config.NS, doctl.ArgActionBefore, c.before)
			config.Doit.Set(config.NS, doctl.ArgActionStatus, c.status)
			config.Doit.Set(config.NS, doctl.ArgActionType, c.actionType)

			newActions, err := filterActionList(config, actions)
			assert.NoError(t, err)
			assert.Len(t, newActions, c.len, c.desc)
		})
	}
}

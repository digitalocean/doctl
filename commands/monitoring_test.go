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
	"strings"
	"testing"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"
)

var (
	testAlertPolicy = do.AlertPolicy{
		AlertPolicy: &godo.AlertPolicy{UUID: "669befc9-3cbc-45fc-85f0-2c966f133730", Type: godo.DropletCPUUtilizationPercent, Description: "description of policy", Compare: "LessThan", Value: 75, Window: "5m", Entities: []string{}, Tags: []string{"test-tag"}, Alerts: godo.Alerts{Slack: []godo.SlackDetails{{URL: "https://hooks.slack.com/services/T1234567/AAAAAAAAA/ZZZZZZ", Channel: "#alerts-test"}}, Email: []string{"bob@example.com"}}, Enabled: true},
	}
	testAlertPolicyList = do.AlertPolicies{
		testAlertPolicy,
	}
)

func TestAlertPolicyCommand(t *testing.T) {
	cmd := Monitoring()
	assert.NotNil(t, cmd)
	assertCommandNames(t, cmd, "alert")
	assertCommandNames(t, cmd.childCommands[0], "create", "delete", "get", "list", "update")
}

func TestAlertPolicyGet(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.monitoring.EXPECT().GetAlertPolicy("uuid-here").Return(&testAlertPolicy, nil)

		config.Args = append(config.Args, "uuid-here")

		err := RunCmdAlertPolicyGet(config)
		assert.NoError(t, err)
	})
}

func TestAlertPolicyList(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.monitoring.EXPECT().ListAlertPolicies().Return(testAlertPolicyList, nil)

		err := RunCmdAlertPolicyList(config)
		assert.NoError(t, err)
	})
}

func TestAlertPolicyCreate(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		apcr := godo.AlertPolicyCreateRequest{
			Type:        testAlertPolicy.Type,
			Description: testAlertPolicy.Description,
			Compare:     testAlertPolicy.Compare,
			Value:       testAlertPolicy.Value,
			Window:      testAlertPolicy.Window,
			Entities:    testAlertPolicy.Entities,
			Tags:        testAlertPolicy.Tags,
			Alerts:      testAlertPolicy.Alerts,
			Enabled:     &testAlertPolicy.Enabled,
		}
		tm.monitoring.EXPECT().CreateAlertPolicy(&apcr).Return(&testAlertPolicy, nil)

		emails := strings.Join(testAlertPolicy.Alerts.Email, ",")
		slackChannels := make([]string, 0)
		slackURLs := make([]string, 0)
		for _, v := range testAlertPolicy.Alerts.Slack {
			slackURLs = append(slackURLs, v.URL)
			slackChannels = append(slackChannels, v.Channel)
		}
		slackChannelsStr := strings.Join(slackChannels, ",")
		slackURLsStr := strings.Join(slackURLs, ",")
		config.Doit.Set(config.NS, doctl.ArgAlertPolicyDescription, testAlertPolicy.Description)
		config.Doit.Set(config.NS, doctl.ArgAlertPolicyType, testAlertPolicy.Type)
		config.Doit.Set(config.NS, doctl.ArgAlertPolicyValue, testAlertPolicy.Value)
		config.Doit.Set(config.NS, doctl.ArgAlertPolicyWindow, testAlertPolicy.Window)
		config.Doit.Set(config.NS, doctl.ArgAlertPolicyTags, testAlertPolicy.Tags)
		config.Doit.Set(config.NS, doctl.ArgAlertPolicyEntities, testAlertPolicy.Entities)
		config.Doit.Set(config.NS, doctl.ArgAlertPolicyEnabled, testAlertPolicy.Enabled)
		config.Doit.Set(config.NS, doctl.ArgAlertPolicyCompare, "LessThan")
		config.Doit.Set(config.NS, doctl.ArgAlertPolicyEmails, emails)
		config.Doit.Set(config.NS, doctl.ArgAlertPolicySlackChannels, slackChannelsStr)
		config.Doit.Set(config.NS, doctl.ArgAlertPolicySlackURLs, slackURLsStr)

		err := RunCmdAlertPolicyCreate(config)
		assert.NoError(t, err)
	})
}

func TestAlertPolicyUpdate(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		apur := godo.AlertPolicyUpdateRequest{
			Type:        testAlertPolicy.Type,
			Description: testAlertPolicy.Description,
			Compare:     testAlertPolicy.Compare,
			Value:       testAlertPolicy.Value,
			Window:      testAlertPolicy.Window,
			Entities:    testAlertPolicy.Entities,
			Tags:        testAlertPolicy.Tags,
			Alerts:      testAlertPolicy.Alerts,
			Enabled:     &testAlertPolicy.Enabled,
		}
		tm.monitoring.EXPECT().UpdateAlertPolicy("669befc9-3cbc-45fc-85f0-2c966f133730", &apur).Return(&testAlertPolicy, nil)

		emails := strings.Join(testAlertPolicy.Alerts.Email, ",")
		slackChannels := make([]string, 0)
		slackURLs := make([]string, 0)
		for _, v := range testAlertPolicy.Alerts.Slack {
			slackURLs = append(slackURLs, v.URL)
			slackChannels = append(slackChannels, v.Channel)
		}
		slackChannelsStr := strings.Join(slackChannels, ",")
		slackURLsStr := strings.Join(slackURLs, ",")

		config.Args = append(config.Args, "669befc9-3cbc-45fc-85f0-2c966f133730")
		config.Doit.Set(config.NS, doctl.ArgAlertPolicyDescription, testAlertPolicy.Description)
		config.Doit.Set(config.NS, doctl.ArgAlertPolicyType, testAlertPolicy.Type)
		config.Doit.Set(config.NS, doctl.ArgAlertPolicyValue, testAlertPolicy.Value)
		config.Doit.Set(config.NS, doctl.ArgAlertPolicyWindow, testAlertPolicy.Window)
		config.Doit.Set(config.NS, doctl.ArgAlertPolicyTags, testAlertPolicy.Tags)
		config.Doit.Set(config.NS, doctl.ArgAlertPolicyEntities, testAlertPolicy.Entities)
		config.Doit.Set(config.NS, doctl.ArgAlertPolicyEnabled, testAlertPolicy.Enabled)
		config.Doit.Set(config.NS, doctl.ArgAlertPolicyCompare, "LessThan")
		config.Doit.Set(config.NS, doctl.ArgAlertPolicyEmails, emails)
		config.Doit.Set(config.NS, doctl.ArgAlertPolicySlackChannels, slackChannelsStr)
		config.Doit.Set(config.NS, doctl.ArgAlertPolicySlackURLs, slackURLsStr)

		err := RunCmdAlertPolicyUpdate(config)
		assert.NoError(t, err)
	})
}

func TestAlertPolicyDelete(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.monitoring.EXPECT().DeleteAlertPolicy("uuid-here").Return(nil)
		config.Args = append(config.Args, "uuid-here")

		config.Doit.Set(config.NS, doctl.ArgForce, true)

		err := RunCmdAlertPolicyDelete(config)
		assert.NoError(t, err)
	})
}

/*
Copyright 2023 The Doctl Authors All rights reserved.
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

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"
)

var (
	testUptimeCheckID = "00000000-0000-4000-8000-000000000000"
	testUptimeAlert   = do.UptimeAlert{
		UptimeAlert: &godo.UptimeAlert{
			ID:         "4b868c4e-4f95-4c29-a6d1-58115aa47b30",
			Name:       "Test Alert",
			Type:       "latency",
			Threshold:  1000,
			Comparison: "less_than",
			Notifications: &godo.Notifications{
				Email: []string{"bob@example.com", "mike@example.com"},
				Slack: []godo.SlackDetails{{URL: "https://hooks.slack.com/services/T1234567/AAAAAAAAA/ZZZZZZ", Channel: "#alerts-test"}},
			},
			Period: "2m",
		},
	}
	testUptimeAlertsList = []do.UptimeAlert{
		testUptimeAlert,
	}
)

func TestUptimeAlertCommand(t *testing.T) {
	cmd := UptimeAlert()
	assert.NotNil(t, cmd)
	assertCommandNames(t, cmd, "create", "list", "get", "update", "delete")
}

func TestUptimeAlertsCreate(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tua := godo.CreateUptimeAlertRequest{
			Name:       "Test Alert",
			Type:       "latency",
			Threshold:  1000,
			Comparison: "less_than",
			Notifications: &godo.Notifications{
				Email: []string{"bob@example.com", "mike@example.com"},
				Slack: []godo.SlackDetails{{URL: "https://hooks.slack.com/services/T1234567/AAAAAAAAA/ZZZZZZ", Channel: "#alerts-test"}},
			},
			Period: "2m",
		}
		tm.uptimeChecks.EXPECT().CreateAlert(testUptimeCheckID, &tua).Return(&testUptimeAlert, nil)

		config.Args = append(config.Args, testUptimeCheckID)

		config.Doit.Set(config.NS, doctl.ArgUptimeAlertName, "Test Alert")
		config.Doit.Set(config.NS, doctl.ArgUptimeAlertType, "latency")
		config.Doit.Set(config.NS, doctl.ArgUptimeAlertThreshold, 1000)
		config.Doit.Set(config.NS, doctl.ArgUptimeAlertComparison, "less_than")
		config.Doit.Set(config.NS, doctl.ArgUptimeAlertEmails, []string{"bob@example.com", "mike@example.com"})
		config.Doit.Set(config.NS, doctl.ArgUptimeAlertSlackURLs, "https://hooks.slack.com/services/T1234567/AAAAAAAAA/ZZZZZZ")
		config.Doit.Set(config.NS, doctl.ArgUptimeAlertSlackChannels, "#alerts-test")
		config.Doit.Set(config.NS, doctl.ArgUptimeAlertPeriod, "2m")

		err := RunUptimeAlertsCreate(config)
		assert.NoError(t, err)
	})
}

func TestUptimeAlertsList(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.uptimeChecks.EXPECT().ListAlerts(testUptimeCheckID).Return(testUptimeAlertsList, nil)

		config.Args = append(config.Args, testUptimeCheckID)

		err := RunUptimeAlertsList(config)
		assert.NoError(t, err)
	})
}

func TestUptimeAlertsUpdate(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tua := godo.UpdateUptimeAlertRequest{
			Name:       "Test Alert",
			Type:       "latency",
			Threshold:  2000,
			Comparison: "greater_than",
			Notifications: &godo.Notifications{
				Email: []string{"bob@example.com", "mike@example.com"},
				Slack: []godo.SlackDetails{{URL: "https://hooks.slack.com/services/T1234567/AAAAAAAAA/ZZZZZZ", Channel: "#alerts-test"}},
			},
			Period: "5m",
		}
		tm.uptimeChecks.EXPECT().UpdateAlert(testUptimeCheckID, testUptimeAlert.ID, &tua).Return(&testUptimeAlert, nil)

		config.Args = append(config.Args, testUptimeCheckID)
		config.Args = append(config.Args, testUptimeAlert.ID)

		config.Doit.Set(config.NS, doctl.ArgUptimeAlertName, "Test Alert")
		config.Doit.Set(config.NS, doctl.ArgUptimeAlertType, "latency")
		config.Doit.Set(config.NS, doctl.ArgUptimeAlertThreshold, 2000)
		config.Doit.Set(config.NS, doctl.ArgUptimeAlertComparison, "greater_than")
		config.Doit.Set(config.NS, doctl.ArgUptimeAlertEmails, []string{"bob@example.com", "mike@example.com"})
		config.Doit.Set(config.NS, doctl.ArgUptimeAlertSlackURLs, "https://hooks.slack.com/services/T1234567/AAAAAAAAA/ZZZZZZ")
		config.Doit.Set(config.NS, doctl.ArgUptimeAlertSlackChannels, "#alerts-test")
		config.Doit.Set(config.NS, doctl.ArgUptimeAlertPeriod, "5m")

		err := RunUptimeAlertsUpdate(config)
		assert.NoError(t, err)
	})
}

func TestUptimeAlertsDelete(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.uptimeChecks.EXPECT().DeleteAlert(testUptimeCheckID, testUptimeAlert.ID).Return(nil)

		config.Args = append(config.Args, testUptimeCheckID)
		config.Args = append(config.Args, testUptimeAlert.ID)

		err := RunUptimeAlertsDelete(config)
		assert.NoError(t, err)
	})
}

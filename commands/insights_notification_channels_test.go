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

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testInsightsNotificationChannel = godo.NotificationChannel{
	ID:          "channel-id",
	Name:        "Operations email",
	ChannelType: godo.InsightsChannelTypeEmail,
	Email:       &godo.EmailNotificationConfig{To: "ops@example.com"},
}

func TestInsightsNotificationChannelCommand(t *testing.T) {
	command := InsightsNotificationChannel()
	assertCommandNames(t, command, "create", "get", "list", "update", "delete")

	deleteCommand, _, err := command.Find([]string{"delete"})
	require.NoError(t, err)
	forceFlag := deleteCommand.Flags().Lookup(doctl.ArgForce)
	require.NotNil(t, forceFlag)
	assert.Equal(t, doctl.ArgShortForce, forceFlag.Shorthand)
}

func TestRunInsightsNotificationChannelCreate(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		request := &godo.NotificationChannelRequest{
			Name:  "Operations email",
			Email: &godo.EmailNotificationConfig{To: "ops@example.com,oncall@example.com"},
		}
		tm.insights.EXPECT().CreateNotificationChannel(request).Return(&testInsightsNotificationChannel, nil)

		config.Doit.Set(config.NS, doctl.ArgInsightsName, "Operations email")
		config.Doit.Set(config.NS, doctl.ArgInsightsType, "email")
		config.Doit.Set(config.NS, doctl.ArgInsightsEmails, []string{"ops@example.com", "oncall@example.com"})

		require.NoError(t, RunInsightsNotificationChannelCreate(config))
	})
}

func TestRunInsightsNotificationChannelCreateRejectsMixedTypes(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, _ *tcMocks) {
		config.Doit.Set(config.NS, doctl.ArgInsightsName, "Invalid")
		config.Doit.Set(config.NS, doctl.ArgInsightsType, "email")
		config.Doit.Set(config.NS, doctl.ArgInsightsEmails, []string{"ops@example.com"})
		config.Doit.Set(config.NS, doctl.ArgInsightsSlackChannel, "#alerts")

		err := RunInsightsNotificationChannelCreate(config)
		require.EqualError(t, err, "email notification channels cannot use Slack or webhook flags")
	})
}

func TestRunInsightsNotificationChannelGet(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.insights.EXPECT().GetNotificationChannel("channel-id").Return(&testInsightsNotificationChannel, nil)
		config.Args = []string{"channel-id"}

		require.NoError(t, RunInsightsNotificationChannelGet(config))
	})
}

func TestRunInsightsNotificationChannelList(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		options := &godo.ListOptions{Page: 2, PerPage: 25}
		tm.insights.EXPECT().
			ListNotificationChannels(options).
			Return([]godo.NotificationChannel{testInsightsNotificationChannel}, nil, nil)
		config.Doit.Set(config.NS, doctl.ArgInsightsPage, 2)
		config.Doit.Set(config.NS, doctl.ArgInsightsPerPage, 25)

		require.NoError(t, RunInsightsNotificationChannelList(config))
	})
}

func TestRunInsightsNotificationChannelUpdatePreservesEmailDestination(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		updated := testInsightsNotificationChannel
		updated.Name = "Primary operations email"
		request := &godo.NotificationChannelRequest{
			Name:  "Primary operations email",
			Email: &godo.EmailNotificationConfig{To: "ops@example.com"},
		}
		tm.insights.EXPECT().GetNotificationChannel("channel-id").Return(&testInsightsNotificationChannel, nil)
		tm.insights.EXPECT().UpdateNotificationChannel("channel-id", request).Return(&updated, nil)

		config.Args = []string{"channel-id"}
		config.Doit.Set(config.NS, doctl.ArgInsightsName, "Primary operations email")

		require.NoError(t, RunInsightsNotificationChannelUpdate(config))
	})
}

func TestRunInsightsNotificationChannelDelete(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.insights.EXPECT().DeleteNotificationChannel("channel-id").Return(nil)
		config.Args = []string{"channel-id"}
		config.Doit.Set(config.NS, doctl.ArgForce, true)

		require.NoError(t, RunInsightsNotificationChannelDelete(config))
	})
}

func TestRunInsightsNotificationChannelDeleteRequiresConfirmation(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, _ *tcMocks) {
		config.Args = []string{"channel-id"}

		require.ErrorIs(t, RunInsightsNotificationChannelDelete(config), errConfirmationRequired)
	})
}

func TestValidateNotificationChannelRequest(t *testing.T) {
	t.Run("requires exactly one config", func(t *testing.T) {
		err := validateNotificationChannelRequest(&godo.NotificationChannelRequest{Name: "invalid"}, nil, false)
		assert.EqualError(t, err, "exactly one email, Slack, or webhook configuration is required")
	})

	t.Run("rejects insecure webhook URL", func(t *testing.T) {
		err := validateNotificationChannelRequest(&godo.NotificationChannelRequest{
			Name:    "Webhook",
			Webhook: &godo.WebhookNotificationConfig{URL: "http://example.com/hook"},
		}, nil, false)
		assert.EqualError(t, err, "webhook URL must be a valid HTTPS URL")
	})

	t.Run("requires Slack secret when changing type", func(t *testing.T) {
		current := &godo.NotificationChannel{
			ChannelType: godo.InsightsChannelTypeEmail,
			Email:       &godo.EmailNotificationConfig{To: "ops@example.com"},
		}
		err := validateNotificationChannelRequest(&godo.NotificationChannelRequest{
			Name:  "Slack",
			Slack: &godo.SlackNotificationConfig{Channel: "#alerts"},
		}, current, true)
		assert.EqualError(t, err, "Slack webhook URL is required")
	})

	t.Run("preserves Slack secret for an existing Slack channel", func(t *testing.T) {
		current := &godo.NotificationChannel{
			ChannelType: godo.InsightsChannelTypeSlack,
			Slack:       &godo.SlackNotificationConfig{Channel: "#alerts", WebhookURL: "********"},
		}
		err := validateNotificationChannelRequest(&godo.NotificationChannelRequest{
			Name:  "Slack",
			Slack: &godo.SlackNotificationConfig{Channel: "#operations"},
		}, current, true)
		assert.NoError(t, err)
	})
}

func TestInsightsWebhookConfigRequiresPasswordForNewBasicAuth(t *testing.T) {
	config := &CmdConfig{NS: "test", Doit: doctl.NewTestConfig()}
	config.Doit.Set(config.NS, doctl.ArgInsightsWebhookBasicAuthUsername, "alerts")
	current := &godo.WebhookNotificationConfig{
		URL:         "https://example.com/hook",
		BearerToken: &godo.WebhookBearerToken{},
	}

	_, err := insightsWebhookConfig(config, current, true)
	require.EqualError(t, err, "--webhook-basic-auth-password is required with --webhook-basic-auth-username")
}

func TestInsightsWebhookConfigPreservesExistingBasicAuthSecret(t *testing.T) {
	config := &CmdConfig{NS: "test", Doit: doctl.NewTestConfig()}
	config.Doit.Set(config.NS, doctl.ArgInsightsWebhookBasicAuthUsername, "new-alerts-user")
	current := &godo.WebhookNotificationConfig{
		URL:       "https://example.com/hook",
		BasicAuth: &godo.WebhookBasicAuth{Username: "alerts"},
	}

	webhook, err := insightsWebhookConfig(config, current, true)
	require.NoError(t, err)
	assert.Equal(t, &godo.WebhookBasicAuth{Username: "new-alerts-user"}, webhook.BasicAuth)
}

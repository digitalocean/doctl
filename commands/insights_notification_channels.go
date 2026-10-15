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
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/commands/displayers"
	"github.com/digitalocean/godo"
	"github.com/spf13/cobra"
)

var notificationChannelRequestFlags = []string{
	doctl.ArgInsightsName,
	doctl.ArgInsightsType,
	doctl.ArgInsightsEmails,
	doctl.ArgInsightsSlackChannel,
	doctl.ArgInsightsSlackWebhookURL,
	doctl.ArgInsightsWebhookURL,
	doctl.ArgInsightsWebhookHeader,
	doctl.ArgInsightsWebhookBasicAuthUsername,
	doctl.ArgInsightsWebhookBasicAuthPassword,
	doctl.ArgInsightsWebhookBearerToken,
	doctl.ArgInsightsWebhookSigningSecret,
}

// InsightsNotificationChannel creates the Insights notification-channel command group.
func InsightsNotificationChannel() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:   "notification-channel",
			Short: "Manage Insights notification channels",
			Long:  "Create and manage reusable email, Slack, and webhook destinations for Insights alerts.",
		},
	}

	create := CmdBuilder(cmd, RunInsightsNotificationChannelCreate, "create", "Create a notification channel",
		"Creates an Insights notification channel.", Writer,
		aliasOpt("c"),
		displayerType(&displayers.InsightsNotificationChannel{}))
	create.Args = cobra.NoArgs
	addInsightsNotificationChannelRequestFlags(create)

	get := CmdBuilder(cmd, RunInsightsNotificationChannelGet, "get <channel-id>", "Get a notification channel",
		"Retrieves an Insights notification channel. Secret values are masked by the API.", Writer,
		aliasOpt("g"),
		displayerType(&displayers.InsightsNotificationChannel{}))
	get.Args = cobra.ExactArgs(1)

	list := CmdBuilder(cmd, RunInsightsNotificationChannelList, "list", "List notification channels",
		"Lists Insights notification channels.", Writer,
		aliasOpt("ls"),
		displayerType(&displayers.InsightsNotificationChannel{}))
	list.Args = cobra.NoArgs
	AddIntFlag(list, doctl.ArgInsightsPage, "", 0, "Page number")
	AddIntFlag(list, doctl.ArgInsightsPerPage, "", 0, "Number of notification channels per page")

	update := CmdBuilder(cmd, RunInsightsNotificationChannelUpdate, "update <channel-id>", "Update a notification channel",
		"Replaces an Insights notification channel. Omit write-only secret flags to preserve their current values.", Writer,
		aliasOpt("u"),
		displayerType(&displayers.InsightsNotificationChannel{}))
	update.Args = cobra.ExactArgs(1)
	addInsightsNotificationChannelRequestFlags(update)

	deleteCmd := CmdBuilder(cmd, RunInsightsNotificationChannelDelete, "delete <channel-id>", "Delete a notification channel",
		"Deletes an Insights notification channel. Channels referenced by alert rules cannot be deleted.", Writer,
		aliasOpt("d", "del", "rm"))
	deleteCmd.Args = cobra.ExactArgs(1)

	return cmd
}

func addInsightsNotificationChannelRequestFlags(cmd *Command) {
	AddStringFlag(cmd, doctl.ArgInsightsName, "", "", "Notification channel name")
	AddStringFlag(cmd, doctl.ArgInsightsType, "", "", "Notification channel type: email, slack, or webhook")
	AddStringFlag(cmd, doctl.ArgInsightsRequestFile, "", "", "Path to a complete JSON or YAML request; use - for stdin")
	AddStringSliceFlag(cmd, doctl.ArgInsightsEmails, "", nil, "Email recipients")
	AddStringFlag(cmd, doctl.ArgInsightsSlackChannel, "", "", "Slack channel")
	AddStringFlag(cmd, doctl.ArgInsightsSlackWebhookURL, "", "", "Slack incoming webhook URL")
	AddStringFlag(cmd, doctl.ArgInsightsWebhookURL, "", "", "HTTPS webhook destination")
	AddStringMapStringFlag(cmd, doctl.ArgInsightsWebhookHeader, "", nil, "Webhook header in key=value form; repeat for multiple headers")
	AddStringFlag(cmd, doctl.ArgInsightsWebhookBasicAuthUsername, "", "", "Webhook basic-auth username")
	AddStringFlag(cmd, doctl.ArgInsightsWebhookBasicAuthPassword, "", "", "Webhook basic-auth password")
	AddStringFlag(cmd, doctl.ArgInsightsWebhookBearerToken, "", "", "Webhook bearer token")
	AddStringFlag(cmd, doctl.ArgInsightsWebhookSigningSecret, "", "", "Webhook HMAC signing secret")
}

// RunInsightsNotificationChannelCreate creates a notification channel.
func RunInsightsNotificationChannelCreate(c *CmdConfig) error {
	request, err := insightsNotificationChannelRequest(c, nil, false)
	if err != nil {
		return err
	}
	channel, err := c.Insights().CreateNotificationChannel(request)
	if err != nil {
		return err
	}
	return displayInsightsNotificationChannels(c, *channel)
}

// RunInsightsNotificationChannelGet retrieves a notification channel.
func RunInsightsNotificationChannelGet(c *CmdConfig) error {
	if len(c.Args) != 1 {
		return doctl.NewMissingArgsErr(c.NS)
	}
	channel, err := c.Insights().GetNotificationChannel(c.Args[0])
	if err != nil {
		return err
	}
	return displayInsightsNotificationChannels(c, *channel)
}

// RunInsightsNotificationChannelList lists notification channels.
func RunInsightsNotificationChannelList(c *CmdConfig) error {
	page, err := c.Doit.GetInt(c.NS, doctl.ArgInsightsPage)
	if err != nil {
		return err
	}
	perPage, err := c.Doit.GetInt(c.NS, doctl.ArgInsightsPerPage)
	if err != nil {
		return err
	}
	if page < 0 || perPage < 0 {
		return fmt.Errorf("--page and --per-page cannot be negative")
	}

	channels, _, err := c.Insights().ListNotificationChannels(&godo.ListOptions{Page: page, PerPage: perPage})
	if err != nil {
		return err
	}
	return displayInsightsNotificationChannels(c, channels...)
}

// RunInsightsNotificationChannelUpdate updates a notification channel.
func RunInsightsNotificationChannelUpdate(c *CmdConfig) error {
	if len(c.Args) != 1 {
		return doctl.NewMissingArgsErr(c.NS)
	}
	requestFile, err := c.Doit.GetString(c.NS, doctl.ArgInsightsRequestFile)
	if err != nil {
		return err
	}

	if requestFile == "" {
		if !anyInsightsFlagSet(c, notificationChannelRequestFlags...) {
			return fmt.Errorf("nothing to update")
		}
	}
	current, err := c.Insights().GetNotificationChannel(c.Args[0])
	if err != nil {
		return err
	}

	request, err := insightsNotificationChannelRequest(c, current, true)
	if err != nil {
		return err
	}
	channel, err := c.Insights().UpdateNotificationChannel(c.Args[0], request)
	if err != nil {
		return err
	}
	return displayInsightsNotificationChannels(c, *channel)
}

// RunInsightsNotificationChannelDelete deletes a notification channel.
func RunInsightsNotificationChannelDelete(c *CmdConfig) error {
	if len(c.Args) != 1 {
		return doctl.NewMissingArgsErr(c.NS)
	}
	return c.Insights().DeleteNotificationChannel(c.Args[0])
}

func displayInsightsNotificationChannels(c *CmdConfig, channels ...godo.NotificationChannel) error {
	return c.Display(&displayers.InsightsNotificationChannel{Channels: channels})
}

func insightsNotificationChannelRequest(c *CmdConfig, current *godo.NotificationChannel, update bool) (*godo.NotificationChannelRequest, error) {
	requestFile, err := c.Doit.GetString(c.NS, doctl.ArgInsightsRequestFile)
	if err != nil {
		return nil, err
	}
	if requestFile != "" {
		if err := rejectRequestFileFlagMix(c, requestFile, notificationChannelRequestFlags...); err != nil {
			return nil, err
		}
		request, err := readInsightsRequest[godo.NotificationChannelRequest](os.Stdin, requestFile)
		if err != nil {
			return nil, err
		}
		if err := validateNotificationChannelRequest(request, current, update); err != nil {
			return nil, err
		}
		return request, nil
	}

	name := ""
	channelType := ""
	if current != nil {
		name = current.Name
		channelType = notificationChannelTypeName(current.ChannelType)
	}
	if current == nil || c.Doit.IsSet(doctl.ArgInsightsName) {
		name, err = c.Doit.GetString(c.NS, doctl.ArgInsightsName)
		if err != nil {
			return nil, err
		}
	}
	if current == nil || c.Doit.IsSet(doctl.ArgInsightsType) {
		channelType, err = c.Doit.GetString(c.NS, doctl.ArgInsightsType)
		if err != nil {
			return nil, err
		}
	}
	if strings.TrimSpace(name) == "" {
		return nil, fmt.Errorf("--name is required")
	}
	if channelType == "" {
		return nil, fmt.Errorf("--type is required")
	}

	request := &godo.NotificationChannelRequest{Name: name}
	switch strings.ToLower(channelType) {
	case "email":
		if anyInsightsFlagSet(c, doctl.ArgInsightsSlackChannel, doctl.ArgInsightsSlackWebhookURL,
			doctl.ArgInsightsWebhookURL, doctl.ArgInsightsWebhookHeader,
			doctl.ArgInsightsWebhookBasicAuthUsername, doctl.ArgInsightsWebhookBasicAuthPassword,
			doctl.ArgInsightsWebhookBearerToken, doctl.ArgInsightsWebhookSigningSecret) {
			return nil, fmt.Errorf("email notification channels cannot use Slack or webhook flags")
		}
		to := ""
		if current != nil && notificationChannelTypeName(current.ChannelType) == "email" && current.Email != nil {
			to = current.Email.To
		}
		if current == nil || c.Doit.IsSet(doctl.ArgInsightsEmails) {
			emails, err := c.Doit.GetStringSlice(c.NS, doctl.ArgInsightsEmails)
			if err != nil {
				return nil, err
			}
			to = strings.Join(emails, ",")
		}
		request.Email = &godo.EmailNotificationConfig{To: to}
	case "slack":
		if anyInsightsFlagSet(c, doctl.ArgInsightsEmails, doctl.ArgInsightsWebhookURL,
			doctl.ArgInsightsWebhookHeader, doctl.ArgInsightsWebhookBasicAuthUsername,
			doctl.ArgInsightsWebhookBasicAuthPassword, doctl.ArgInsightsWebhookBearerToken,
			doctl.ArgInsightsWebhookSigningSecret) {
			return nil, fmt.Errorf("Slack notification channels cannot use email or webhook flags")
		}
		channel := ""
		if current != nil && notificationChannelTypeName(current.ChannelType) == "slack" && current.Slack != nil {
			channel = current.Slack.Channel
		}
		if current == nil || c.Doit.IsSet(doctl.ArgInsightsSlackChannel) {
			channel, err = c.Doit.GetString(c.NS, doctl.ArgInsightsSlackChannel)
			if err != nil {
				return nil, err
			}
		}
		webhookURL := ""
		if c.Doit.IsSet(doctl.ArgInsightsSlackWebhookURL) {
			webhookURL, err = c.Doit.GetString(c.NS, doctl.ArgInsightsSlackWebhookURL)
			if err != nil {
				return nil, err
			}
		}
		request.Slack = &godo.SlackNotificationConfig{Channel: channel, WebhookURL: webhookURL}
	case "webhook":
		if anyInsightsFlagSet(c, doctl.ArgInsightsEmails, doctl.ArgInsightsSlackChannel, doctl.ArgInsightsSlackWebhookURL) {
			return nil, fmt.Errorf("webhook notification channels cannot use email or Slack flags")
		}
		var currentWebhook *godo.WebhookNotificationConfig
		if current != nil && notificationChannelTypeName(current.ChannelType) == "webhook" {
			currentWebhook = current.Webhook
		}
		webhook, err := insightsWebhookConfig(c, currentWebhook, update)
		if err != nil {
			return nil, err
		}
		request.Webhook = webhook
	default:
		return nil, fmt.Errorf("--type must be one of email, slack, or webhook")
	}

	if err := validateNotificationChannelRequest(request, current, update); err != nil {
		return nil, err
	}
	return request, nil
}

func insightsWebhookConfig(c *CmdConfig, current *godo.WebhookNotificationConfig, update bool) (*godo.WebhookNotificationConfig, error) {
	webhook := &godo.WebhookNotificationConfig{}
	if current != nil {
		webhook.URL = current.URL
		webhook.Headers = current.Headers
		if current.BasicAuth != nil {
			webhook.BasicAuth = &godo.WebhookBasicAuth{Username: current.BasicAuth.Username}
		}
		if current.BearerToken != nil {
			webhook.BearerToken = &godo.WebhookBearerToken{}
		}
		if current.Signature != nil {
			webhook.Signature = &godo.WebhookSignatureConfig{}
		}
	}

	var err error
	if current == nil || c.Doit.IsSet(doctl.ArgInsightsWebhookURL) {
		webhook.URL, err = c.Doit.GetString(c.NS, doctl.ArgInsightsWebhookURL)
		if err != nil {
			return nil, err
		}
	}
	if c.Doit.IsSet(doctl.ArgInsightsWebhookHeader) {
		webhook.Headers, err = c.Doit.GetStringMapString(c.NS, doctl.ArgInsightsWebhookHeader)
		if err != nil {
			return nil, err
		}
	}

	username, password, bearer, signingSecret := "", "", "", ""
	if c.Doit.IsSet(doctl.ArgInsightsWebhookBasicAuthUsername) {
		username, err = c.Doit.GetString(c.NS, doctl.ArgInsightsWebhookBasicAuthUsername)
		if err != nil {
			return nil, err
		}
	}
	if c.Doit.IsSet(doctl.ArgInsightsWebhookBasicAuthPassword) {
		password, err = c.Doit.GetString(c.NS, doctl.ArgInsightsWebhookBasicAuthPassword)
		if err != nil {
			return nil, err
		}
	}
	if c.Doit.IsSet(doctl.ArgInsightsWebhookBearerToken) {
		bearer, err = c.Doit.GetString(c.NS, doctl.ArgInsightsWebhookBearerToken)
		if err != nil {
			return nil, err
		}
	}
	if c.Doit.IsSet(doctl.ArgInsightsWebhookSigningSecret) {
		signingSecret, err = c.Doit.GetString(c.NS, doctl.ArgInsightsWebhookSigningSecret)
		if err != nil {
			return nil, err
		}
	}

	effectiveUsername := username
	if effectiveUsername == "" && webhook.BasicAuth != nil {
		effectiveUsername = webhook.BasicAuth.Username
	}
	if (username != "" || password != "") && bearer != "" {
		return nil, fmt.Errorf("webhook basic authentication and bearer-token authentication are mutually exclusive")
	}
	if password != "" && effectiveUsername == "" {
		return nil, fmt.Errorf("--webhook-basic-auth-username is required with --webhook-basic-auth-password")
	}
	hasExistingBasicAuth := current != nil && current.BasicAuth != nil
	if effectiveUsername != "" && password == "" && (!update || !hasExistingBasicAuth) {
		return nil, fmt.Errorf("--webhook-basic-auth-password is required with --webhook-basic-auth-username")
	}

	if username != "" || password != "" {
		webhook.BasicAuth = &godo.WebhookBasicAuth{Username: effectiveUsername, Password: password}
		webhook.BearerToken = nil
	}
	if bearer != "" {
		webhook.BearerToken = &godo.WebhookBearerToken{Token: bearer}
		webhook.BasicAuth = nil
	}
	if signingSecret != "" {
		webhook.Signature = &godo.WebhookSignatureConfig{Secret: signingSecret}
	}
	return webhook, nil
}

func notificationChannelTypeName(channelType string) string {
	switch channelType {
	case godo.InsightsChannelTypeEmail:
		return "email"
	case godo.InsightsChannelTypeSlack:
		return "slack"
	case godo.InsightsChannelTypeWebhook:
		return "webhook"
	default:
		return strings.ToLower(channelType)
	}
}

func validateNotificationChannelRequest(request *godo.NotificationChannelRequest, current *godo.NotificationChannel, update bool) error {
	if request == nil {
		return fmt.Errorf("notification channel request is required")
	}
	if strings.TrimSpace(request.Name) == "" {
		return fmt.Errorf("notification channel name is required")
	}

	configCount := 0
	if request.Email != nil {
		configCount++
		if strings.TrimSpace(request.Email.To) == "" {
			return fmt.Errorf("at least one email recipient is required")
		}
	}
	if request.Slack != nil {
		configCount++
		if strings.TrimSpace(request.Slack.Channel) == "" {
			return fmt.Errorf("Slack channel is required")
		}
		canPreserveSlackSecret := update && current != nil && current.Slack != nil
		if strings.TrimSpace(request.Slack.WebhookURL) == "" && !canPreserveSlackSecret {
			return fmt.Errorf("Slack webhook URL is required")
		}
		if request.Slack.WebhookURL != "" {
			if err := validateHTTPSURL(request.Slack.WebhookURL, "Slack webhook URL"); err != nil {
				return err
			}
		}
	}
	if request.Webhook != nil {
		configCount++
		if err := validateHTTPSURL(request.Webhook.URL, "webhook URL"); err != nil {
			return err
		}
		if request.Webhook.BasicAuth != nil && request.Webhook.BearerToken != nil {
			return fmt.Errorf("webhook basic authentication and bearer-token authentication are mutually exclusive")
		}
		if request.Webhook.BasicAuth != nil {
			if strings.TrimSpace(request.Webhook.BasicAuth.Username) == "" {
				return fmt.Errorf("webhook basic-auth username is required")
			}
			canPreservePassword := update && current != nil && current.Webhook != nil && current.Webhook.BasicAuth != nil
			if request.Webhook.BasicAuth.Password == "" && !canPreservePassword {
				return fmt.Errorf("webhook basic-auth password is required")
			}
		}
		if request.Webhook.BearerToken != nil && request.Webhook.BearerToken.Token == "" {
			canPreserveBearer := update && current != nil && current.Webhook != nil && current.Webhook.BearerToken != nil
			if !canPreserveBearer {
				return fmt.Errorf("webhook bearer token is required")
			}
		}
		if request.Webhook.Signature != nil && request.Webhook.Signature.Secret == "" {
			canPreserveSignature := update && current != nil && current.Webhook != nil && current.Webhook.Signature != nil
			if !canPreserveSignature {
				return fmt.Errorf("webhook signing secret is required")
			}
		}
	}
	if configCount != 1 {
		return fmt.Errorf("exactly one email, Slack, or webhook configuration is required")
	}
	return nil
}

func validateHTTPSURL(rawURL, field string) error {
	parsed, err := url.ParseRequestURI(rawURL)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" {
		return fmt.Errorf("%s must be a valid HTTPS URL", field)
	}
	return nil
}

func anyInsightsFlagSet(c *CmdConfig, flags ...string) bool {
	for _, flag := range flags {
		if c.Doit.IsSet(flag) {
			return true
		}
	}
	return false
}

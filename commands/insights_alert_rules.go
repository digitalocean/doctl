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
	"os"
	"slices"
	"strings"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/commands/displayers"
	"github.com/digitalocean/godo"
	"github.com/spf13/cobra"
)

var alertRuleRequestFlags = []string{
	doctl.ArgInsightsName,
	doctl.ArgInsightsMetric,
	doctl.ArgInsightsResourceURN,
	doctl.ArgInsightsTag,
	doctl.ArgInsightsFilter,
	doctl.ArgInsightsWindow,
	doctl.ArgInsightsOperator,
	doctl.ArgInsightsWarningThreshold,
	doctl.ArgInsightsCriticalThreshold,
	doctl.ArgInsightsNotificationChannel,
	doctl.ArgInsightsNotifyOn,
	doctl.ArgInsightsReAlertDuration,
	doctl.ArgInsightsStatus,
}

// InsightsAlertRule creates the Insights alert-rule command group.
func InsightsAlertRule() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:   "alert-rule",
			Short: "Manage Insights alert rules",
			Long:  "Create and manage Insights metric alert rules.",
		},
	}

	create := CmdBuilder(cmd, RunInsightsAlertRuleCreate, "create", "Create an alert rule",
		"Creates an Insights metric alert rule.", Writer,
		aliasOpt("c"),
		displayerType(&displayers.InsightsAlertRule{}))
	create.Args = cobra.NoArgs
	addInsightsAlertRuleRequestFlags(create)

	get := CmdBuilder(cmd, RunInsightsAlertRuleGet, "get <rule-id>", "Get an alert rule",
		"Retrieves an Insights alert rule.", Writer,
		aliasOpt("g"),
		displayerType(&displayers.InsightsAlertRule{}))
	get.Args = cobra.ExactArgs(1)

	list := CmdBuilder(cmd, RunInsightsAlertRuleList, "list", "List alert rules",
		"Lists Insights alert rules.", Writer,
		aliasOpt("ls"),
		displayerType(&displayers.InsightsAlertRule{}))
	list.Args = cobra.NoArgs
	AddStringFlag(list, doctl.ArgInsightsResourceURN, "", "", "Filter by resource URN")
	AddIntFlag(list, doctl.ArgInsightsPage, "", 0, "Page number")
	AddIntFlag(list, doctl.ArgInsightsPerPage, "", 0, "Number of alert rules per page")

	update := CmdBuilder(cmd, RunInsightsAlertRuleUpdate, "update <rule-id>", "Update an alert rule",
		"Updates an Insights alert rule. Omitted flags preserve their current values.", Writer,
		aliasOpt("u"),
		displayerType(&displayers.InsightsAlertRule{}))
	update.Args = cobra.ExactArgs(1)
	addInsightsAlertRuleRequestFlags(update)

	deleteCmd := CmdBuilder(cmd, RunInsightsAlertRuleDelete, "delete <rule-id>", "Delete an alert rule",
		"Deletes an Insights alert rule.", Writer,
		aliasOpt("d", "del", "rm"))
	deleteCmd.Args = cobra.ExactArgs(1)

	return cmd
}

func addInsightsAlertRuleRequestFlags(cmd *Command) {
	AddStringFlag(cmd, doctl.ArgInsightsName, "", "", "Alert rule name")
	AddStringFlag(cmd, doctl.ArgInsightsMetric, "", "", "Dotted OpenTelemetry metric name")
	AddStringSliceFlag(cmd, doctl.ArgInsightsResourceURN, "", nil, "Resource URN; repeat to include multiple resources")
	AddStringSliceFlag(cmd, doctl.ArgInsightsTag, "", nil, "Resource tag; repeat to include multiple tags")
	AddStringArrayFlag(cmd, doctl.ArgInsightsFilter, "", nil, "Metric filter in field,operator,value form; repeat to include multiple filters")
	AddStringFlag(cmd, doctl.ArgInsightsWindow, "", "", "Evaluation window: 1m, 5m, 10m, 15m, 30m, or 1h")
	AddStringFlag(cmd, doctl.ArgInsightsOperator, "", "", "Threshold operator: eq, neq, lt, lte, gt, or gte")
	AddFloatFlag(cmd, doctl.ArgInsightsWarningThreshold, "", 0, "Warning threshold")
	AddFloatFlag(cmd, doctl.ArgInsightsCriticalThreshold, "", 0, "Critical threshold")
	AddStringSliceFlag(cmd, doctl.ArgInsightsNotificationChannel, "", nil, "Notification channel ID; repeat to include multiple channels")
	AddStringSliceFlag(cmd, doctl.ArgInsightsNotifyOn, "", nil, "Severity to notify on: warning or critical; repeat for both")
	AddStringFlag(cmd, doctl.ArgInsightsReAlertDuration, "", "", "Re-alert interval: 30m, 1h, 4h, or never")
	AddStringFlag(cmd, doctl.ArgInsightsStatus, "", "", "Alert rule status: active or paused")
	AddStringFlag(cmd, doctl.ArgInsightsRequestFile, "", "", "Path to a complete JSON or YAML request; use - for stdin")
}

// RunInsightsAlertRuleCreate creates an alert rule.
func RunInsightsAlertRuleCreate(c *CmdConfig) error {
	request, err := insightsAlertRuleRequest(c, nil, false)
	if err != nil {
		return err
	}
	rule, err := c.Insights().CreateAlertRule(request)
	if err != nil {
		return err
	}
	return displayInsightsAlertRules(c, *rule)
}

// RunInsightsAlertRuleGet retrieves an alert rule.
func RunInsightsAlertRuleGet(c *CmdConfig) error {
	if len(c.Args) != 1 {
		return doctl.NewMissingArgsErr(c.NS)
	}
	rule, err := c.Insights().GetAlertRule(c.Args[0])
	if err != nil {
		return err
	}
	return displayInsightsAlertRules(c, *rule)
}

// RunInsightsAlertRuleList lists alert rules.
func RunInsightsAlertRuleList(c *CmdConfig) error {
	resourceURN, err := c.Doit.GetString(c.NS, doctl.ArgInsightsResourceURN)
	if err != nil {
		return err
	}
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

	rules, _, err := c.Insights().ListAlertRules(&godo.AlertRuleListOptions{
		Page:        page,
		PerPage:     perPage,
		ResourceURN: resourceURN,
	})
	if err != nil {
		return err
	}
	return displayInsightsAlertRules(c, rules...)
}

// RunInsightsAlertRuleUpdate updates an alert rule.
func RunInsightsAlertRuleUpdate(c *CmdConfig) error {
	if len(c.Args) != 1 {
		return doctl.NewMissingArgsErr(c.NS)
	}
	requestFile, err := c.Doit.GetString(c.NS, doctl.ArgInsightsRequestFile)
	if err != nil {
		return err
	}

	var current *godo.AlertRule
	if requestFile == "" {
		if !anyInsightsFlagSet(c, alertRuleRequestFlags...) {
			return fmt.Errorf("nothing to update")
		}
		current, err = c.Insights().GetAlertRule(c.Args[0])
		if err != nil {
			return err
		}
	}

	request, err := insightsAlertRuleRequest(c, current, true)
	if err != nil {
		return err
	}
	rule, err := c.Insights().UpdateAlertRule(c.Args[0], request)
	if err != nil {
		return err
	}
	return displayInsightsAlertRules(c, *rule)
}

// RunInsightsAlertRuleDelete deletes an alert rule.
func RunInsightsAlertRuleDelete(c *CmdConfig) error {
	if len(c.Args) != 1 {
		return doctl.NewMissingArgsErr(c.NS)
	}
	return c.Insights().DeleteAlertRule(c.Args[0])
}

func displayInsightsAlertRules(c *CmdConfig, rules ...godo.AlertRule) error {
	return c.Display(&displayers.InsightsAlertRule{Rules: rules})
}

func insightsAlertRuleRequest(c *CmdConfig, current *godo.AlertRule, update bool) (*godo.AlertRuleRequest, error) {
	requestFile, err := c.Doit.GetString(c.NS, doctl.ArgInsightsRequestFile)
	if err != nil {
		return nil, err
	}
	if requestFile != "" {
		if err := rejectRequestFileFlagMix(c, requestFile, alertRuleRequestFlags...); err != nil {
			return nil, err
		}
		request, err := readInsightsRequest[godo.AlertRuleRequest](os.Stdin, requestFile)
		if err != nil {
			return nil, err
		}
		if err := validateAlertRuleRequest(request, update); err != nil {
			return nil, err
		}
		return request, nil
	}

	request := &godo.AlertRuleRequest{}
	if current != nil {
		request.Spec = current.Spec
		// These update fields have preserve-on-omission semantics. Populate
		// them below only when the corresponding flag is explicitly set.
		request.Spec.NotificationChannels = nil
		request.Spec.ReAlertDuration = ""
		request.Status = ""
	}

	if current == nil || c.Doit.IsSet(doctl.ArgInsightsName) {
		request.Spec.Name, err = c.Doit.GetString(c.NS, doctl.ArgInsightsName)
		if err != nil {
			return nil, err
		}
	}
	if current == nil || c.Doit.IsSet(doctl.ArgInsightsMetric) {
		request.Spec.Query.Metric, err = c.Doit.GetString(c.NS, doctl.ArgInsightsMetric)
		if err != nil {
			return nil, err
		}
	}
	if current == nil || c.Doit.IsSet(doctl.ArgInsightsResourceURN) {
		request.Spec.Query.ResourceURNs, err = c.Doit.GetStringSlice(c.NS, doctl.ArgInsightsResourceURN)
		if err != nil {
			return nil, err
		}
	}
	if current == nil || c.Doit.IsSet(doctl.ArgInsightsTag) {
		request.Spec.Query.Tags, err = c.Doit.GetStringSlice(c.NS, doctl.ArgInsightsTag)
		if err != nil {
			return nil, err
		}
	}
	if current == nil || c.Doit.IsSet(doctl.ArgInsightsFilter) {
		rawFilters, err := getInsightsStringArray(c, doctl.ArgInsightsFilter)
		if err != nil {
			return nil, err
		}
		request.Spec.Query.Filters, err = parseAlertRuleFilters(rawFilters)
		if err != nil {
			return nil, err
		}
	}
	if c.Doit.IsSet(doctl.ArgInsightsWindow) {
		window, err := c.Doit.GetString(c.NS, doctl.ArgInsightsWindow)
		if err != nil {
			return nil, err
		}
		mapped, err := alertWindow(window)
		if err != nil {
			return nil, err
		}
		request.Spec.Condition = &godo.AlertCondition{Window: mapped}
	}
	if current == nil && request.Spec.Condition == nil {
		window, err := c.Doit.GetString(c.NS, doctl.ArgInsightsWindow)
		if err != nil {
			return nil, err
		}
		if window != "" {
			mapped, err := alertWindow(window)
			if err != nil {
				return nil, err
			}
			request.Spec.Condition = &godo.AlertCondition{Window: mapped}
		}
	}
	if current == nil || c.Doit.IsSet(doctl.ArgInsightsOperator) {
		operator, err := c.Doit.GetString(c.NS, doctl.ArgInsightsOperator)
		if err != nil {
			return nil, err
		}
		request.Spec.Thresholds.Operator, err = alertThresholdOperator(operator)
		if err != nil {
			return nil, err
		}
	}
	if c.Doit.IsSet(doctl.ArgInsightsWarningThreshold) {
		value, err := c.Doit.GetFloat64(c.NS, doctl.ArgInsightsWarningThreshold)
		if err != nil {
			return nil, err
		}
		request.Spec.Thresholds.Warning = &value
	}
	if c.Doit.IsSet(doctl.ArgInsightsCriticalThreshold) {
		value, err := c.Doit.GetFloat64(c.NS, doctl.ArgInsightsCriticalThreshold)
		if err != nil {
			return nil, err
		}
		request.Spec.Thresholds.Critical = &value
	}

	channelsSet := c.Doit.IsSet(doctl.ArgInsightsNotificationChannel)
	notifyOnSet := c.Doit.IsSet(doctl.ArgInsightsNotifyOn)
	if current == nil || channelsSet || notifyOnSet {
		channelIDs, err := c.Doit.GetStringSlice(c.NS, doctl.ArgInsightsNotificationChannel)
		if err != nil {
			return nil, err
		}
		if !channelsSet && current != nil && current.Spec.NotificationChannels != nil {
			channelIDs = make([]string, 0, len(*current.Spec.NotificationChannels))
			for _, binding := range *current.Spec.NotificationChannels {
				channelIDs = append(channelIDs, binding.NotificationChannelID)
			}
		}
		notifyOn, err := c.Doit.GetStringSlice(c.NS, doctl.ArgInsightsNotifyOn)
		if err != nil {
			return nil, err
		}
		mappedNotifyOn, err := alertNotifyOn(notifyOn)
		if err != nil {
			return nil, err
		}
		bindings := make([]godo.NotificationChannelBinding, 0, len(channelIDs))
		for _, channelID := range channelIDs {
			bindings = append(bindings, godo.NotificationChannelBinding{
				NotificationChannelID: channelID,
				NotifyOn:              mappedNotifyOn,
			})
		}
		request.Spec.NotificationChannels = &bindings
	}

	if current == nil || c.Doit.IsSet(doctl.ArgInsightsReAlertDuration) {
		value, err := c.Doit.GetString(c.NS, doctl.ArgInsightsReAlertDuration)
		if err != nil {
			return nil, err
		}
		if value != "" {
			request.Spec.ReAlertDuration, err = alertReAlertDuration(value)
			if err != nil {
				return nil, err
			}
		}
	}
	if current == nil || c.Doit.IsSet(doctl.ArgInsightsStatus) {
		value, err := c.Doit.GetString(c.NS, doctl.ArgInsightsStatus)
		if err != nil {
			return nil, err
		}
		if value != "" {
			request.Status, err = alertRuleStatus(value)
			if err != nil {
				return nil, err
			}
		}
	}

	if err := validateAlertRuleRequest(request, update); err != nil {
		return nil, err
	}
	return request, nil
}

func parseAlertRuleFilters(values []string) ([]godo.AlertQueryFilter, error) {
	filters := make([]godo.AlertQueryFilter, 0, len(values))
	for _, value := range values {
		parts := strings.SplitN(value, ",", 3)
		if len(parts) != 3 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[2]) == "" {
			return nil, fmt.Errorf("--filter %q must use field,operator,value format", value)
		}
		operator, err := alertFilterOperator(strings.TrimSpace(parts[1]))
		if err != nil {
			return nil, err
		}
		filters = append(filters, godo.AlertQueryFilter{
			Field:    strings.TrimSpace(parts[0]),
			Operator: operator,
			Value:    strings.TrimSpace(parts[2]),
		})
	}
	return filters, nil
}

func validateAlertRuleRequest(request *godo.AlertRuleRequest, update bool) error {
	if request == nil {
		return fmt.Errorf("alert rule request is required")
	}
	if strings.TrimSpace(request.Spec.Name) == "" {
		return fmt.Errorf("alert rule name is required")
	}
	if strings.TrimSpace(request.Spec.Query.Metric) == "" {
		return fmt.Errorf("alert rule metric is required")
	}
	if !validAlertThresholdOperator(request.Spec.Thresholds.Operator) {
		return fmt.Errorf("alert rule threshold operator is invalid")
	}
	if request.Spec.Thresholds.Warning == nil && request.Spec.Thresholds.Critical == nil {
		return fmt.Errorf("at least one warning or critical threshold is required")
	}
	if request.Spec.Condition != nil && !validAlertWindow(request.Spec.Condition.Window) {
		return fmt.Errorf("alert rule evaluation window is invalid")
	}
	if request.Spec.ReAlertDuration != "" && !validAlertReAlertDuration(request.Spec.ReAlertDuration) {
		return fmt.Errorf("alert rule re-alert duration is invalid")
	}
	if request.Status != "" && !validAlertRuleStatus(request.Status) {
		return fmt.Errorf("alert rule status is invalid")
	}
	for _, filter := range request.Spec.Query.Filters {
		if strings.TrimSpace(filter.Field) == "" || strings.TrimSpace(filter.Value) == "" || !validAlertFilterOperator(filter.Operator) {
			return fmt.Errorf("alert rule query filter is invalid")
		}
	}
	if !update && (request.Spec.NotificationChannels == nil || len(*request.Spec.NotificationChannels) == 0) {
		return fmt.Errorf("at least one notification channel is required")
	}
	if request.Spec.NotificationChannels != nil {
		if len(*request.Spec.NotificationChannels) == 0 {
			return fmt.Errorf("at least one notification channel is required")
		}
		for _, binding := range *request.Spec.NotificationChannels {
			if strings.TrimSpace(binding.NotificationChannelID) == "" {
				return fmt.Errorf("notification channel ID cannot be empty")
			}
			for _, severity := range binding.NotifyOn {
				if severity != godo.InsightsSeverityWarning && severity != godo.InsightsSeverityCritical {
					return fmt.Errorf("notification severity is invalid")
				}
			}
		}
	}
	return nil
}

func alertWindow(value string) (string, error) {
	values := map[string]string{
		"1m": godo.InsightsEvaluationWindow1m, "5m": godo.InsightsEvaluationWindow5m,
		"10m": godo.InsightsEvaluationWindow10m, "15m": godo.InsightsEvaluationWindow15m,
		"30m": godo.InsightsEvaluationWindow30m, "1h": godo.InsightsEvaluationWindow1h,
	}
	if mapped, ok := values[strings.ToLower(value)]; ok {
		return mapped, nil
	}
	return "", fmt.Errorf("--window must be one of 1m, 5m, 10m, 15m, 30m, or 1h")
}

func alertThresholdOperator(value string) (string, error) {
	values := map[string]string{
		"eq": godo.InsightsThresholdOperatorEqual, "neq": godo.InsightsThresholdOperatorNotEqual,
		"lt": godo.InsightsThresholdOperatorLessThan, "lte": godo.InsightsThresholdOperatorLessThanOrEqual,
		"gt": godo.InsightsThresholdOperatorGreaterThan, "gte": godo.InsightsThresholdOperatorGreaterThanOrEqual,
	}
	if mapped, ok := values[strings.ToLower(value)]; ok {
		return mapped, nil
	}
	return "", fmt.Errorf("--operator must be one of eq, neq, lt, lte, gt, or gte")
}

func alertFilterOperator(value string) (string, error) {
	values := map[string]string{
		"eq": godo.InsightsFilterOperatorEqual, "neq": godo.InsightsFilterOperatorNotEqual,
		"lt": godo.InsightsFilterOperatorLessThan, "lte": godo.InsightsFilterOperatorLessThanOrEqual,
		"gt": godo.InsightsFilterOperatorGreaterThan, "gte": godo.InsightsFilterOperatorGreaterThanOrEqual,
	}
	if mapped, ok := values[strings.ToLower(value)]; ok {
		return mapped, nil
	}
	return "", fmt.Errorf("filter operator must be one of eq, neq, lt, lte, gt, or gte")
}

func alertNotifyOn(values []string) ([]string, error) {
	result := make([]string, 0, len(values))
	for _, value := range values {
		switch strings.ToLower(value) {
		case "warning":
			result = append(result, godo.InsightsSeverityWarning)
		case "critical":
			result = append(result, godo.InsightsSeverityCritical)
		default:
			return nil, fmt.Errorf("--notify-on must contain only warning or critical")
		}
	}
	return result, nil
}

func alertReAlertDuration(value string) (string, error) {
	values := map[string]string{
		"30m": godo.InsightsReAlertDuration30m, "1h": godo.InsightsReAlertDuration1h,
		"4h": godo.InsightsReAlertDuration4h, "never": godo.InsightsReAlertDurationNever,
	}
	if mapped, ok := values[strings.ToLower(value)]; ok {
		return mapped, nil
	}
	return "", fmt.Errorf("--re-alert-duration must be one of 30m, 1h, 4h, or never")
}

func alertRuleStatus(value string) (string, error) {
	switch strings.ToLower(value) {
	case "active":
		return godo.InsightsAlertRuleStatusActive, nil
	case "paused":
		return godo.InsightsAlertRuleStatusPaused, nil
	default:
		return "", fmt.Errorf("--status must be active or paused")
	}
}

func validAlertWindow(value string) bool {
	_, err := alertWindowFromAPI(value)
	return err == nil
}

func alertWindowFromAPI(value string) (string, error) {
	valid := map[string]string{
		godo.InsightsEvaluationWindow1m: "1m", godo.InsightsEvaluationWindow5m: "5m",
		godo.InsightsEvaluationWindow10m: "10m", godo.InsightsEvaluationWindow15m: "15m",
		godo.InsightsEvaluationWindow30m: "30m", godo.InsightsEvaluationWindow1h: "1h",
	}
	if mapped, ok := valid[value]; ok {
		return mapped, nil
	}
	return "", fmt.Errorf("invalid alert window")
}

func validAlertThresholdOperator(value string) bool {
	valid := []string{
		godo.InsightsThresholdOperatorEqual, godo.InsightsThresholdOperatorNotEqual,
		godo.InsightsThresholdOperatorLessThan, godo.InsightsThresholdOperatorLessThanOrEqual,
		godo.InsightsThresholdOperatorGreaterThan, godo.InsightsThresholdOperatorGreaterThanOrEqual,
	}
	return slices.Contains(valid, value)
}

func validAlertFilterOperator(value string) bool {
	valid := []string{
		godo.InsightsFilterOperatorEqual, godo.InsightsFilterOperatorNotEqual,
		godo.InsightsFilterOperatorLessThan, godo.InsightsFilterOperatorLessThanOrEqual,
		godo.InsightsFilterOperatorGreaterThan, godo.InsightsFilterOperatorGreaterThanOrEqual,
	}
	return slices.Contains(valid, value)
}

func validAlertReAlertDuration(value string) bool {
	return slices.Contains([]string{
		godo.InsightsReAlertDuration30m, godo.InsightsReAlertDuration1h,
		godo.InsightsReAlertDuration4h, godo.InsightsReAlertDurationNever,
	}, value)
}

func validAlertRuleStatus(value string) bool {
	return value == godo.InsightsAlertRuleStatusActive || value == godo.InsightsAlertRuleStatusPaused
}

package godo

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/go-querystring/query"
)

const (
	insightsNotificationChannelsPath = "/v2/insights/notification-channels"
	insightsAlertRulesPath           = "/v2/insights/alert-rules"
	insightsAlertInstancesPath       = "/v2/insights/alert-instances"
)

// Insights notification channel types. These are the protobuf enum values the
// API returns on read; create and update select the type by which config
// object is set.
const (
	InsightsChannelTypeEmail   = "CHANNEL_TYPE_EMAIL"
	InsightsChannelTypeSlack   = "CHANNEL_TYPE_SLACK"
	InsightsChannelTypeWebhook = "CHANNEL_TYPE_WEBHOOK"
)

// Insights alert rule statuses.
const (
	InsightsAlertRuleStatusActive = "ALERT_RULE_STATUS_ACTIVE"
	InsightsAlertRuleStatusPaused = "ALERT_RULE_STATUS_PAUSED"
)

// Insights alert evaluation windows.
const (
	InsightsEvaluationWindow1m  = "EVALUATION_WINDOW_1M"
	InsightsEvaluationWindow5m  = "EVALUATION_WINDOW_5M"
	InsightsEvaluationWindow10m = "EVALUATION_WINDOW_10M"
	InsightsEvaluationWindow15m = "EVALUATION_WINDOW_15M"
	InsightsEvaluationWindow30m = "EVALUATION_WINDOW_30M"
	InsightsEvaluationWindow1h  = "EVALUATION_WINDOW_1H"
)

// Insights threshold operators.
const (
	InsightsThresholdOperatorEqual              = "THRESHOLD_OPERATOR_EQUAL"
	InsightsThresholdOperatorLessThanOrEqual    = "THRESHOLD_OPERATOR_LESS_THAN_OR_EQUAL"
	InsightsThresholdOperatorLessThan           = "THRESHOLD_OPERATOR_LESS_THAN"
	InsightsThresholdOperatorGreaterThan        = "THRESHOLD_OPERATOR_GREATER_THAN"
	InsightsThresholdOperatorGreaterThanOrEqual = "THRESHOLD_OPERATOR_GREATER_THAN_OR_EQUAL"
	InsightsThresholdOperatorNotEqual           = "THRESHOLD_OPERATOR_NOT_EQUAL"
)

// Insights metrics-query filter operators.
const (
	InsightsFilterOperatorEqual              = "FILTER_OPERATOR_EQUAL"
	InsightsFilterOperatorNotEqual           = "FILTER_OPERATOR_NOT_EQUAL"
	InsightsFilterOperatorLessThan           = "FILTER_OPERATOR_LESS_THAN"
	InsightsFilterOperatorLessThanOrEqual    = "FILTER_OPERATOR_LESS_THAN_OR_EQUAL"
	InsightsFilterOperatorGreaterThan        = "FILTER_OPERATOR_GREATER_THAN"
	InsightsFilterOperatorGreaterThanOrEqual = "FILTER_OPERATOR_GREATER_THAN_OR_EQUAL"
)

// Insights re-alert durations.
const (
	InsightsReAlertDuration30m   = "RE_ALERT_DURATION_30M"
	InsightsReAlertDuration1h    = "RE_ALERT_DURATION_1H"
	InsightsReAlertDuration4h    = "RE_ALERT_DURATION_4H"
	InsightsReAlertDurationNever = "RE_ALERT_DURATION_NEVER"
)

// Insights severities used when binding a notification channel to an alert rule.
const (
	InsightsSeverityWarning  = "SEVERITY_WARNING"
	InsightsSeverityCritical = "SEVERITY_CRITICAL"
)

// Insights alert instance statuses. These are lowercase, unlike the protobuf
// enums used by alert rules and notification channels.
const (
	InsightsAlertInstanceStatusActive   = "active"
	InsightsAlertInstanceStatusResolved = "resolved"
)

// Insights alert instance severities.
const (
	InsightsAlertInstanceSeverityWarning  = "warning"
	InsightsAlertInstanceSeverityCritical = "critical"
)

// InsightsService manages Insights notification channels, alert rules, alert
// instances, and PromQL queries.
type InsightsService interface {
	ListNotificationChannels(context.Context, *ListOptions) ([]NotificationChannel, *Response, error)
	GetNotificationChannel(context.Context, string) (*NotificationChannel, *Response, error)
	CreateNotificationChannel(context.Context, *NotificationChannelRequest) (*NotificationChannel, *Response, error)
	UpdateNotificationChannel(context.Context, string, *NotificationChannelRequest) (*NotificationChannel, *Response, error)
	DeleteNotificationChannel(context.Context, string) (*Response, error)

	ListAlertRules(context.Context, *AlertRuleListOptions) ([]AlertRule, *Response, error)
	GetAlertRule(context.Context, string) (*AlertRule, *Response, error)
	CreateAlertRule(context.Context, *AlertRuleRequest) (*AlertRule, *Response, error)
	UpdateAlertRule(context.Context, string, *AlertRuleRequest) (*AlertRule, *Response, error)
	DeleteAlertRule(context.Context, string) (*Response, error)

	ListAlertInstances(context.Context, *AlertInstanceListOptions) ([]AlertInstance, *Response, error)
	GetAlertInstance(context.Context, string) (*AlertInstance, *Response, error)

	// Query evaluates a PromQL expression at one timestamp (GET).
	Query(context.Context, string, *PromQueryOptions) (*PromQueryResponse, *Response, error)
	// PostQuery is the form-encoded POST form of Query, for expressions that do not fit in a URL.
	PostQuery(context.Context, string, *PromQueryOptions) (*PromQueryResponse, *Response, error)
	QueryRange(context.Context, string, *PromQueryRangeOptions) (*PromQueryRangeResponse, *Response, error)
	PostQueryRange(context.Context, string, *PromQueryRangeOptions) (*PromQueryRangeResponse, *Response, error)
	Series(context.Context, string, *PromSelectorOptions) (*PromSeriesResponse, *Response, error)
	PostSeries(context.Context, string, *PromSelectorOptions) (*PromSeriesResponse, *Response, error)
	Labels(context.Context, string, *PromSelectorOptions) (*PromLabelsResponse, *Response, error)
	PostLabels(context.Context, string, *PromSelectorOptions) (*PromLabelsResponse, *Response, error)
	LabelValues(context.Context, string, string, *PromSelectorOptions) (*PromLabelsResponse, *Response, error)
}

// InsightsServiceOp handles communication with Insights methods of the DigitalOcean API.
type InsightsServiceOp struct {
	client *Client
}

var _ InsightsService = &InsightsServiceOp{}

// NotificationChannel is a reusable Insights notification destination.
type NotificationChannel struct {
	ID          string                     `json:"id"`
	Name        string                     `json:"name"`
	ChannelType string                     `json:"channel_type"`
	Email       *EmailNotificationConfig   `json:"email,omitempty"`
	Slack       *SlackNotificationConfig   `json:"slack,omitempty"`
	Webhook     *WebhookNotificationConfig `json:"webhook,omitempty"`
	Usage       *NotificationChannelUsage  `json:"usage,omitempty"`
	CreatedAt   time.Time                  `json:"created_at"`
	UpdatedAt   time.Time                  `json:"updated_at"`
}

// NotificationChannelRequest is the body for creating or updating a notification channel.
// Set exactly one of Email, Slack, or Webhook. Secret fields are write-only:
// send the full value to set or rotate one, and leave it empty on update to keep the current secret.
type NotificationChannelRequest struct {
	Name    string                     `json:"name"`
	Email   *EmailNotificationConfig   `json:"email,omitempty"`
	Slack   *SlackNotificationConfig   `json:"slack,omitempty"`
	Webhook *WebhookNotificationConfig `json:"webhook,omitempty"`
}

// EmailNotificationConfig is an email notification channel.
// To is one or more verified team-member addresses separated by commas, semicolons, or spaces.
type EmailNotificationConfig struct {
	To string `json:"to"`
}

// SlackNotificationConfig is a Slack notification channel.
// WebhookURL is write-only: reads return a masked value, and an empty WebhookURL on update keeps the existing URL.
type SlackNotificationConfig struct {
	WebhookURL string `json:"webhook_url,omitempty"`
	Channel    string `json:"channel"`
}

// WebhookNotificationConfig is an HTTPS webhook notification channel.
// URL is returned in full on read. Credential fields are write-only and masked on read;
// leave a secret empty on update to keep the existing value. Set BasicAuth or BearerToken, not both.
type WebhookNotificationConfig struct {
	URL         string                  `json:"url"`
	BasicAuth   *WebhookBasicAuth       `json:"basic_auth,omitempty"`
	BearerToken *WebhookBearerToken     `json:"bearer_token,omitempty"`
	Headers     map[string]string       `json:"headers,omitempty"`
	Signature   *WebhookSignatureConfig `json:"signature,omitempty"`
}

// WebhookBasicAuth is HTTP basic auth for a webhook. Password is write-only.
type WebhookBasicAuth struct {
	Username string `json:"username"`
	Password string `json:"password,omitempty"`
}

// WebhookBearerToken is bearer auth for a webhook. Token is write-only.
type WebhookBearerToken struct {
	Token string `json:"token,omitempty"`
}

// WebhookSignatureConfig is the HMAC signing secret for a webhook. Secret is write-only.
type WebhookSignatureConfig struct {
	Secret string `json:"secret,omitempty"`
}

// NotificationChannelUsage counts alert rules that reference a channel.
type NotificationChannelUsage struct {
	RuleCount int `json:"rule_count"`
}

type notificationChannelsRoot struct {
	NotificationChannels []NotificationChannel `json:"notification_channels"`
	Links                *Links                `json:"links"`
	Meta                 *Meta                 `json:"meta"`
}

type notificationChannelRoot struct {
	NotificationChannel *NotificationChannel `json:"notification_channel"`
}

// ListNotificationChannels lists notification channels.
func (s *InsightsServiceOp) ListNotificationChannels(ctx context.Context, opt *ListOptions) ([]NotificationChannel, *Response, error) {
	path, err := addOptions(insightsNotificationChannelsPath, opt)
	if err != nil {
		return nil, nil, err
	}

	req, err := s.client.NewRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, err
	}

	root := new(notificationChannelsRoot)
	resp, err := s.client.Do(ctx, req, root)
	if err != nil {
		return nil, resp, err
	}
	setListPaging(resp, root.Links, root.Meta)
	return root.NotificationChannels, resp, nil
}

// GetNotificationChannel retrieves a notification channel. Secret fields are masked.
func (s *InsightsServiceOp) GetNotificationChannel(ctx context.Context, id string) (*NotificationChannel, *Response, error) {
	req, err := s.client.NewRequest(ctx, http.MethodGet, insightsResourcePath(insightsNotificationChannelsPath, id), nil)
	if err != nil {
		return nil, nil, err
	}

	root := new(notificationChannelRoot)
	resp, err := s.client.Do(ctx, req, root)
	if err != nil {
		return nil, resp, err
	}
	return root.NotificationChannel, resp, nil
}

// CreateNotificationChannel creates a notification channel.
func (s *InsightsServiceOp) CreateNotificationChannel(ctx context.Context, create *NotificationChannelRequest) (*NotificationChannel, *Response, error) {
	req, err := s.client.NewRequest(ctx, http.MethodPost, insightsNotificationChannelsPath, create)
	if err != nil {
		return nil, nil, err
	}

	root := new(notificationChannelRoot)
	resp, err := s.client.Do(ctx, req, root)
	if err != nil {
		return nil, resp, err
	}
	return root.NotificationChannel, resp, nil
}

// UpdateNotificationChannel replaces a notification channel's name and config.
func (s *InsightsServiceOp) UpdateNotificationChannel(ctx context.Context, id string, update *NotificationChannelRequest) (*NotificationChannel, *Response, error) {
	req, err := s.client.NewRequest(ctx, http.MethodPut, insightsResourcePath(insightsNotificationChannelsPath, id), update)
	if err != nil {
		return nil, nil, err
	}

	root := new(notificationChannelRoot)
	resp, err := s.client.Do(ctx, req, root)
	if err != nil {
		return nil, resp, err
	}
	return root.NotificationChannel, resp, nil
}

// DeleteNotificationChannel deletes a notification channel.
// Deleting a channel that alert rules still reference returns 409 Conflict.
func (s *InsightsServiceOp) DeleteNotificationChannel(ctx context.Context, id string) (*Response, error) {
	req, err := s.client.NewRequest(ctx, http.MethodDelete, insightsResourcePath(insightsNotificationChannelsPath, id), nil)
	if err != nil {
		return nil, err
	}
	return s.client.Do(ctx, req, nil)
}

// AlertRuleListOptions are filters for listing alert rules.
type AlertRuleListOptions struct {
	Page        int    `url:"page,omitempty"`
	PerPage     int    `url:"per_page,omitempty"`
	ResourceURN string `url:"resource_urn,omitempty"`
}

// AlertRule is an Insights alert rule.
type AlertRule struct {
	ID        string        `json:"id"`
	Spec      AlertRuleSpec `json:"spec"`
	Status    string        `json:"status"`
	CreatedAt time.Time     `json:"created_at"`
	UpdatedAt time.Time     `json:"updated_at"`
}

// AlertRuleRequest is the body for creating or updating an alert rule.
// Status is optional: create defaults to active, and update keeps the current status when empty.
type AlertRuleRequest struct {
	Spec   AlertRuleSpec `json:"spec"`
	Status string        `json:"status,omitempty"`
}

// AlertRuleSpec is the configuration of an alert rule.
type AlertRuleSpec struct {
	Name       string            `json:"name"`
	Query      AlertMetricsQuery `json:"query"`
	Condition  *AlertCondition   `json:"condition,omitempty"`
	Thresholds AlertThresholds   `json:"thresholds"`
	// NotificationChannels are notified when the rule fires.
	// On create, set a non-empty list. On update, leave nil to keep existing bindings.
	// A pointer to an empty slice is sent as [] and the API rejects it.
	NotificationChannels *[]NotificationChannelBinding `json:"notification_channels,omitempty"`
	ReAlertDuration      string                        `json:"re_alert_duration,omitempty"`
}

// AlertCondition is the evaluation window for an alert rule.
type AlertCondition struct {
	Window string `json:"window,omitempty"`
}

// AlertThresholds is the comparison applied to the aggregated metric.
// At least one of Warning or Critical must be set. Use a pointer so a threshold of 0 is sent.
type AlertThresholds struct {
	Warning  *float64 `json:"warning,omitempty"`
	Critical *float64 `json:"critical,omitempty"`
	Operator string   `json:"operator"`
}

// AlertMetricsQuery is the metric an alert rule evaluates.
// Metric must be a dotted OpenTelemetry name such as do.droplets.cpu_utilization.
type AlertMetricsQuery struct {
	Metric       string             `json:"metric"`
	Filters      []AlertQueryFilter `json:"filters,omitempty"`
	ResourceURNs []string           `json:"resource_urns,omitempty"`
	Tags         []string           `json:"tags,omitempty"`
}

// AlertQueryFilter is a label filter on an alert metrics query.
type AlertQueryFilter struct {
	Field    string `json:"field"`
	Operator string `json:"operator"`
	Value    string `json:"value"`
}

// NotificationChannelBinding attaches a notification channel to an alert rule.
// An empty NotifyOn list notifies on every severity.
type NotificationChannelBinding struct {
	NotificationChannelID string   `json:"notification_channel_id"`
	NotifyOn              []string `json:"notify_on,omitempty"`
}

type alertRulesRoot struct {
	AlertRules []AlertRule `json:"alert_rules"`
	Links      *Links      `json:"links"`
	Meta       *Meta       `json:"meta"`
}

type alertRuleRoot struct {
	AlertRule *AlertRule `json:"alert_rule"`
}

// ListAlertRules lists alert rules.
func (s *InsightsServiceOp) ListAlertRules(ctx context.Context, opt *AlertRuleListOptions) ([]AlertRule, *Response, error) {
	path, err := addOptions(insightsAlertRulesPath, opt)
	if err != nil {
		return nil, nil, err
	}

	req, err := s.client.NewRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, err
	}

	root := new(alertRulesRoot)
	resp, err := s.client.Do(ctx, req, root)
	if err != nil {
		return nil, resp, err
	}
	setListPaging(resp, root.Links, root.Meta)
	return root.AlertRules, resp, nil
}

// GetAlertRule retrieves an alert rule.
func (s *InsightsServiceOp) GetAlertRule(ctx context.Context, id string) (*AlertRule, *Response, error) {
	req, err := s.client.NewRequest(ctx, http.MethodGet, insightsResourcePath(insightsAlertRulesPath, id), nil)
	if err != nil {
		return nil, nil, err
	}

	root := new(alertRuleRoot)
	resp, err := s.client.Do(ctx, req, root)
	if err != nil {
		return nil, resp, err
	}
	return root.AlertRule, resp, nil
}

// CreateAlertRule creates an alert rule.
func (s *InsightsServiceOp) CreateAlertRule(ctx context.Context, create *AlertRuleRequest) (*AlertRule, *Response, error) {
	req, err := s.client.NewRequest(ctx, http.MethodPost, insightsAlertRulesPath, create)
	if err != nil {
		return nil, nil, err
	}

	root := new(alertRuleRoot)
	resp, err := s.client.Do(ctx, req, root)
	if err != nil {
		return nil, resp, err
	}
	return root.AlertRule, resp, nil
}

// UpdateAlertRule updates an alert rule. Omitted status, re-alert duration, and
// notification channel bindings are left unchanged.
func (s *InsightsServiceOp) UpdateAlertRule(ctx context.Context, id string, update *AlertRuleRequest) (*AlertRule, *Response, error) {
	req, err := s.client.NewRequest(ctx, http.MethodPut, insightsResourcePath(insightsAlertRulesPath, id), update)
	if err != nil {
		return nil, nil, err
	}

	root := new(alertRuleRoot)
	resp, err := s.client.Do(ctx, req, root)
	if err != nil {
		return nil, resp, err
	}
	return root.AlertRule, resp, nil
}

// DeleteAlertRule deletes an alert rule.
func (s *InsightsServiceOp) DeleteAlertRule(ctx context.Context, id string) (*Response, error) {
	req, err := s.client.NewRequest(ctx, http.MethodDelete, insightsResourcePath(insightsAlertRulesPath, id), nil)
	if err != nil {
		return nil, err
	}
	return s.client.Do(ctx, req, nil)
}

// AlertInstanceListOptions are filters for listing alert instances.
type AlertInstanceListOptions struct {
	Page        int    `url:"page,omitempty"`
	PerPage     int    `url:"per_page,omitempty"`
	Status      string `url:"status,omitempty"`
	RuleID      string `url:"rule_id,omitempty"`
	ResourceURN string `url:"resource_urn,omitempty"`
}

// AlertInstance is a read-only firing of an alert rule against a resource.
type AlertInstance struct {
	ID              string     `json:"id"`
	RuleID          string     `json:"rule_id"`
	Severity        string     `json:"severity"`
	Status          string     `json:"status"`
	ResourceURN     string     `json:"resource_urn,omitempty"`
	Value           float64    `json:"value"`
	TriggeredAt     time.Time  `json:"triggered_at"`
	LastTriggeredAt time.Time  `json:"last_triggered_at"`
	ResolvedAt      *time.Time `json:"resolved_at,omitempty"`
	LastNotifiedAt  *time.Time `json:"last_notified_at,omitempty"`
}

type alertInstancesRoot struct {
	AlertInstances []AlertInstance `json:"alert_instances"`
	Links          *Links          `json:"links"`
	Meta           *Meta           `json:"meta"`
}

type alertInstanceRoot struct {
	AlertInstance *AlertInstance `json:"alert_instance"`
}

// ListAlertInstances lists alert instances, newest trigger first.
func (s *InsightsServiceOp) ListAlertInstances(ctx context.Context, opt *AlertInstanceListOptions) ([]AlertInstance, *Response, error) {
	path, err := addOptions(insightsAlertInstancesPath, opt)
	if err != nil {
		return nil, nil, err
	}

	req, err := s.client.NewRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, err
	}

	root := new(alertInstancesRoot)
	resp, err := s.client.Do(ctx, req, root)
	if err != nil {
		return nil, resp, err
	}
	setListPaging(resp, root.Links, root.Meta)
	return root.AlertInstances, resp, nil
}

// GetAlertInstance retrieves an alert instance.
func (s *InsightsServiceOp) GetAlertInstance(ctx context.Context, id string) (*AlertInstance, *Response, error) {
	req, err := s.client.NewRequest(ctx, http.MethodGet, insightsResourcePath(insightsAlertInstancesPath, id), nil)
	if err != nil {
		return nil, nil, err
	}

	root := new(alertInstanceRoot)
	resp, err := s.client.Do(ctx, req, root)
	if err != nil {
		return nil, resp, err
	}
	return root.AlertInstance, resp, nil
}

// PromQueryOptions are parameters for an instant PromQL query.
// Time and Timeout accept Prometheus timestamps and duration strings.
type PromQueryOptions struct {
	Query   string `url:"query"`
	Time    string `url:"time,omitempty"`
	Timeout string `url:"timeout,omitempty"`
}

// PromQueryRangeOptions are parameters for a range PromQL query.
type PromQueryRangeOptions struct {
	Query   string `url:"query"`
	Start   string `url:"start"`
	End     string `url:"end"`
	Step    string `url:"step"`
	Timeout string `url:"timeout,omitempty"`
}

// PromSelectorOptions are parameters for series, label, and label-value lookups.
// Match is repeated as match[] selectors.
type PromSelectorOptions struct {
	Match []string `url:"match,brackets,omitempty"`
	Start string   `url:"start,omitempty"`
	End   string   `url:"end,omitempty"`
}

// PromQueryResponse is a successful instant PromQL query.
// ResultType is vector, matrix, scalar, or string. Exactly one of Vector, Matrix, or Sample is set.
type PromQueryResponse struct {
	Status     string
	ResultType string
	Vector     []PromVectorSample
	Matrix     []PromMatrixSample
	Sample     *PromSample
}

// PromQueryRangeResponse is a successful range PromQL query. Data.ResultType is matrix.
type PromQueryRangeResponse struct {
	Status string           `json:"status"`
	Data   PromMatrixResult `json:"data"`
}

// PromMatrixResult is the data object of a range query.
type PromMatrixResult struct {
	ResultType string             `json:"resultType"`
	Result     []PromMatrixSample `json:"result"`
}

// PromVectorSample is one series from an instant vector result.
type PromVectorSample struct {
	Metric map[string]string `json:"metric"`
	Value  PromSample        `json:"value"`
}

// PromMatrixSample is one series from a range result.
type PromMatrixSample struct {
	Metric map[string]string `json:"metric"`
	Values []PromSample      `json:"values"`
}

// PromSample is a Prometheus sample pair [timestamp_seconds, value].
// Value is the raw string from the API, including non-numeric string results.
type PromSample struct {
	Timestamp float64
	Value     string
}

// PromSeriesResponse lists series matching one or more selectors.
type PromSeriesResponse struct {
	Status string              `json:"status"`
	Data   []map[string]string `json:"data"`
}

// PromLabelsResponse lists label names or the values of one label.
type PromLabelsResponse struct {
	Status string   `json:"status"`
	Data   []string `json:"data"`
}

// UnmarshalJSON decodes a Prometheus instant-query envelope.
func (r *PromQueryResponse) UnmarshalJSON(data []byte) error {
	var raw struct {
		Status string `json:"status"`
		Data   struct {
			ResultType string          `json:"resultType"`
			Result     json.RawMessage `json:"result"`
		} `json:"data"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	r.Status = raw.Status
	r.ResultType = raw.Data.ResultType
	if len(raw.Data.Result) == 0 || string(raw.Data.Result) == "null" {
		return nil
	}
	switch raw.Data.ResultType {
	case "vector":
		return json.Unmarshal(raw.Data.Result, &r.Vector)
	case "matrix":
		return json.Unmarshal(raw.Data.Result, &r.Matrix)
	case "scalar", "string":
		r.Sample = &PromSample{}
		return json.Unmarshal(raw.Data.Result, r.Sample)
	default:
		return fmt.Errorf("godo: unknown PromQL result type %q", raw.Data.ResultType)
	}
}

// UnmarshalJSON decodes a Prometheus [timestamp, value] sample.
func (s *PromSample) UnmarshalJSON(data []byte) error {
	var pair []json.RawMessage
	if err := json.Unmarshal(data, &pair); err != nil {
		return err
	}
	if len(pair) != 2 {
		return fmt.Errorf("godo: prom sample has %d elements", len(pair))
	}
	if err := json.Unmarshal(pair[0], &s.Timestamp); err != nil {
		return err
	}
	return json.Unmarshal(pair[1], &s.Value)
}

// Query evaluates a PromQL expression at a single timestamp.
func (s *InsightsServiceOp) Query(ctx context.Context, region string, opt *PromQueryOptions) (*PromQueryResponse, *Response, error) {
	return s.promGet(ctx, insightsPromPath(region, "query"), opt)
}

// PostQuery evaluates a PromQL expression at a single timestamp using a form body.
func (s *InsightsServiceOp) PostQuery(ctx context.Context, region string, opt *PromQueryOptions) (*PromQueryResponse, *Response, error) {
	return s.promPost(ctx, insightsPromPath(region, "query"), opt)
}

// QueryRange evaluates a PromQL expression over a time range.
func (s *InsightsServiceOp) QueryRange(ctx context.Context, region string, opt *PromQueryRangeOptions) (*PromQueryRangeResponse, *Response, error) {
	out := new(PromQueryRangeResponse)
	resp, err := s.promDo(ctx, http.MethodGet, insightsPromPath(region, "query_range"), opt, out)
	if err != nil {
		return nil, resp, err
	}
	return out, resp, nil
}

// PostQueryRange evaluates a PromQL expression over a time range using a form body.
func (s *InsightsServiceOp) PostQueryRange(ctx context.Context, region string, opt *PromQueryRangeOptions) (*PromQueryRangeResponse, *Response, error) {
	out := new(PromQueryRangeResponse)
	resp, err := s.promDo(ctx, http.MethodPost, insightsPromPath(region, "query_range"), opt, out)
	if err != nil {
		return nil, resp, err
	}
	return out, resp, nil
}

// Series finds series matching label selectors.
func (s *InsightsServiceOp) Series(ctx context.Context, region string, opt *PromSelectorOptions) (*PromSeriesResponse, *Response, error) {
	out := new(PromSeriesResponse)
	resp, err := s.promDo(ctx, http.MethodGet, insightsPromPath(region, "series"), opt, out)
	if err != nil {
		return nil, resp, err
	}
	return out, resp, nil
}

// PostSeries finds series matching label selectors using a form body.
func (s *InsightsServiceOp) PostSeries(ctx context.Context, region string, opt *PromSelectorOptions) (*PromSeriesResponse, *Response, error) {
	out := new(PromSeriesResponse)
	resp, err := s.promDo(ctx, http.MethodPost, insightsPromPath(region, "series"), opt, out)
	if err != nil {
		return nil, resp, err
	}
	return out, resp, nil
}

// Labels lists label names.
func (s *InsightsServiceOp) Labels(ctx context.Context, region string, opt *PromSelectorOptions) (*PromLabelsResponse, *Response, error) {
	out := new(PromLabelsResponse)
	resp, err := s.promDo(ctx, http.MethodGet, insightsPromPath(region, "labels"), opt, out)
	if err != nil {
		return nil, resp, err
	}
	return out, resp, nil
}

// PostLabels lists label names using a form body.
func (s *InsightsServiceOp) PostLabels(ctx context.Context, region string, opt *PromSelectorOptions) (*PromLabelsResponse, *Response, error) {
	out := new(PromLabelsResponse)
	resp, err := s.promDo(ctx, http.MethodPost, insightsPromPath(region, "labels"), opt, out)
	if err != nil {
		return nil, resp, err
	}
	return out, resp, nil
}

// LabelValues lists values for a label name.
func (s *InsightsServiceOp) LabelValues(ctx context.Context, region, name string, opt *PromSelectorOptions) (*PromLabelsResponse, *Response, error) {
	out := new(PromLabelsResponse)
	path := fmt.Sprintf("/v2/insights/query/%s/prom/api/v1/label/%s/values", url.PathEscape(region), url.PathEscape(name))
	resp, err := s.promDo(ctx, http.MethodGet, path, opt, out)
	if err != nil {
		return nil, resp, err
	}
	return out, resp, nil
}

func (s *InsightsServiceOp) promGet(ctx context.Context, path string, opt interface{}) (*PromQueryResponse, *Response, error) {
	out := new(PromQueryResponse)
	resp, err := s.promDo(ctx, http.MethodGet, path, opt, out)
	if err != nil {
		return nil, resp, err
	}
	return out, resp, nil
}

func (s *InsightsServiceOp) promPost(ctx context.Context, path string, opt interface{}) (*PromQueryResponse, *Response, error) {
	out := new(PromQueryResponse)
	resp, err := s.promDo(ctx, http.MethodPost, path, opt, out)
	if err != nil {
		return nil, resp, err
	}
	return out, resp, nil
}

func (s *InsightsServiceOp) promDo(ctx context.Context, method, path string, opt, out interface{}) (*Response, error) {
	var (
		req *http.Request
		err error
	)
	switch method {
	case http.MethodGet:
		path, err = addOptions(path, opt)
		if err != nil {
			return nil, err
		}
		req, err = s.client.NewRequest(ctx, http.MethodGet, path, nil)
	case http.MethodPost:
		req, err = s.newFormRequest(ctx, path, opt)
	default:
		return nil, fmt.Errorf("godo: unsupported prom method %s", method)
	}
	if err != nil {
		return nil, err
	}
	return s.client.Do(ctx, req, out)
}

func (s *InsightsServiceOp) newFormRequest(ctx context.Context, path string, opt interface{}) (*http.Request, error) {
	req, err := s.client.NewRequest(ctx, http.MethodPost, path, nil)
	if err != nil {
		return nil, err
	}
	encoded := ""
	if opt != nil {
		values, err := query.Values(opt)
		if err != nil {
			return nil, err
		}
		encoded = values.Encode()
	}
	req.Body = io.NopCloser(strings.NewReader(encoded))
	req.ContentLength = int64(len(encoded))
	req.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(strings.NewReader(encoded)), nil
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return req, nil
}

func insightsResourcePath(base, id string) string {
	return base + "/" + url.PathEscape(id)
}

func insightsPromPath(region, suffix string) string {
	return fmt.Sprintf("/v2/insights/query/%s/prom/api/v1/%s", url.PathEscape(region), suffix)
}

func setListPaging(resp *Response, links *Links, meta *Meta) {
	if resp == nil {
		return
	}
	if links != nil {
		resp.Links = links
	}
	if meta != nil {
		resp.Meta = meta
	}
}

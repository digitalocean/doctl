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

package do

import (
	"context"

	"github.com/digitalocean/godo"
)

// InsightsService is an interface for interacting with the Insights API.
type InsightsService interface {
	ListNotificationChannels(*godo.ListOptions) ([]godo.NotificationChannel, *godo.Response, error)
	GetNotificationChannel(string) (*godo.NotificationChannel, error)
	CreateNotificationChannel(*godo.NotificationChannelRequest) (*godo.NotificationChannel, error)
	UpdateNotificationChannel(string, *godo.NotificationChannelRequest) (*godo.NotificationChannel, error)
	DeleteNotificationChannel(string) error

	ListAlertRules(*godo.AlertRuleListOptions) ([]godo.AlertRule, *godo.Response, error)
	GetAlertRule(string) (*godo.AlertRule, error)
	CreateAlertRule(*godo.AlertRuleRequest) (*godo.AlertRule, error)
	UpdateAlertRule(string, *godo.AlertRuleRequest) (*godo.AlertRule, error)
	DeleteAlertRule(string) error

	ListAlertInstances(*godo.AlertInstanceListOptions) ([]godo.AlertInstance, *godo.Response, error)
	GetAlertInstance(string) (*godo.AlertInstance, error)

	Query(string, *godo.PromQueryOptions) (*godo.PromQueryResponse, error)
	PostQuery(string, *godo.PromQueryOptions) (*godo.PromQueryResponse, error)
	QueryRange(string, *godo.PromQueryRangeOptions) (*godo.PromQueryRangeResponse, error)
	PostQueryRange(string, *godo.PromQueryRangeOptions) (*godo.PromQueryRangeResponse, error)
	Series(string, *godo.PromSelectorOptions) (*godo.PromSeriesResponse, error)
	PostSeries(string, *godo.PromSelectorOptions) (*godo.PromSeriesResponse, error)
	Labels(string, *godo.PromSelectorOptions) (*godo.PromLabelsResponse, error)
	PostLabels(string, *godo.PromSelectorOptions) (*godo.PromLabelsResponse, error)
	LabelValues(string, string, *godo.PromSelectorOptions) (*godo.PromLabelsResponse, error)

	SearchLogs(string, *godo.LogsSearchRequest) (*godo.LogsSearchResponse, error)
	SearchSpans(string, *godo.SpansSearchRequest) (*godo.SpansSearchResponse, error)
	SearchTraces(string, *godo.TracesSearchRequest) (*godo.TracesSearchResponse, error)
	GetTrace(string, string, *godo.TraceGetOptions) (*godo.Trace, error)
}

type insightsService struct {
	client *godo.Client
}

var _ InsightsService = &insightsService{}

// NewInsightsService builds an InsightsService instance.
func NewInsightsService(client *godo.Client) InsightsService {
	return &insightsService{client: client}
}

func (s *insightsService) ListNotificationChannels(opt *godo.ListOptions) ([]godo.NotificationChannel, *godo.Response, error) {
	return s.client.Insights.ListNotificationChannels(context.TODO(), opt)
}

func (s *insightsService) GetNotificationChannel(id string) (*godo.NotificationChannel, error) {
	channel, _, err := s.client.Insights.GetNotificationChannel(context.TODO(), id)
	return channel, err
}

func (s *insightsService) CreateNotificationChannel(req *godo.NotificationChannelRequest) (*godo.NotificationChannel, error) {
	channel, _, err := s.client.Insights.CreateNotificationChannel(context.TODO(), req)
	return channel, err
}

func (s *insightsService) UpdateNotificationChannel(id string, req *godo.NotificationChannelRequest) (*godo.NotificationChannel, error) {
	channel, _, err := s.client.Insights.UpdateNotificationChannel(context.TODO(), id, req)
	return channel, err
}

func (s *insightsService) DeleteNotificationChannel(id string) error {
	_, err := s.client.Insights.DeleteNotificationChannel(context.TODO(), id)
	return err
}

func (s *insightsService) ListAlertRules(opt *godo.AlertRuleListOptions) ([]godo.AlertRule, *godo.Response, error) {
	return s.client.Insights.ListAlertRules(context.TODO(), opt)
}

func (s *insightsService) GetAlertRule(id string) (*godo.AlertRule, error) {
	rule, _, err := s.client.Insights.GetAlertRule(context.TODO(), id)
	return rule, err
}

func (s *insightsService) CreateAlertRule(req *godo.AlertRuleRequest) (*godo.AlertRule, error) {
	rule, _, err := s.client.Insights.CreateAlertRule(context.TODO(), req)
	return rule, err
}

func (s *insightsService) UpdateAlertRule(id string, req *godo.AlertRuleRequest) (*godo.AlertRule, error) {
	rule, _, err := s.client.Insights.UpdateAlertRule(context.TODO(), id, req)
	return rule, err
}

func (s *insightsService) DeleteAlertRule(id string) error {
	_, err := s.client.Insights.DeleteAlertRule(context.TODO(), id)
	return err
}

func (s *insightsService) ListAlertInstances(opt *godo.AlertInstanceListOptions) ([]godo.AlertInstance, *godo.Response, error) {
	return s.client.Insights.ListAlertInstances(context.TODO(), opt)
}

func (s *insightsService) GetAlertInstance(id string) (*godo.AlertInstance, error) {
	instance, _, err := s.client.Insights.GetAlertInstance(context.TODO(), id)
	return instance, err
}

func (s *insightsService) Query(region string, opt *godo.PromQueryOptions) (*godo.PromQueryResponse, error) {
	result, _, err := s.client.Insights.Query(context.TODO(), region, opt)
	return result, err
}

func (s *insightsService) PostQuery(region string, opt *godo.PromQueryOptions) (*godo.PromQueryResponse, error) {
	result, _, err := s.client.Insights.PostQuery(context.TODO(), region, opt)
	return result, err
}

func (s *insightsService) QueryRange(region string, opt *godo.PromQueryRangeOptions) (*godo.PromQueryRangeResponse, error) {
	result, _, err := s.client.Insights.QueryRange(context.TODO(), region, opt)
	return result, err
}

func (s *insightsService) PostQueryRange(region string, opt *godo.PromQueryRangeOptions) (*godo.PromQueryRangeResponse, error) {
	result, _, err := s.client.Insights.PostQueryRange(context.TODO(), region, opt)
	return result, err
}

func (s *insightsService) Series(region string, opt *godo.PromSelectorOptions) (*godo.PromSeriesResponse, error) {
	result, _, err := s.client.Insights.Series(context.TODO(), region, opt)
	return result, err
}

func (s *insightsService) PostSeries(region string, opt *godo.PromSelectorOptions) (*godo.PromSeriesResponse, error) {
	result, _, err := s.client.Insights.PostSeries(context.TODO(), region, opt)
	return result, err
}

func (s *insightsService) Labels(region string, opt *godo.PromSelectorOptions) (*godo.PromLabelsResponse, error) {
	result, _, err := s.client.Insights.Labels(context.TODO(), region, opt)
	return result, err
}

func (s *insightsService) PostLabels(region string, opt *godo.PromSelectorOptions) (*godo.PromLabelsResponse, error) {
	result, _, err := s.client.Insights.PostLabels(context.TODO(), region, opt)
	return result, err
}

func (s *insightsService) LabelValues(region, name string, opt *godo.PromSelectorOptions) (*godo.PromLabelsResponse, error) {
	result, _, err := s.client.Insights.LabelValues(context.TODO(), region, name, opt)
	return result, err
}

func (s *insightsService) SearchLogs(region string, req *godo.LogsSearchRequest) (*godo.LogsSearchResponse, error) {
	result, _, err := s.client.Insights.SearchLogs(context.TODO(), region, req)
	return result, err
}

func (s *insightsService) SearchSpans(region string, req *godo.SpansSearchRequest) (*godo.SpansSearchResponse, error) {
	result, _, err := s.client.Insights.SearchSpans(context.TODO(), region, req)
	return result, err
}

func (s *insightsService) SearchTraces(region string, req *godo.TracesSearchRequest) (*godo.TracesSearchResponse, error) {
	result, _, err := s.client.Insights.SearchTraces(context.TODO(), region, req)
	return result, err
}

func (s *insightsService) GetTrace(region, traceID string, opt *godo.TraceGetOptions) (*godo.Trace, error) {
	result, _, err := s.client.Insights.GetTrace(context.TODO(), region, traceID, opt)
	return result, err
}

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
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/commands/displayers"
	"github.com/digitalocean/godo"
	"github.com/spf13/cobra"
)

func insightsSpans() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:     "spans",
			Aliases: []string{"span"},
			Short:   "Display commands for searching Insights spans",
			Long:    "The commands under `doctl insights spans` search individual tracing spans stored in Insights.",
		},
	}

	search := CmdBuilder(cmd, RunInsightsSpansSearch, "search", "Search spans",
		`Searches tracing spans in a region.

Build the search with flags, or pass the complete snake_case public API request as JSON or YAML with `+"`"+`--request-file`+"`"+`. Request files use `+"`"+`time_range`+"`"+` and `+"`"+`order_by`+"`"+`; `+"`"+`--limit`+"`"+` overrides pagination from a request file.

The `+"`"+`--filter`+"`"+` and `+"`"+`--filter-file`+"`"+` values use logs-style expressions with exactly one of `+"`"+`condition`+"`"+`, `+"`"+`and`+"`"+`, `+"`"+`or`+"`"+`, `+"`"+`not`+"`"+`, or `+"`"+`text_search`+"`"+`. Condition values are typed with `+"`"+`string_value`+"`"+`, `+"`"+`number_value`+"`"+`, `+"`"+`bool_value`+"`"+`, `+"`"+`string_array_value`+"`"+`, or `+"`"+`number_array_value`+"`"+`.

Time ranges must not exceed seven days. Results are returned one page at a time; spans pagination reports whether more results matched but does not provide a cursor.`,
		Writer, displayerType(&displayers.InsightsSpans{}))
	search.Args = cobra.NoArgs
	addInsightsRegionFlag(search)
	addInsightsTracingSearchFlags(search)
	search.Example = `The following example searches the last hour of spans in ` + "`" + `nyc3` + "`" + `: doctl insights spans search --region nyc3 --from 1h --order-by start_time:desc --filter '{"condition":{"field":{"name":"service_name"},"operator":"FILTER_OPERATOR_EQ","value":{"string_value":"api"}}}'`

	return cmd
}

// RunInsightsSpansSearch searches tracing spans.
func RunInsightsSpansSearch(c *CmdConfig) error {
	region, err := insightsRequiredString(c, doctl.ArgRegionSlug)
	if err != nil {
		return err
	}
	request, err := buildInsightsSpansSearchRequest(c)
	if err != nil {
		return err
	}
	response, err := c.Insights().SearchSpans(region, request)
	if err != nil {
		return err
	}
	if err := c.Display(&displayers.InsightsSpans{Response: response}); err != nil {
		return err
	}
	if Output == "text" && response != nil && response.Pagination != nil && response.Pagination.HasMore {
		fmt.Fprintln(insightsStderr, "More spans matched than were returned. Narrow the time range or filter to see additional results.")
	}
	return nil
}

func buildInsightsSpansSearchRequest(c *CmdConfig) (*godo.SpansSearchRequest, error) {
	options, err := getInsightsTracingSearchOptions(c)
	if err != nil {
		return nil, err
	}

	request := &godo.SpansSearchRequest{}
	if options.requestFile != "" {
		if err := rejectRequestFileFlagMix(c, options.requestFile, insightsTracingRequestFlags...); err != nil {
			return nil, err
		}
		request, err = readInsightsRequest[godo.SpansSearchRequest](insightsStdin, options.requestFile)
		if err != nil {
			return nil, err
		}
	} else {
		if options.from == "" {
			options.from = insightsTracingDefaultFrom
		}
		if options.to == "" {
			options.to = insightsTracingDefaultTo
		}
		request.TimeRange = godo.SpansTimeRange{
			From: spansTimeBound(options.from),
			To:   spansTimeBound(options.to),
		}

		if options.filter != "" && options.filterFile != "" {
			return nil, fmt.Errorf("--%s and --%s cannot be combined", doctl.ArgInsightsTracingFilter, doctl.ArgInsightsTracingFilterFile)
		}
		switch {
		case options.filter != "":
			request.Filter = &godo.SpansFilterExpression{}
			if err := decodeInsightsJSON(strings.NewReader(options.filter), request.Filter); err != nil {
				return nil, fmt.Errorf("parsing --%s: %w", doctl.ArgInsightsTracingFilter, err)
			}
		case options.filterFile != "":
			request.Filter, err = readInsightsRequest[godo.SpansFilterExpression](insightsStdin, options.filterFile)
			if err != nil {
				return nil, fmt.Errorf("--%s: %w", doctl.ArgInsightsTracingFilterFile, err)
			}
		}

		for _, clause := range options.orderBy {
			field, direction, err := parseInsightsTracingOrderBy(clause)
			if err != nil {
				return nil, err
			}
			request.OrderBy = append(request.OrderBy, godo.SpansOrderBy{
				Field:     godo.SpansFilterField{Name: field},
				Direction: direction,
			})
		}
	}

	if options.limit > 0 {
		if request.Pagination == nil {
			request.Pagination = &godo.SpansPaginationRequest{}
		}
		request.Pagination.Limit = options.limit
	}
	if err := validateSpansSearchRequest(request); err != nil {
		return nil, err
	}
	return request, nil
}

func spansTimeBound(value string) godo.SpansTimeBound {
	if insightsRelativeTimeRE.MatchString(value) {
		return godo.SpansTimeBound{Relative: value}
	}
	return godo.SpansTimeBound{Absolute: value}
}

func validateSpansSearchRequest(request *godo.SpansSearchRequest) error {
	if request == nil {
		return fmt.Errorf("spans search request is required")
	}
	from, err := insightsSearchTimeValue(request.TimeRange.From.Relative, request.TimeRange.From.Absolute)
	if err != nil {
		return fmt.Errorf("invalid spans time_range.from: %w", err)
	}
	to, err := insightsSearchTimeValue(request.TimeRange.To.Relative, request.TimeRange.To.Absolute)
	if err != nil {
		return fmt.Errorf("invalid spans time_range.to: %w", err)
	}
	if err := validateInsightsTracingTimeRange(from, to); err != nil {
		return err
	}
	if request.Filter != nil {
		if err := validateSpansFilterExpression(request.Filter); err != nil {
			return fmt.Errorf("invalid spans filter: %w", err)
		}
	}
	for _, order := range request.OrderBy {
		if strings.TrimSpace(order.Field.Name) == "" {
			return fmt.Errorf("invalid spans order_by: field name is required")
		}
		if !validInsightsFieldScope(order.Field.Scope) {
			return fmt.Errorf("invalid spans order_by field scope %q", order.Field.Scope)
		}
		if order.Direction != "" &&
			order.Direction != godo.InsightsSpansSortDirectionAscending &&
			order.Direction != godo.InsightsSpansSortDirectionDescending {
			return fmt.Errorf("invalid spans order_by direction %q", order.Direction)
		}
	}
	if request.Pagination != nil {
		if err := validateInsightsTracingLimit(request.Pagination.Limit); err != nil {
			return err
		}
	}
	return nil
}

func validateSpansFilterExpression(expression *godo.SpansFilterExpression) error {
	if expression == nil {
		return fmt.Errorf("expression is required")
	}
	encoded, err := json.Marshal(expression)
	if err != nil {
		return err
	}
	var canonical godo.SpansFilterExpression
	if err := json.Unmarshal(encoded, &canonical); err != nil {
		return err
	}
	return validateCanonicalSpansFilterExpression(&canonical)
}

func validateCanonicalSpansFilterExpression(expression *godo.SpansFilterExpression) error {
	switch {
	case expression.Condition != nil:
		condition := expression.Condition
		if strings.TrimSpace(condition.Field.Name) == "" {
			return fmt.Errorf("condition field is required")
		}
		if !validInsightsFieldScope(condition.Field.Scope) {
			return fmt.Errorf("invalid field scope %q", condition.Field.Scope)
		}
		if !slices.Contains([]string{
			godo.InsightsSpansFilterOperatorEqual,
			godo.InsightsSpansFilterOperatorNotEqual,
			godo.InsightsSpansFilterOperatorGreaterThanOrEqual,
			godo.InsightsSpansFilterOperatorLessThanOrEqual,
			godo.InsightsSpansFilterOperatorIn,
			godo.InsightsSpansFilterOperatorExists,
		}, condition.Operator) {
			return fmt.Errorf("invalid condition operator %q", condition.Operator)
		}
		if condition.Operator != godo.InsightsSpansFilterOperatorExists {
			if condition.Value == nil {
				return fmt.Errorf("condition value is required")
			}
			if _, err := json.Marshal(condition.Value); err != nil {
				return err
			}
		}
	case expression.And != nil:
		if len(expression.And.Expressions) == 0 {
			return fmt.Errorf("and expression must contain child expressions")
		}
		for i := range expression.And.Expressions {
			if err := validateCanonicalSpansFilterExpression(&expression.And.Expressions[i]); err != nil {
				return err
			}
		}
	case expression.Or != nil:
		if len(expression.Or.Expressions) == 0 {
			return fmt.Errorf("or expression must contain child expressions")
		}
		for i := range expression.Or.Expressions {
			if err := validateCanonicalSpansFilterExpression(&expression.Or.Expressions[i]); err != nil {
				return err
			}
		}
	case expression.Not != nil:
		return validateCanonicalSpansFilterExpression(expression.Not)
	case expression.TextSearch != nil:
		if strings.TrimSpace(expression.TextSearch.Query) == "" {
			return fmt.Errorf("text_search query is required")
		}
	default:
		return fmt.Errorf("expression must contain exactly one node")
	}
	return nil
}

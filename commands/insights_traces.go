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
	"time"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/commands/displayers"
	"github.com/digitalocean/godo"
	"github.com/spf13/cobra"
)

func insightsTraces() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:     "traces",
			Aliases: []string{"trace"},
			Short:   "Display commands for searching Insights traces",
			Long:    "The commands under `doctl insights traces` search traces and retrieve their spans.",
		},
	}

	search := CmdBuilder(cmd, RunInsightsTracesSearch, "search", "Search traces",
		`Searches trace summaries in a region.

Build the search with flags, or pass the complete snake_case public API request as JSON or YAML with `+"`"+`--request-file`+"`"+`. Request files use `+"`"+`time_range`+"`"+` and `+"`"+`order_by`+"`"+`; `+"`"+`--limit`+"`"+` overrides pagination from a request file.

The `+"`"+`--filter`+"`"+` and `+"`"+`--filter-file`+"`"+` values use logs-style expressions with exactly one of `+"`"+`condition`+"`"+`, `+"`"+`and`+"`"+`, `+"`"+`or`+"`"+`, `+"`"+`not`+"`"+`, or `+"`"+`text_search`+"`"+`. Condition values are typed with `+"`"+`string_value`+"`"+`, `+"`"+`number_value`+"`"+`, `+"`"+`string_array_value`+"`"+`, or `+"`"+`number_array_value`+"`"+`.

Time ranges must not exceed seven days. Results are returned one page at a time; traces pagination reports whether more results matched but does not provide a cursor.`,
		Writer, displayerType(&displayers.InsightsTraces{}))
	search.Args = cobra.NoArgs
	addInsightsRegionFlag(search)
	addInsightsTracingSearchFlags(search)
	search.Example = `The following example searches the last hour of traces in ` + "`" + `nyc3` + "`" + `: doctl insights traces search --region nyc3 --from 1h --order-by started:desc --filter '{"condition":{"field":{"name":"service_name"},"operator":"FILTER_OPERATOR_EQ","value":{"string_value":"api"}}}'`

	get := CmdBuilder(cmd, RunInsightsTraceGet, "get <trace-id>", "Retrieve a trace",
		"Retrieves all available spans for a trace in a region. The optional lookup window must not exceed seven days.",
		Writer, aliasOpt("g"), displayerType(&displayers.InsightsTrace{}))
	get.Args = cobra.ExactArgs(1)
	addInsightsRegionFlag(get)
	AddStringFlag(get, doctl.ArgInsightsTracingFrom, "", "", "Start of the trace lookup window")
	AddStringFlag(get, doctl.ArgInsightsTracingTo, "", "", "End of the trace lookup window")

	return cmd
}

// RunInsightsTracesSearch searches trace summaries.
func RunInsightsTracesSearch(c *CmdConfig) error {
	region, err := insightsRequiredString(c, doctl.ArgRegionSlug)
	if err != nil {
		return err
	}
	request, err := buildInsightsTracesSearchRequest(c)
	if err != nil {
		return err
	}
	response, err := c.Insights().SearchTraces(region, request)
	if err != nil {
		return err
	}
	if err := c.Display(&displayers.InsightsTraces{Response: response}); err != nil {
		return err
	}
	if Output == "text" && response != nil && response.Pagination != nil && response.Pagination.HasMore {
		fmt.Fprintln(insightsStderr, "More traces matched than were returned. Narrow the time range or filter to see additional results.")
	}
	return nil
}

// RunInsightsTraceGet retrieves a trace and its spans.
func RunInsightsTraceGet(c *CmdConfig) error {
	if len(c.Args) != 1 {
		return doctl.NewMissingArgsErr(c.NS)
	}
	traceID := strings.TrimSpace(c.Args[0])
	if traceID == "" {
		return fmt.Errorf("trace ID is required")
	}
	region, err := insightsRequiredString(c, doctl.ArgRegionSlug)
	if err != nil {
		return err
	}
	from, err := insightsTrimmedString(c, doctl.ArgInsightsTracingFrom)
	if err != nil {
		return err
	}
	to, err := insightsTrimmedString(c, doctl.ArgInsightsTracingTo)
	if err != nil {
		return err
	}
	if err := validateInsightsTraceGetWindow(from, to); err != nil {
		return err
	}

	var options *godo.TraceGetOptions
	if from != "" || to != "" {
		options = &godo.TraceGetOptions{From: from, To: to}
	}
	trace, err := c.Insights().GetTrace(region, traceID, options)
	if err != nil {
		return err
	}
	if err := c.Display(&displayers.InsightsTrace{Trace: trace}); err != nil {
		return err
	}
	if Output == "text" && trace != nil && trace.Truncated {
		fmt.Fprintln(insightsStderr, "The trace contains additional spans that were not returned. Narrow the lookup window to see more.")
	}
	return nil
}

func buildInsightsTracesSearchRequest(c *CmdConfig) (*godo.TracesSearchRequest, error) {
	options, err := getInsightsTracingSearchOptions(c)
	if err != nil {
		return nil, err
	}

	request := &godo.TracesSearchRequest{}
	if options.requestFile != "" {
		if err := rejectRequestFileFlagMix(c, options.requestFile, insightsTracingRequestFlags...); err != nil {
			return nil, err
		}
		request, err = readInsightsRequest[godo.TracesSearchRequest](insightsStdin, options.requestFile)
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
		request.TimeRange = godo.TracesTimeRange{
			From: tracesTimeBound(options.from),
			To:   tracesTimeBound(options.to),
		}

		if options.filter != "" && options.filterFile != "" {
			return nil, fmt.Errorf("--%s and --%s cannot be combined", doctl.ArgInsightsTracingFilter, doctl.ArgInsightsTracingFilterFile)
		}
		switch {
		case options.filter != "":
			request.Filter = &godo.TracesFilterExpression{}
			if err := decodeInsightsJSON(strings.NewReader(options.filter), request.Filter); err != nil {
				return nil, fmt.Errorf("parsing --%s: %w", doctl.ArgInsightsTracingFilter, err)
			}
		case options.filterFile != "":
			request.Filter, err = readInsightsRequest[godo.TracesFilterExpression](insightsStdin, options.filterFile)
			if err != nil {
				return nil, fmt.Errorf("--%s: %w", doctl.ArgInsightsTracingFilterFile, err)
			}
		}

		for _, clause := range options.orderBy {
			field, direction, err := parseInsightsTracingOrderBy(clause)
			if err != nil {
				return nil, err
			}
			request.OrderBy = append(request.OrderBy, godo.TracesOrderBy{
				Field:     godo.TracesFilterField{Name: field},
				Direction: direction,
			})
		}
	}

	if options.limit > 0 {
		if request.Pagination == nil {
			request.Pagination = &godo.TracesPaginationRequest{}
		}
		request.Pagination.Limit = options.limit
	}
	if err := validateTracesSearchRequest(request); err != nil {
		return nil, err
	}
	return request, nil
}

func tracesTimeBound(value string) godo.TracesTimeBound {
	if insightsRelativeTimeRE.MatchString(value) {
		return godo.TracesTimeBound{Relative: value}
	}
	return godo.TracesTimeBound{Absolute: value}
}

func validateTracesSearchRequest(request *godo.TracesSearchRequest) error {
	if request == nil {
		return fmt.Errorf("traces search request is required")
	}
	from, err := insightsSearchTimeValue(request.TimeRange.From.Relative, request.TimeRange.From.Absolute)
	if err != nil {
		return fmt.Errorf("invalid traces time_range.from: %w", err)
	}
	to, err := insightsSearchTimeValue(request.TimeRange.To.Relative, request.TimeRange.To.Absolute)
	if err != nil {
		return fmt.Errorf("invalid traces time_range.to: %w", err)
	}
	if err := validateInsightsTracingTimeRange(from, to); err != nil {
		return err
	}
	if request.Filter != nil {
		if err := validateTracesFilterExpression(request.Filter); err != nil {
			return fmt.Errorf("invalid traces filter: %w", err)
		}
	}
	for _, order := range request.OrderBy {
		if strings.TrimSpace(order.Field.Name) == "" {
			return fmt.Errorf("invalid traces order_by: field name is required")
		}
		if !validInsightsFieldScope(order.Field.Scope) {
			return fmt.Errorf("invalid traces order_by field scope %q", order.Field.Scope)
		}
		if order.Direction != "" &&
			order.Direction != godo.InsightsTracesSortDirectionAscending &&
			order.Direction != godo.InsightsTracesSortDirectionDescending {
			return fmt.Errorf("invalid traces order_by direction %q", order.Direction)
		}
	}
	if request.Pagination != nil {
		if err := validateInsightsTracingLimit(request.Pagination.Limit); err != nil {
			return err
		}
	}
	return nil
}

func validateTracesFilterExpression(expression *godo.TracesFilterExpression) error {
	if expression == nil {
		return fmt.Errorf("expression is required")
	}
	encoded, err := json.Marshal(expression)
	if err != nil {
		return err
	}
	var canonical godo.TracesFilterExpression
	if err := json.Unmarshal(encoded, &canonical); err != nil {
		return err
	}
	return validateCanonicalTracesFilterExpression(&canonical)
}

func validateCanonicalTracesFilterExpression(expression *godo.TracesFilterExpression) error {
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
			godo.InsightsTracesFilterOperatorEqual,
			godo.InsightsTracesFilterOperatorNotEqual,
			godo.InsightsTracesFilterOperatorGreaterThanOrEqual,
			godo.InsightsTracesFilterOperatorLessThanOrEqual,
			godo.InsightsTracesFilterOperatorIn,
			godo.InsightsTracesFilterOperatorExists,
		}, condition.Operator) {
			return fmt.Errorf("invalid condition operator %q", condition.Operator)
		}
		if condition.Operator != godo.InsightsTracesFilterOperatorExists {
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
			if err := validateCanonicalTracesFilterExpression(&expression.And.Expressions[i]); err != nil {
				return err
			}
		}
	case expression.Or != nil:
		if len(expression.Or.Expressions) == 0 {
			return fmt.Errorf("or expression must contain child expressions")
		}
		for i := range expression.Or.Expressions {
			if err := validateCanonicalTracesFilterExpression(&expression.Or.Expressions[i]); err != nil {
				return err
			}
		}
	case expression.Not != nil:
		return validateCanonicalTracesFilterExpression(expression.Not)
	case expression.TextSearch != nil:
		if strings.TrimSpace(expression.TextSearch.Query) == "" {
			return fmt.Errorf("text_search query is required")
		}
	default:
		return fmt.Errorf("expression must contain exactly one node")
	}
	return nil
}

func validateInsightsTraceGetWindow(from, to string) error {
	if from == "" && to == "" {
		return nil
	}
	now := time.Now().UTC()
	if from != "" {
		if _, err := parseInsightsTracingTime(from, now); err != nil {
			return fmt.Errorf("invalid --from %q: %w", from, err)
		}
	}
	if to != "" {
		if _, err := parseInsightsTracingTime(to, now); err != nil {
			return fmt.Errorf("invalid --to %q: %w", to, err)
		}
	}
	if from != "" && to != "" {
		return validateInsightsTracingTimeRange(from, to)
	}
	return nil
}

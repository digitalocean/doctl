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
	"strings"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/commands/displayers"
	"github.com/digitalocean/godo"
	"github.com/spf13/cobra"
)

const (
	insightsLogsDefaultFrom = "1h"
	insightsLogsDefaultTo   = "now"
	insightsLogsMaxLimit    = 1000
)

var insightsLogsRequestFlags = []string{
	doctl.ArgInsightsLogsFrom,
	doctl.ArgInsightsLogsTo,
	doctl.ArgInsightsLogsText,
	doctl.ArgInsightsLogsFilter,
	doctl.ArgInsightsLogsFilterFile,
	doctl.ArgInsightsLogsOrderBy,
}

func insightsLogs() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:     "logs",
			Aliases: []string{"log"},
			Short:   "Display commands for searching Insights logs",
			Long:    `The commands under ` + "`" + `doctl insights logs` + "`" + ` search log records stored in Insights.`,
		},
	}

	cmdSearch := CmdBuilder(cmd, RunInsightsLogsSearch, "search", "Search log records",
		`Searches log records in a region.

Build the search with flags, or pass a complete request body with `+"`"+`--request-file`+"`"+`. The request file uses the JSON body of the public logs search API, as JSON or YAML. When using a request file, `+"`"+`--limit`+"`"+` and `+"`"+`--cursor`+"`"+` override the file's pagination so you can page through results.

`+"`"+`--from`+"`"+` and `+"`"+`--to`+"`"+` accept an RFC3339 timestamp, a Unix timestamp, `+"`"+`now`+"`"+`, a look-back duration such as `+"`"+`15m`+"`"+` or `+"`"+`7d`+"`"+`, or an offset such as `+"`"+`now-1h`+"`"+`.

`+"`"+`--filter`+"`"+` and `+"`"+`--filter-file`+"`"+` take a JSON filter expression, for example `+"`"+`{"condition":{"field":{"name":"service.name"},"operator":"FILTER_OPERATOR_EQ","value":{"string_value":"api"}}}`+"`"+`. When `+"`"+`--text`+"`"+` is also set, both must match.

Results are returned one page at a time. Cursor pagination is only available when ordering by `+"`"+`timestamp`+"`"+` alone; when more results are available, the next cursor is printed to stderr in text output and returned in `+"`"+`pagination.next_cursor`+"`"+` in JSON output.`,
		Writer, displayerType(&displayers.InsightsLogs{}))
	addInsightsRegionFlag(cmdSearch)
	AddStringFlag(cmdSearch, doctl.ArgInsightsLogsFrom, "", "", "Start of the time range. Defaults to "+insightsLogsDefaultFrom)
	AddStringFlag(cmdSearch, doctl.ArgInsightsLogsTo, "", "", "End of the time range. Defaults to "+insightsLogsDefaultTo)
	AddStringFlag(cmdSearch, doctl.ArgInsightsLogsText, "", "", "Search for a substring across the log body, service name, and resource URN")
	AddStringFlag(cmdSearch, doctl.ArgInsightsLogsFilter, "", "", "A JSON filter expression")
	AddStringFlag(cmdSearch, doctl.ArgInsightsLogsFilterFile, "", "", "Read a JSON or YAML filter expression from `path`, or - to read from stdin")
	AddStringFlag(cmdSearch, doctl.ArgInsightsRequestFile, "", "", "Read a complete JSON or YAML logs search request from `path`, or - to read from stdin")
	AddStringSliceFlag(cmdSearch, doctl.ArgInsightsLogsOrderBy, "", nil, "Sort clause in the form `field[:asc|desc]`, for example timestamp:desc. Repeat the flag to sort by several fields")
	AddIntFlag(cmdSearch, doctl.ArgInsightsLogsLimit, "", 0, fmt.Sprintf("Maximum number of records to return, up to %d. The server defaults to 100", insightsLogsMaxLimit))
	AddStringFlag(cmdSearch, doctl.ArgInsightsLogsCursor, "", "", "Cursor from a previous search to retrieve the next page")
	cmdSearch.Example = `The following example searches the last 15 minutes of logs in ` + "`" + `nyc3` + "`" + ` for "timeout", newest first: doctl insights logs search --region nyc3 --from 15m --text timeout --order-by timestamp:desc`

	return cmd
}

// RunInsightsLogsSearch searches log records.
func RunInsightsLogsSearch(c *CmdConfig) error {
	region, err := insightsRequiredString(c, doctl.ArgRegionSlug)
	if err != nil {
		return err
	}

	req, err := buildInsightsLogsSearchRequest(c)
	if err != nil {
		return err
	}

	resp, err := c.Insights().SearchLogs(region, req)
	if err != nil {
		return err
	}

	if err := c.Display(&displayers.InsightsLogs{Response: resp}); err != nil {
		return err
	}

	if Output == "text" && resp != nil && resp.Pagination != nil && resp.Pagination.HasMore && resp.Pagination.NextCursor != "" {
		fmt.Fprintf(insightsStderr, "More results are available. To retrieve the next page, use --%s %s\n",
			doctl.ArgInsightsLogsCursor, resp.Pagination.NextCursor)
	}
	return nil
}

func buildInsightsLogsSearchRequest(c *CmdConfig) (*godo.LogsSearchRequest, error) {
	requestFile, err := insightsTrimmedString(c, doctl.ArgInsightsRequestFile)
	if err != nil {
		return nil, err
	}
	from, err := insightsTrimmedString(c, doctl.ArgInsightsLogsFrom)
	if err != nil {
		return nil, err
	}
	to, err := insightsTrimmedString(c, doctl.ArgInsightsLogsTo)
	if err != nil {
		return nil, err
	}
	text, err := insightsTrimmedString(c, doctl.ArgInsightsLogsText)
	if err != nil {
		return nil, err
	}
	filter, err := insightsTrimmedString(c, doctl.ArgInsightsLogsFilter)
	if err != nil {
		return nil, err
	}
	filterFile, err := insightsTrimmedString(c, doctl.ArgInsightsLogsFilterFile)
	if err != nil {
		return nil, err
	}
	orderBy, err := c.Doit.GetStringSlice(c.NS, doctl.ArgInsightsLogsOrderBy)
	if err != nil {
		return nil, err
	}
	limit, err := c.Doit.GetInt(c.NS, doctl.ArgInsightsLogsLimit)
	if err != nil {
		return nil, err
	}
	cursor, err := insightsTrimmedString(c, doctl.ArgInsightsLogsCursor)
	if err != nil {
		return nil, err
	}

	if limit < 0 || limit > insightsLogsMaxLimit {
		return nil, fmt.Errorf("--%s must be between 1 and %d", doctl.ArgInsightsLogsLimit, insightsLogsMaxLimit)
	}

	req := &godo.LogsSearchRequest{}
	if requestFile != "" {
		if err := rejectRequestFileFlagMix(c, requestFile, insightsLogsRequestFlags...); err != nil {
			return nil, err
		}

		req, err = readInsightsRequest[godo.LogsSearchRequest](insightsStdin, requestFile)
		if err != nil {
			return nil, err
		}
		if req.TimeRange.From == (godo.LogsTimeInstant{}) || req.TimeRange.To == (godo.LogsTimeInstant{}) {
			return nil, fmt.Errorf("--%s must set time_range.from and time_range.to", doctl.ArgInsightsRequestFile)
		}
	} else {
		if from == "" {
			from = insightsLogsDefaultFrom
		}
		if to == "" {
			to = insightsLogsDefaultTo
		}
		req.TimeRange = godo.LogsTimeRange{
			From: parseInsightsLogsTime(from),
			To:   parseInsightsLogsTime(to),
		}

		if filter != "" && filterFile != "" {
			return nil, fmt.Errorf("--%s and --%s cannot be combined", doctl.ArgInsightsLogsFilter, doctl.ArgInsightsLogsFilterFile)
		}
		var expr *godo.LogsFilterExpression
		switch {
		case filter != "":
			expr = &godo.LogsFilterExpression{}
			if err := decodeInsightsJSON(strings.NewReader(filter), expr); err != nil {
				return nil, fmt.Errorf("parsing --%s: %w", doctl.ArgInsightsLogsFilter, err)
			}
		case filterFile != "":
			expr, err = readInsightsRequest[godo.LogsFilterExpression](insightsStdin, filterFile)
			if err != nil {
				return nil, fmt.Errorf("--%s: %w", doctl.ArgInsightsLogsFilterFile, err)
			}
		}

		if text != "" {
			textExpr := godo.LogsFilterExpression{TextSearch: &godo.LogsTextSearch{Query: text}}
			if expr == nil {
				expr = &textExpr
			} else {
				expr = &godo.LogsFilterExpression{And: &godo.LogsFilterGroup{
					Expressions: []godo.LogsFilterExpression{*expr, textExpr},
				}}
			}
		}
		req.Filter = expr

		for _, clause := range orderBy {
			ob, err := parseInsightsLogsOrderBy(clause)
			if err != nil {
				return nil, err
			}
			req.OrderBy = append(req.OrderBy, ob)
		}
	}

	if limit > 0 || cursor != "" {
		if req.Pagination == nil {
			req.Pagination = &godo.LogsPaginationRequest{}
		}
		if limit > 0 {
			req.Pagination.Limit = limit
		}
		if cursor != "" {
			req.Pagination.Cursor = cursor
		}
	}

	if req.Pagination != nil && req.Pagination.Cursor != "" && !insightsLogsOrderedByTimestampOnly(req.OrderBy) {
		return nil, fmt.Errorf("--%s requires results to be ordered by timestamp alone", doctl.ArgInsightsLogsCursor)
	}

	return req, nil
}

// parseInsightsLogsTime maps a CLI time value onto the logs API time
// instant. Anything that is not a relative expression is sent as an absolute
// time, which the API accepts as RFC3339 or Unix seconds/nanoseconds.
func parseInsightsLogsTime(s string) godo.LogsTimeInstant {
	if insightsRelativeTimeRE.MatchString(s) {
		return godo.LogsTimeInstant{Relative: s}
	}
	return godo.LogsTimeInstant{Absolute: s}
}

func parseInsightsLogsOrderBy(clause string) (godo.LogsOrderBy, error) {
	clause = strings.TrimSpace(clause)
	field, direction := clause, ""
	if i := strings.LastIndex(clause, ":"); i >= 0 {
		field, direction = strings.TrimSpace(clause[:i]), strings.ToLower(strings.TrimSpace(clause[i+1:]))
	}
	if field == "" {
		return godo.LogsOrderBy{}, fmt.Errorf("invalid --%s %q: a field name is required", doctl.ArgInsightsLogsOrderBy, clause)
	}

	ob := godo.LogsOrderBy{Field: godo.LogsFieldRef{Name: field}}
	switch direction {
	case "":
	case "asc":
		ob.Direction = "SORT_DIRECTION_ASC"
	case "desc":
		ob.Direction = "SORT_DIRECTION_DESC"
	default:
		return godo.LogsOrderBy{}, fmt.Errorf("invalid --%s %q: direction must be `asc` or `desc`", doctl.ArgInsightsLogsOrderBy, clause)
	}
	return ob, nil
}

func insightsLogsOrderedByTimestampOnly(orderBy []godo.LogsOrderBy) bool {
	return len(orderBy) == 0 || (len(orderBy) == 1 && orderBy[0].Field.Name == "timestamp")
}

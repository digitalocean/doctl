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
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/digitalocean/doctl"
)

const (
	insightsTracingDefaultFrom = "1h"
	insightsTracingDefaultTo   = "now"
	insightsTracingMaxLimit    = 1000
	insightsTracingMaxRange    = 7 * 24 * time.Hour
)

var (
	insightsTracingNowOffsetRE = regexp.MustCompile(`^now([+-])(\d+)([smhdw])$`)
	insightsTracingLookbackRE  = regexp.MustCompile(`^(\d+)([smhdw])$`)

	insightsTracingRequestFlags = []string{
		doctl.ArgInsightsTracingFrom,
		doctl.ArgInsightsTracingTo,
		doctl.ArgInsightsTracingFilter,
		doctl.ArgInsightsTracingFilterFile,
		doctl.ArgInsightsTracingOrderBy,
	}
)

func addInsightsTracingSearchFlags(cmd *Command) {
	AddStringFlag(cmd, doctl.ArgInsightsTracingFrom, "", "", "Start of the time range. Defaults to "+insightsTracingDefaultFrom)
	AddStringFlag(cmd, doctl.ArgInsightsTracingTo, "", "", "End of the time range. Defaults to "+insightsTracingDefaultTo)
	AddStringFlag(cmd, doctl.ArgInsightsTracingFilter, "", "", "A JSON logs-style filter expression using condition, and, or, not, or text_search")
	AddStringFlag(cmd, doctl.ArgInsightsTracingFilterFile, "", "", "Read a JSON or YAML filter expression from `path`, or - to read from stdin")
	AddStringArrayFlag(cmd, doctl.ArgInsightsTracingOrderBy, "", nil, "Sort clause in the form `field[:asc|desc]`; repeat to sort by several fields")
	AddIntFlag(cmd, doctl.ArgInsightsTracingLimit, "", 0, fmt.Sprintf("Maximum number of results to return, from 1 to %d", insightsTracingMaxLimit))
	AddStringFlag(cmd, doctl.ArgInsightsRequestFile, "", "", "Read a complete JSON or YAML search request from `path`, or - to read from stdin")
}

type insightsTracingSearchOptions struct {
	requestFile string
	from        string
	to          string
	filter      string
	filterFile  string
	orderBy     []string
	limit       int
}

func getInsightsTracingSearchOptions(c *CmdConfig) (*insightsTracingSearchOptions, error) {
	options := &insightsTracingSearchOptions{}
	var err error
	if options.requestFile, err = insightsTrimmedString(c, doctl.ArgInsightsRequestFile); err != nil {
		return nil, err
	}
	if options.from, err = insightsTrimmedString(c, doctl.ArgInsightsTracingFrom); err != nil {
		return nil, err
	}
	if options.to, err = insightsTrimmedString(c, doctl.ArgInsightsTracingTo); err != nil {
		return nil, err
	}
	if options.filter, err = insightsTrimmedString(c, doctl.ArgInsightsTracingFilter); err != nil {
		return nil, err
	}
	if options.filterFile, err = insightsTrimmedString(c, doctl.ArgInsightsTracingFilterFile); err != nil {
		return nil, err
	}
	if options.orderBy, err = getInsightsStringArray(c, doctl.ArgInsightsTracingOrderBy); err != nil {
		return nil, err
	}
	if options.limit, err = c.Doit.GetInt(c.NS, doctl.ArgInsightsTracingLimit); err != nil {
		return nil, err
	}
	if options.limit != 0 || c.Doit.IsSet(doctl.ArgInsightsTracingLimit) {
		if err := validateInsightsTracingLimit(options.limit); err != nil {
			return nil, err
		}
	}
	return options, nil
}

func validateInsightsTracingTimeRange(from, to string) error {
	now := time.Now().UTC()
	start, err := parseInsightsTracingTime(from, now)
	if err != nil {
		return fmt.Errorf("invalid --from %q: %w", from, err)
	}
	end, err := parseInsightsTracingTime(to, now)
	if err != nil {
		return fmt.Errorf("invalid --to %q: %w", to, err)
	}
	if start.After(end) {
		return fmt.Errorf("--from must not be after --to")
	}
	if end.Sub(start) > insightsTracingMaxRange {
		return fmt.Errorf("time range must not exceed 7 days")
	}
	return nil
}

func parseInsightsTracingTime(value string, now time.Time) (time.Time, error) {
	value = strings.TrimSpace(value)
	switch {
	case value == "now":
		return now, nil
	case insightsTracingLookbackRE.MatchString(value):
		match := insightsTracingLookbackRE.FindStringSubmatch(value)
		duration, err := insightsTracingDuration(match[1], match[2])
		if err != nil {
			return time.Time{}, err
		}
		return now.Add(-duration), nil
	case insightsTracingNowOffsetRE.MatchString(value):
		match := insightsTracingNowOffsetRE.FindStringSubmatch(value)
		duration, err := insightsTracingDuration(match[2], match[3])
		if err != nil {
			return time.Time{}, err
		}
		if match[1] == "-" {
			duration = -duration
		}
		return now.Add(duration), nil
	}

	if parsed, err := time.Parse(time.RFC3339Nano, value); err == nil {
		return parsed, nil
	}

	unixValue, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return time.Time{}, fmt.Errorf("must be relative time, RFC3339, or a Unix timestamp")
	}
	digits := len(strings.TrimLeft(value, "-"))
	switch {
	case digits <= 10:
		return time.Unix(unixValue, 0), nil
	case digits <= 13:
		return time.UnixMilli(unixValue), nil
	case digits <= 16:
		return time.UnixMicro(unixValue), nil
	default:
		return time.Unix(0, unixValue), nil
	}
}

func insightsTracingDuration(amount, unit string) (time.Duration, error) {
	value, err := strconv.ParseInt(amount, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid relative duration")
	}
	var multiplier time.Duration
	switch unit {
	case "s":
		multiplier = time.Second
	case "m":
		multiplier = time.Minute
	case "h":
		multiplier = time.Hour
	case "d":
		multiplier = 24 * time.Hour
	case "w":
		multiplier = 7 * 24 * time.Hour
	default:
		return 0, fmt.Errorf("invalid relative duration unit")
	}
	if value > int64((time.Duration(1<<63-1))/multiplier) {
		return 0, fmt.Errorf("relative duration is too large")
	}
	return time.Duration(value) * multiplier, nil
}

func parseInsightsTracingOrderBy(value string) (field, direction string, err error) {
	value = strings.TrimSpace(value)
	field = value
	if i := strings.LastIndex(value, ":"); i >= 0 {
		field = strings.TrimSpace(value[:i])
		direction = strings.ToLower(strings.TrimSpace(value[i+1:]))
	}
	if field == "" {
		return "", "", fmt.Errorf("invalid --order-by %q: a field name is required", value)
	}
	switch direction {
	case "", "asc", "desc":
		return field, direction, nil
	default:
		return "", "", fmt.Errorf("invalid --order-by %q: direction must be `asc` or `desc`", value)
	}
}

func validateInsightsTracingLimit(limit int) error {
	if limit < 1 || limit > insightsTracingMaxLimit {
		return fmt.Errorf("--limit must be between 1 and %d", insightsTracingMaxLimit)
	}
	return nil
}

func insightsSearchTimeValue(relative, absolute string) (string, error) {
	if relative != "" && absolute != "" {
		return "", fmt.Errorf("time bound must set exactly one of relative or absolute")
	}
	if relative != "" {
		return relative, nil
	}
	if absolute != "" {
		return absolute, nil
	}
	return "", fmt.Errorf("time bound is required")
}

func validInsightsFieldScope(scope string) bool {
	return scope == "" ||
		scope == "FIELD_SCOPE_RESOURCE" ||
		scope == "FIELD_SCOPE_ATTRIBUTES"
}

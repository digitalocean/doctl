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
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/commands/displayers"
	"github.com/digitalocean/godo"
	"github.com/spf13/cobra"
)

const (
	insightsTimeHelp     = "RFC3339 timestamp or Unix timestamp in seconds, for example 2026-09-01T00:00:00Z or 1756684800"
	insightsDurationHelp = "Prometheus duration or number of seconds, for example 30s, 5m, or 1h30m"
)

func insightsMetrics() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:     "metrics",
			Aliases: []string{"metric", "promql"},
			Short:   "Display commands for querying Insights metrics with PromQL",
			Long: `The commands under ` + "`" + `doctl insights metrics` + "`" + ` query Insights metrics in a region using the Prometheus-compatible HTTP API.

Text output shows one row per sample or series. JSON output (` + "`" + `--output json` + "`" + `) preserves the Prometheus response envelope, so it can be piped to tools that expect Prometheus API responses.`,
		},
	}

	cmdQuery := CmdBuilder(cmd, RunInsightsMetricsQuery, "query", "Evaluate an instant PromQL query",
		`Evaluates a PromQL expression at a single point in time. If `+"`"+`--time`+"`"+` is omitted, the server evaluates the query at the current time.`,
		Writer, displayerType(&displayers.InsightsPromQuery{}))
	addInsightsRegionFlag(cmdQuery)
	AddStringFlag(cmdQuery, doctl.ArgInsightsQuery, "", "", "The PromQL expression to evaluate", requiredOpt())
	AddStringFlag(cmdQuery, doctl.ArgInsightsTime, "", "", "Evaluation time. "+insightsTimeHelp)
	AddStringFlag(cmdQuery, doctl.ArgTimeout, "", "", "Evaluation timeout. "+insightsDurationHelp)
	cmdQuery.Example = `The following example returns the current value of the ` + "`" + `up` + "`" + ` metric in ` + "`" + `nyc3` + "`" + `: doctl insights metrics query --region nyc3 --query 'up'`

	cmdQueryRange := CmdBuilder(cmd, RunInsightsMetricsQueryRange, "query-range", "Evaluate a PromQL query over a time range",
		`Evaluates a PromQL expression over a time range at a fixed resolution step.`,
		Writer, displayerType(&displayers.InsightsPromQueryRange{}))
	addInsightsRegionFlag(cmdQueryRange)
	AddStringFlag(cmdQueryRange, doctl.ArgInsightsQuery, "", "", "The PromQL expression to evaluate", requiredOpt())
	AddStringFlag(cmdQueryRange, doctl.ArgInsightsStart, "", "", "Start of the range. "+insightsTimeHelp, requiredOpt())
	AddStringFlag(cmdQueryRange, doctl.ArgInsightsEnd, "", "", "End of the range. "+insightsTimeHelp, requiredOpt())
	AddStringFlag(cmdQueryRange, doctl.ArgInsightsStep, "", "", "Resolution step between samples. "+insightsDurationHelp, requiredOpt())
	AddStringFlag(cmdQueryRange, doctl.ArgTimeout, "", "", "Evaluation timeout. "+insightsDurationHelp)
	cmdQueryRange.Example = `The following example returns the per-second request rate over one hour at one-minute resolution: doctl insights metrics query-range --region nyc3 --query 'rate(http_requests_total[5m])' --start 2026-09-01T00:00:00Z --end 2026-09-01T01:00:00Z --step 1m`

	cmdLabels := CmdBuilder(cmd, RunInsightsMetricsLabels, "labels", "List label names",
		`Lists label names, optionally restricted to series that match one or more selectors and to a time range.`,
		Writer, displayerType(&displayers.InsightsPromLabels{}))
	addInsightsRegionFlag(cmdLabels)
	addInsightsSelectorFlags(cmdLabels, false)
	cmdLabels.Example = `The following example lists the label names on ` + "`" + `up` + "`" + ` series: doctl insights metrics labels --region nyc3 --match 'up'`

	cmdLabelValues := CmdBuilder(cmd, RunInsightsMetricsLabelValues, "label-values <label-name>", "List the values of a label",
		`Lists the values of a label, optionally restricted to series that match one or more selectors and to a time range.`,
		Writer, displayerType(&displayers.InsightsPromLabelValues{}))
	addInsightsRegionFlag(cmdLabelValues)
	addInsightsSelectorFlags(cmdLabelValues, false)
	cmdLabelValues.Example = `The following example lists the values of the ` + "`" + `job` + "`" + ` label: doctl insights metrics label-values job --region nyc3`

	cmdSeries := CmdBuilder(cmd, RunInsightsMetricsSeries, "series", "List series that match selectors",
		`Lists the label sets of series that match one or more selectors. At least one `+"`"+`--match`+"`"+` selector is required.`,
		Writer, displayerType(&displayers.InsightsPromSeries{}))
	addInsightsRegionFlag(cmdSeries)
	addInsightsSelectorFlags(cmdSeries, true)
	cmdSeries.Example = `The following example lists series for two jobs: doctl insights metrics series --region nyc3 --match 'up{job="api"}' --match 'up{job="worker"}'`

	return cmd
}

func addInsightsRegionFlag(cmd *Command) {
	AddStringFlag(cmd, doctl.ArgRegionSlug, "", "", "The region slug to query, for example nyc3", requiredOpt())
}

func addInsightsSelectorFlags(cmd *Command, matchRequired bool) {
	desc := "A series `selector`, for example up{job=\"api\"}. Repeat the flag to pass several selectors"
	if matchRequired {
		desc += " (required)"
	}
	cmd.Flags().StringArray(doctl.ArgInsightsMatch, nil, desc)
	AddStringFlag(cmd, doctl.ArgInsightsStart, "", "", "Only consider series with samples after this time. "+insightsTimeHelp)
	AddStringFlag(cmd, doctl.ArgInsightsEnd, "", "", "Only consider series with samples before this time. "+insightsTimeHelp)
}

// RunInsightsMetricsQuery evaluates an instant PromQL query.
func RunInsightsMetricsQuery(c *CmdConfig) error {
	region, err := insightsRequiredString(c, doctl.ArgRegionSlug)
	if err != nil {
		return err
	}
	query, err := insightsRequiredString(c, doctl.ArgInsightsQuery)
	if err != nil {
		return err
	}
	evalTime, err := insightsOptionalTime(c, doctl.ArgInsightsTime)
	if err != nil {
		return err
	}
	timeout, err := insightsOptionalDuration(c, doctl.ArgTimeout)
	if err != nil {
		return err
	}

	resp, err := c.Insights().PostQuery(region, &godo.PromQueryOptions{
		Query:   query,
		Time:    evalTime,
		Timeout: timeout,
	})
	if err != nil {
		return err
	}

	return c.Display(&displayers.InsightsPromQuery{Response: resp})
}

// RunInsightsMetricsQueryRange evaluates a PromQL query over a time range.
func RunInsightsMetricsQueryRange(c *CmdConfig) error {
	region, err := insightsRequiredString(c, doctl.ArgRegionSlug)
	if err != nil {
		return err
	}
	query, err := insightsRequiredString(c, doctl.ArgInsightsQuery)
	if err != nil {
		return err
	}
	start, end, err := insightsTimeRange(c, true)
	if err != nil {
		return err
	}
	step, err := insightsRequiredString(c, doctl.ArgInsightsStep)
	if err != nil {
		return err
	}
	stepDuration, err := parsePromDuration(step)
	if err != nil {
		return fmt.Errorf("invalid --%s %q: %w", doctl.ArgInsightsStep, step, err)
	}
	if stepDuration <= 0 {
		return fmt.Errorf("invalid --%s %q: must be greater than zero", doctl.ArgInsightsStep, step)
	}
	timeout, err := insightsOptionalDuration(c, doctl.ArgTimeout)
	if err != nil {
		return err
	}

	resp, err := c.Insights().PostQueryRange(region, &godo.PromQueryRangeOptions{
		Query:   query,
		Start:   start,
		End:     end,
		Step:    step,
		Timeout: timeout,
	})
	if err != nil {
		return err
	}

	return c.Display(&displayers.InsightsPromQueryRange{Response: resp})
}

// RunInsightsMetricsLabels lists label names.
func RunInsightsMetricsLabels(c *CmdConfig) error {
	region, err := insightsRequiredString(c, doctl.ArgRegionSlug)
	if err != nil {
		return err
	}
	opt, err := insightsSelectorOptions(c, false)
	if err != nil {
		return err
	}

	resp, err := c.Insights().PostLabels(region, opt)
	if err != nil {
		return err
	}

	return c.Display(&displayers.InsightsPromLabels{Response: resp})
}

// RunInsightsMetricsLabelValues lists the values of a label.
func RunInsightsMetricsLabelValues(c *CmdConfig) error {
	if err := ensureOneArg(c); err != nil {
		return err
	}
	name := strings.TrimSpace(c.Args[0])
	if name == "" {
		return fmt.Errorf("a label name is required")
	}
	region, err := insightsRequiredString(c, doctl.ArgRegionSlug)
	if err != nil {
		return err
	}
	opt, err := insightsSelectorOptions(c, false)
	if err != nil {
		return err
	}

	resp, err := c.Insights().LabelValues(region, name, opt)
	if err != nil {
		return err
	}

	return c.Display(&displayers.InsightsPromLabelValues{Response: resp})
}

// RunInsightsMetricsSeries lists series that match selectors.
func RunInsightsMetricsSeries(c *CmdConfig) error {
	region, err := insightsRequiredString(c, doctl.ArgRegionSlug)
	if err != nil {
		return err
	}
	opt, err := insightsSelectorOptions(c, true)
	if err != nil {
		return err
	}

	resp, err := c.Insights().PostSeries(region, opt)
	if err != nil {
		return err
	}

	return c.Display(&displayers.InsightsPromSeries{Response: resp})
}

func insightsRequiredString(c *CmdConfig, name string) (string, error) {
	val, err := c.Doit.GetString(c.NS, name)
	if err != nil {
		return "", err
	}
	val = strings.TrimSpace(val)
	if val == "" {
		return "", fmt.Errorf("--%s is required", name)
	}
	return val, nil
}

func insightsOptionalTime(c *CmdConfig, name string) (string, error) {
	val, err := c.Doit.GetString(c.NS, name)
	if err != nil {
		return "", err
	}
	val = strings.TrimSpace(val)
	if val == "" {
		return "", nil
	}
	if _, err := parsePromTime(val); err != nil {
		return "", fmt.Errorf("invalid --%s %q: %w", name, val, err)
	}
	return val, nil
}

func insightsOptionalDuration(c *CmdConfig, name string) (string, error) {
	val, err := c.Doit.GetString(c.NS, name)
	if err != nil {
		return "", err
	}
	val = strings.TrimSpace(val)
	if val == "" {
		return "", nil
	}
	if _, err := parsePromDuration(val); err != nil {
		return "", fmt.Errorf("invalid --%s %q: %w", name, val, err)
	}
	return val, nil
}

// insightsTimeRange reads --start and --end, requiring both when required is
// set, and rejects ranges where start is after end.
func insightsTimeRange(c *CmdConfig, required bool) (string, string, error) {
	read := insightsOptionalTime
	if required {
		read = func(c *CmdConfig, name string) (string, error) {
			if _, err := insightsRequiredString(c, name); err != nil {
				return "", err
			}
			return insightsOptionalTime(c, name)
		}
	}

	start, err := read(c, doctl.ArgInsightsStart)
	if err != nil {
		return "", "", err
	}
	end, err := read(c, doctl.ArgInsightsEnd)
	if err != nil {
		return "", "", err
	}

	if start != "" && end != "" {
		s, _ := parsePromTime(start)
		e, _ := parsePromTime(end)
		if s.After(e) {
			return "", "", fmt.Errorf("--%s must not be after --%s", doctl.ArgInsightsStart, doctl.ArgInsightsEnd)
		}
	}

	return start, end, nil
}

func insightsSelectorOptions(c *CmdConfig, matchRequired bool) (*godo.PromSelectorOptions, error) {
	matchers, err := insightsMatchers(c)
	if err != nil {
		return nil, err
	}
	if matchRequired && len(matchers) == 0 {
		return nil, fmt.Errorf("at least one --%s selector is required", doctl.ArgInsightsMatch)
	}
	start, end, err := insightsTimeRange(c, false)
	if err != nil {
		return nil, err
	}

	return &godo.PromSelectorOptions{Match: matchers, Start: start, End: end}, nil
}

// insightsMatchers reads the repeatable --match flag. Selectors routinely
// contain commas and quotes, so the value is read from the cobra flag set
// rather than through viper, which would split it as CSV.
func insightsMatchers(c *CmdConfig) ([]string, error) {
	var raw []string
	if c.Command != nil && c.Command.Flags().Lookup(doctl.ArgInsightsMatch) != nil {
		vals, err := c.Command.Flags().GetStringArray(doctl.ArgInsightsMatch)
		if err != nil {
			return nil, err
		}
		raw = vals
	} else {
		vals, err := c.Doit.GetStringSlice(c.NS, doctl.ArgInsightsMatch)
		if err != nil {
			return nil, err
		}
		raw = vals
	}

	matchers := make([]string, 0, len(raw))
	for _, m := range raw {
		if m = strings.TrimSpace(m); m != "" {
			matchers = append(matchers, m)
		}
	}
	return matchers, nil
}

// parsePromTime parses the timestamp formats accepted by the Prometheus HTTP
// API: RFC3339 or a Unix timestamp in (possibly fractional) seconds.
func parsePromTime(s string) (time.Time, error) {
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		if math.IsNaN(f) || math.IsInf(f, 0) {
			return time.Time{}, fmt.Errorf("must be a finite Unix timestamp")
		}
		sec, frac := math.Modf(f)
		return time.Unix(int64(sec), int64(frac*1e9)).UTC(), nil
	}
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		return time.Time{}, fmt.Errorf("must be an RFC3339 timestamp or a Unix timestamp in seconds")
	}
	return t, nil
}

var promDurationRE = regexp.MustCompile(`^(?:(\d+)y)?(?:(\d+)w)?(?:(\d+)d)?(?:(\d+)h)?(?:(\d+)m)?(?:(\d+)s)?(?:(\d+)ms)?$`)

// parsePromDuration parses the duration formats accepted by the Prometheus
// HTTP API: a duration such as `1h30m` or a number of seconds.
func parsePromDuration(s string) (time.Duration, error) {
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		if math.IsNaN(f) || math.IsInf(f, 0) || f < 0 {
			return 0, fmt.Errorf("must be a non-negative number of seconds")
		}
		return time.Duration(f * float64(time.Second)), nil
	}

	m := promDurationRE.FindStringSubmatch(s)
	if s == "" || m == nil {
		return 0, fmt.Errorf("must be a duration such as `30s`, `5m`, or `1h30m`, or a number of seconds")
	}

	units := []time.Duration{
		365 * 24 * time.Hour,
		7 * 24 * time.Hour,
		24 * time.Hour,
		time.Hour,
		time.Minute,
		time.Second,
		time.Millisecond,
	}
	var d time.Duration
	for i, unit := range units {
		if m[i+1] == "" {
			continue
		}
		n, err := strconv.ParseInt(m[i+1], 10, 64)
		if err != nil {
			return 0, err
		}
		d += time.Duration(n) * unit
	}
	return d, nil
}

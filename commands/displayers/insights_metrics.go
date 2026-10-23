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

package displayers

import (
	"encoding/json"
	"io"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/digitalocean/godo"
)

// promSample marshals a sample as the Prometheus [timestamp, "value"] pair.
// godo decodes this pair into a struct but does not encode it back.
type promSample godo.PromSample

func (s promSample) MarshalJSON() ([]byte, error) {
	return json.Marshal([]any{json.Number(strconv.FormatFloat(s.Timestamp, 'f', -1, 64)), s.Value})
}

type promVectorSample struct {
	Metric map[string]string `json:"metric"`
	Value  promSample        `json:"value"`
}

type promMatrixSample struct {
	Metric map[string]string `json:"metric"`
	Values []promSample      `json:"values"`
}

type promEnvelope struct {
	Status string   `json:"status"`
	Data   promData `json:"data"`
}

type promData struct {
	ResultType string `json:"resultType"`
	Result     any    `json:"result"`
}

func toPromVector(in []godo.PromVectorSample) []promVectorSample {
	out := make([]promVectorSample, 0, len(in))
	for _, s := range in {
		out = append(out, promVectorSample{Metric: nonNilLabels(s.Metric), Value: promSample(s.Value)})
	}
	return out
}

func toPromMatrix(in []godo.PromMatrixSample) []promMatrixSample {
	out := make([]promMatrixSample, 0, len(in))
	for _, s := range in {
		values := make([]promSample, 0, len(s.Values))
		for _, v := range s.Values {
			values = append(values, promSample(v))
		}
		out = append(out, promMatrixSample{Metric: nonNilLabels(s.Metric), Values: values})
	}
	return out
}

func nonNilLabels(m map[string]string) map[string]string {
	if m == nil {
		return map[string]string{}
	}
	return m
}

// formatPromMetric renders a label set in PromQL selector notation, for
// example `up{instance="a", job="api"}`.
func formatPromMetric(labels map[string]string) string {
	keys := make([]string, 0, len(labels))
	for k := range labels {
		if k != "__name__" {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)

	pairs := make([]string, 0, len(keys))
	for _, k := range keys {
		pairs = append(pairs, k+"="+strconv.Quote(labels[k]))
	}
	return labels["__name__"] + "{" + strings.Join(pairs, ", ") + "}"
}

func formatPromTimestamp(ts float64) string {
	sec, frac := math.Modf(ts)
	return time.Unix(int64(sec), int64(math.Round(frac*1e3))*int64(time.Millisecond)).UTC().Format(time.RFC3339Nano)
}

var promSampleCols = []string{"Metric", "Timestamp", "Value"}

var promSampleColMap = map[string]string{
	"Metric":    "Metric",
	"Timestamp": "Timestamp",
	"Value":     "Value",
}

func promSampleRow(metric map[string]string, s godo.PromSample) map[string]any {
	return map[string]any{
		"Metric":    formatPromMetric(metric),
		"Timestamp": formatPromTimestamp(s.Timestamp),
		"Value":     s.Value,
	}
}

func promMatrixRows(series []godo.PromMatrixSample) []map[string]any {
	out := []map[string]any{}
	for _, s := range series {
		for _, v := range s.Values {
			out = append(out, promSampleRow(s.Metric, v))
		}
	}
	return out
}

// InsightsPromQuery displays the result of an instant PromQL query.
type InsightsPromQuery struct {
	Response *godo.PromQueryResponse
}

var _ Displayable = &InsightsPromQuery{}

func (p *InsightsPromQuery) JSON(out io.Writer) error {
	r := p.Response
	if r == nil {
		r = &godo.PromQueryResponse{}
	}

	var result any
	switch r.ResultType {
	case "matrix":
		result = toPromMatrix(r.Matrix)
	case "scalar", "string":
		if r.Sample != nil {
			result = promSample(*r.Sample)
		}
	default:
		result = toPromVector(r.Vector)
	}

	return writeJSON(promEnvelope{Status: r.Status, Data: promData{ResultType: r.ResultType, Result: result}}, out)
}

func (p *InsightsPromQuery) Cols() []string { return promSampleCols }

func (p *InsightsPromQuery) ColMap() map[string]string { return promSampleColMap }

func (p *InsightsPromQuery) KV() []map[string]any {
	out := []map[string]any{}
	r := p.Response
	if r == nil {
		return out
	}

	switch {
	case r.Sample != nil:
		out = append(out, promSampleRow(nil, *r.Sample))
	case r.Matrix != nil:
		out = promMatrixRows(r.Matrix)
	default:
		for _, s := range r.Vector {
			out = append(out, promSampleRow(s.Metric, s.Value))
		}
	}
	return out
}

// InsightsPromQueryRange displays the result of a PromQL range query.
type InsightsPromQueryRange struct {
	Response *godo.PromQueryRangeResponse
}

var _ Displayable = &InsightsPromQueryRange{}

func (p *InsightsPromQueryRange) JSON(out io.Writer) error {
	r := p.Response
	if r == nil {
		r = &godo.PromQueryRangeResponse{}
	}
	return writeJSON(promEnvelope{
		Status: r.Status,
		Data:   promData{ResultType: r.Data.ResultType, Result: toPromMatrix(r.Data.Result)},
	}, out)
}

func (p *InsightsPromQueryRange) Cols() []string { return promSampleCols }

func (p *InsightsPromQueryRange) ColMap() map[string]string { return promSampleColMap }

func (p *InsightsPromQueryRange) KV() []map[string]any {
	if p.Response == nil {
		return []map[string]any{}
	}
	return promMatrixRows(p.Response.Data.Result)
}

// InsightsPromSeries displays the series that match one or more selectors.
type InsightsPromSeries struct {
	Response *godo.PromSeriesResponse
}

var _ Displayable = &InsightsPromSeries{}

func (p *InsightsPromSeries) JSON(out io.Writer) error {
	r := godo.PromSeriesResponse{Data: []map[string]string{}}
	if p.Response != nil {
		r.Status = p.Response.Status
		if p.Response.Data != nil {
			r.Data = p.Response.Data
		}
	}
	return writeJSON(r, out)
}

func (p *InsightsPromSeries) Cols() []string { return []string{"Series"} }

func (p *InsightsPromSeries) ColMap() map[string]string { return map[string]string{"Series": "Series"} }

func (p *InsightsPromSeries) KV() []map[string]any {
	out := []map[string]any{}
	if p.Response == nil {
		return out
	}
	for _, labels := range p.Response.Data {
		out = append(out, map[string]any{"Series": formatPromMetric(labels)})
	}
	return out
}

// InsightsPromLabels displays label names.
type InsightsPromLabels struct {
	Response *godo.PromLabelsResponse
}

var _ Displayable = &InsightsPromLabels{}

func (p *InsightsPromLabels) JSON(out io.Writer) error { return writePromLabelsJSON(p.Response, out) }

func (p *InsightsPromLabels) Cols() []string { return []string{"Name"} }

func (p *InsightsPromLabels) ColMap() map[string]string { return map[string]string{"Name": "Name"} }

func (p *InsightsPromLabels) KV() []map[string]any { return promLabelRows(p.Response, "Name") }

// InsightsPromLabelValues displays the values of a label.
type InsightsPromLabelValues struct {
	Response *godo.PromLabelsResponse
}

var _ Displayable = &InsightsPromLabelValues{}

func (p *InsightsPromLabelValues) JSON(out io.Writer) error {
	return writePromLabelsJSON(p.Response, out)
}

func (p *InsightsPromLabelValues) Cols() []string { return []string{"Value"} }

func (p *InsightsPromLabelValues) ColMap() map[string]string {
	return map[string]string{"Value": "Value"}
}

func (p *InsightsPromLabelValues) KV() []map[string]any { return promLabelRows(p.Response, "Value") }

func writePromLabelsJSON(resp *godo.PromLabelsResponse, out io.Writer) error {
	r := godo.PromLabelsResponse{Data: []string{}}
	if resp != nil {
		r.Status = resp.Status
		if resp.Data != nil {
			r.Data = resp.Data
		}
	}
	return writeJSON(r, out)
}

func promLabelRows(resp *godo.PromLabelsResponse, col string) []map[string]any {
	out := []map[string]any{}
	if resp == nil {
		return out
	}
	for _, v := range resp.Data {
		out = append(out, map[string]any{col: v})
	}
	return out
}

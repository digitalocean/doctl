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
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"github.com/digitalocean/doctl"
	yamlv2 "gopkg.in/yaml.v2"
	"sigs.k8s.io/yaml"
)

var (
	insightsStdin  io.Reader = os.Stdin
	insightsStderr io.Writer = os.Stderr

	insightsRelativeTimeRE = regexp.MustCompile(`^(now([+-]\d+[smhdw])?|\d+[smhdw])$`)
)

type stringArrayConfig interface {
	GetStringArray(ns, key string) ([]string, error)
}

func getInsightsStringArray(c *CmdConfig, key string) ([]string, error) {
	if config, ok := c.Doit.(stringArrayConfig); ok {
		return config.GetStringArray(c.NS, key)
	}
	return c.Doit.GetStringSlice(c.NS, key)
}

func insightsTrimmedString(c *CmdConfig, name string) (string, error) {
	value, err := c.Doit.GetString(c.NS, name)
	return strings.TrimSpace(value), err
}

func decodeInsightsJSON(r io.Reader, value any) error {
	decoder := json.NewDecoder(r)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(value); err != nil {
		return err
	}
	return ensureJSONEOF(decoder)
}

func readInsightsRequest[T any](stdin io.Reader, path string) (*T, error) {
	var (
		data []byte
		err  error
	)
	if path == "-" {
		data, err = io.ReadAll(stdin)
	} else {
		data, err = os.ReadFile(path)
	}
	if err != nil {
		return nil, fmt.Errorf("reading request file: %w", err)
	}

	jsonData, err := yaml.YAMLToJSON(data)
	if err != nil {
		return nil, fmt.Errorf("parsing request file: %w", err)
	}

	decoder := json.NewDecoder(bytes.NewReader(jsonData))
	decoder.DisallowUnknownFields()

	var request T
	if err := decoder.Decode(&request); err != nil {
		return nil, fmt.Errorf("parsing request file: %w", err)
	}
	if err := ensureJSONEOF(decoder); err != nil {
		return nil, fmt.Errorf("parsing request file: %w", err)
	}
	return &request, nil
}

// readInsightsDashboardSpec reads a dashboard spec document from path, or
// from stdin when path is "-". JSON input is sent as written rather than
// re-serialized; YAML input is converted to JSON. The file must contain
// exactly one document and it must be an object. The API may return the spec
// with keys reordered and whitespace normalized.
func readInsightsDashboardSpec(stdin io.Reader, path string) (json.RawMessage, error) {
	var (
		data []byte
		err  error
	)
	if path == "-" {
		data, err = io.ReadAll(stdin)
	} else {
		data, err = os.ReadFile(path)
	}
	if err != nil {
		return nil, fmt.Errorf("reading spec file: %w", err)
	}

	spec := bytes.TrimSpace(data)
	if len(spec) == 0 {
		return nil, errInsightsDashboardSpecNotObject
	}
	if json.Valid(spec) {
		if spec[0] != '{' {
			return nil, errInsightsDashboardSpecNotObject
		}
		return json.RawMessage(spec), nil
	}

	// Not JSON, so treat it as YAML. Flow-style YAML such as {kind: Dashboard}
	// also starts with "{", which is why the JSON check above is on validity
	// rather than on the first byte. yaml.YAMLToJSON only converts the first
	// document and silently drops anything after it, so count documents first
	// with the same parser it uses underneath.
	decoder := yamlv2.NewDecoder(bytes.NewReader(spec))
	var document, extra any
	if err := decoder.Decode(&document); err != nil {
		return nil, fmt.Errorf("parsing spec file: %w", err)
	}
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return nil, errInsightsDashboardSpecNotObject
	}

	converted, err := yaml.YAMLToJSON(spec)
	if err != nil {
		return nil, fmt.Errorf("parsing spec file: %w", err)
	}
	if len(converted) == 0 || converted[0] != '{' {
		return nil, errInsightsDashboardSpecNotObject
	}
	return json.RawMessage(converted), nil
}

var errInsightsDashboardSpecNotObject = fmt.Errorf("spec file must contain a single JSON object")

// insightsPageOptions reads --page and --per-page, returning zero for flags
// that were not set so the server applies its defaults.
func insightsPageOptions(c *CmdConfig, maxPerPage int) (page, perPage int, err error) {
	page, err = c.Doit.GetInt(c.NS, doctl.ArgInsightsPage)
	if err != nil {
		return 0, 0, err
	}
	if page < 0 || (page == 0 && c.Doit.IsSet(doctl.ArgInsightsPage)) {
		return 0, 0, fmt.Errorf("--%s must be at least 1", doctl.ArgInsightsPage)
	}
	perPage, err = c.Doit.GetInt(c.NS, doctl.ArgInsightsPerPage)
	if err != nil {
		return 0, 0, err
	}
	if perPage < 0 || perPage > maxPerPage ||
		(perPage == 0 && c.Doit.IsSet(doctl.ArgInsightsPerPage)) {
		return 0, 0, fmt.Errorf("--%s must be between 1 and %d", doctl.ArgInsightsPerPage, maxPerPage)
	}
	return page, perPage, nil
}

func ensureJSONEOF(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return fmt.Errorf("request file must contain exactly one object")
		}
		return err
	}
	return nil
}

func rejectRequestFileFlagMix(c *CmdConfig, requestFile string, flags ...string) error {
	if requestFile == "" {
		return nil
	}
	for _, flag := range flags {
		if c.Doit.IsSet(flag) {
			return fmt.Errorf("--request-file cannot be combined with --%s", flag)
		}
	}
	return nil
}

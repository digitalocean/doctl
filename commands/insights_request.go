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
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

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

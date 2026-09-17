/*
Copyright 2026 The Doctl Authors All rights reserved.
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
	"io"
	"os"
	"strings"
)

// Shared parsing for repeatable `KEY=VALUE` flags that carry sensitive values:
// `doctl secrets create --value` and `doctl harness-runtime create --secret`.
// Both spell the three value forms identically — a literal, `@path` to read a
// file, and `-` to read stdin — so a recipe learned on one works on the other.

// parseKeyValueInputs turns repeated KEY=VALUE flag values into a map,
// resolving each value through resolveKeyValueInput. Duplicate keys are an
// error rather than a silent last-one-wins, since which value survived would
// otherwise be invisible.
func parseKeyValueInputs(pairs []string) (map[string]string, error) {
	values := make(map[string]string, len(pairs))
	for _, pair := range pairs {
		key, value, err := parseKeyValueLine(pair)
		if err != nil {
			return nil, err
		}
		if _, exists := values[key]; exists {
			return nil, fmt.Errorf("duplicate key %q", key)
		}

		resolved, err := resolveKeyValueInput(value)
		if err != nil {
			return nil, err
		}
		values[key] = resolved
	}

	return values, nil
}

// parseKeyValueLine splits on the first "=" only, so values may contain "="
// (query strings, base64 padding, connection strings).
func parseKeyValueLine(line string) (string, string, error) {
	parts := strings.SplitN(strings.TrimSpace(line), "=", 2)
	if len(parts) != 2 || parts[0] == "" {
		return "", "", fmt.Errorf("invalid key-value pair %q, expected key=value", line)
	}

	return parts[0], parts[1], nil
}

// resolveKeyValueInput expands the indirection forms: "-" reads stdin and
// "@path" reads a file, both with the trailing newline trimmed so a value
// written by `echo` or an editor does not carry one. Anything else is literal.
func resolveKeyValueInput(value string) (string, error) {
	if value == "-" {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return "", err
		}
		return strings.TrimRight(string(data), "\r\n"), nil
	}

	if strings.HasPrefix(value, "@") {
		path := strings.TrimPrefix(value, "@")
		if path == "" {
			return "", fmt.Errorf("file path is required after @")
		}

		content, err := readInputFromFile(path)
		if err != nil {
			return "", err
		}

		return strings.TrimRight(content, "\r\n"), nil
	}

	return value, nil
}

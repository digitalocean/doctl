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
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateHostedAgentIdentifier(t *testing.T) {
	t.Parallel()

	valid := []string{
		"a",
		"ab",
		"gh-ci",
		"my.trigger_name",
		"Named-E2E-Test",
		strings.Repeat("a", 64),
	}
	for _, name := range valid {
		t.Run("valid/"+name, func(t *testing.T) {
			t.Parallel()
			assert.NoError(t, validateHostedAgentIdentifier(name))
		})
	}

	invalid := []string{
		"",
		"-bad",
		"bad-",
		".bad",
		"bad.",
		"_bad",
		"bad_",
		"<script>alert(1)</script>",
		"'; DROP TABLE sessions;--",
		"../../etc/passwd",
		strings.Repeat("a", 65),
		"019f275e-96dc-7ea0-98bd-9ecf2a0834c3",
	}
	for _, name := range invalid {
		t.Run("invalid/"+name, func(t *testing.T) {
			t.Parallel()
			assert.Error(t, validateHostedAgentIdentifier(name))
		})
	}
}

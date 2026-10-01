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

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
)

// placeholderByFlagType is the value name pflag derives from a flag's type when
// its usage string does not override it. Anything else means a back-quoted word
// in the usage was read as the placeholder.
var placeholderByFlagType = map[string]string{
	"string":      "string",
	"int":         "int",
	"bool":        "",
	"stringSlice": "strings",
	"stringArray": "strings",
	"duration":    "duration",
	"float64":     "float",
}

// Prose in a flag's usage string must not be back-quoted. pflag's UnquoteUsage
// takes the first back-quoted word as the flag's value placeholder, so
// "Same values as on `template create`" renders as
// `--base-template template create`, which reads as a two-word value the flag
// accepts. It is silent — help still renders, just wrongly — and it has now
// happened on three separate flags, so the rule is asserted rather than
// remembered. Quote commands with single quotes in usage strings.
func TestAgentFlagUsageStringsSetNoPlaceholder(t *testing.T) {
	for _, root := range []*Command{Agents(), AgentTemplates(), AgentConfigs(), AgentTriggers()} {
		walkCommands(root.Command, func(cmd *cobra.Command) {
			cmd.Flags().VisitAll(func(f *pflag.Flag) {
				// --format is exempt, and deliberately so: doctl back-quotes
				// column names there across every command in the repo, and a
				// column name genuinely is a value the flag takes. The rule
				// here is about prose, which that is not.
				if f.Name == "format" {
					return
				}
				want, known := placeholderByFlagType[f.Value.Type()]
				if !known {
					return
				}
				got, _ := pflag.UnquoteUsage(f)
				assert.Equal(t, want, got,
					"%s --%s: usage back-quotes %q, which pflag renders as the value placeholder; use single quotes instead",
					cmd.CommandPath(), f.Name, got)
			})
		})
	}
}

func walkCommands(cmd *cobra.Command, fn func(*cobra.Command)) {
	fn(cmd)
	for _, child := range cmd.Commands() {
		walkCommands(child, fn)
	}
}

// The guard above is only meaningful if it actually fires on the mistake, and
// the mistake is subtle enough that a reader should not have to take that on
// faith.
func TestAgentFlagUsagePlaceholderGuardCatchesBackquotedProse(t *testing.T) {
	fs := pflag.NewFlagSet("t", pflag.ContinueOnError)
	fs.String("base-template", "", "New platform base. Same values as on `template create`")

	got, usage := pflag.UnquoteUsage(fs.Lookup("base-template"))
	assert.Equal(t, "template create", got, "pflag takes the back-quoted words as the placeholder")
	assert.NotContains(t, usage, "`", "and strips the quotes from the help text, so nothing looks wrong")
	assert.NotEqual(t, placeholderByFlagType["string"], got)
	assert.True(t, strings.Contains(usage, "Same values as on template create"))
}

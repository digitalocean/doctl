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

	"github.com/digitalocean/doctl"
)

// agentStructuredOutput reports whether this invocation asked for output meant
// for another program rather than for a person: `-o json`, or `--format` /
// `--no-header`, which pick columns and drop the header row.
//
// Only the shared displayer honors those two flags, so every verb here that has
// a styled renderer of its own has to ask before choosing one. Printing the
// card anyway is the failure this guards against, and it is a quiet one: the
// command still exits 0, so a `$(...)` capture collects the whole rendered box
// and the problem only surfaces wherever that value is finally used.
//
// A lookup that fails reads as "not asked for": the only way either get can
// fail is a required flag left empty, which neither of these ever is, and
// c.Display re-reads both and returns the error itself if that changes.
func agentStructuredOutput(c *CmdConfig) bool {
	return Output == "json" || agentColumnOutput(c)
}

// agentColumnOutput reports whether the caller passed --format or --no-header,
// i.e. asked for the column table specifically. Verbs whose text output is
// more than one displayable row — a created trigger and the one-time secret
// that comes back with it, say — need to tell that apart from `-o json`, which
// can carry both in one document.
func agentColumnOutput(c *CmdConfig) bool {
	if c == nil || c.Doit == nil {
		return false
	}
	if cols, err := c.Doit.GetString(c.NS, doctl.ArgFormat); err == nil && strings.TrimSpace(cols) != "" {
		return true
	}
	noHeader, err := c.Doit.GetBool(c.NS, doctl.ArgNoHeader)
	return err == nil && noHeader
}

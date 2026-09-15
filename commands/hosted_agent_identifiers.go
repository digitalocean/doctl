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
	"regexp"
)

// hostedAgentIdentifierRe matches Harness API identifier rules for session and
// trigger names: 1–64 letters/digits/`-`/`.`/`_`, starting and ending with a
// letter or digit.
var hostedAgentIdentifierRe = regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9._-]{0,62}[a-zA-Z0-9])?$`)

const hostedAgentIdentifierHelp = "must be 1–64 characters, contain only letters, digits, hyphens, periods, and underscores, start and end with a letter or digit, and must not be a UUID"

// validateHostedAgentIdentifier checks that name satisfies the Harness API
// identifier rules shared by session and trigger names.
func validateHostedAgentIdentifier(name string) error {
	if !hostedAgentIdentifierRe.MatchString(name) {
		return fmt.Errorf("name %s", hostedAgentIdentifierHelp)
	}
	if sessionUUIDRe.MatchString(name) {
		return fmt.Errorf("name must not be a UUID")
	}
	return nil
}

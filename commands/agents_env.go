/*
Copyright 2026 The Doctl Authors All rights reserved.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
    15|See the License for the specific language governing permissions and
limitations under the License.
*/

package commands

import (
	"fmt"
	"os"
	"strings"

	"github.com/digitalocean/doctl/commands/charm/input"
	"golang.org/x/term"
)

// serverProvidedEnvPlaceholders are ${VAR} names filled by doctl after talking
// to an upstream API (e.g. OpenAI Agents mint ENV_ID). Never prompt for these.
var serverProvidedEnvPlaceholders = map[string]bool{
	"ENV_ID": true,
}

// promptEnvVarValue asks the user for a missing environment variable on an
// interactive TTY. Tests replace this to avoid real terminal prompts.
var promptEnvVarValue = defaultPromptEnvVarValue

// canPromptForEnv reports whether progressive secret collection is allowed.
func canPromptForEnv() bool {
	if !Interactive {
		return false
	}
	return term.IsTerminal(int(os.Stdin.Fd())) && term.IsTerminal(int(os.Stdout.Fd()))
}

// isSecretEnvVarName chooses a hidden prompt for credential-like names.
func isSecretEnvVarName(name string) bool {
	u := strings.ToUpper(name)
	for _, hint := range []string{"KEY", "TOKEN", "SECRET", "PASSWORD", "PASSWD", "CREDENTIAL", "AUTH"} {
		if strings.Contains(u, hint) {
			return true
		}
	}
	return false
}

func defaultPromptEnvVarValue(name string) (string, error) {
	if !canPromptForEnv() {
		return "", fmt.Errorf("environment variable %s is not set locally (set it in your environment, or re-run in an interactive terminal to be prompted)", name)
	}

	stylingEnabled = detectStyling()
	fmt.Fprintf(os.Stderr, "%s %s\n",
		colorize("•", colHighlight),
		fmt.Sprintf("%s is not set — enter it to continue", boldColor(name, colHighlight)))

	opts := []input.Option{input.WithRequired()}
	if isSecretEnvVarName(name) {
		opts = append(opts, input.WithHidden())
	}
	prompt := input.New(fmt.Sprintf("Enter %s: ", name), opts...)
	val, err := prompt.Prompt()
	if err != nil {
		return "", err
	}
	val = strings.TrimSpace(val)
	if val == "" {
		return "", fmt.Errorf("%s is required", name)
	}
	return val, nil
}

// ensureEnvVar returns the value of name from the process environment, or
// prompts for it (and Setenv's it) when interactive. Empty values are treated
// as missing so a blank export still triggers progressive collection.
func ensureEnvVar(name string) (string, error) {
	if v := strings.TrimSpace(os.Getenv(name)); v != "" {
		return v, nil
	}
	v, err := promptEnvVarValue(name)
	if err != nil {
		return "", err
	}
	if err := os.Setenv(name, v); err != nil {
		return "", fmt.Errorf("setting %s: %w", name, err)
	}
	return v, nil
}

// findMissingManifestEnvRefs returns unique ${VAR} names that lookup does not
// satisfy. Escaped $${VAR} forms are ignored. Server-provided placeholders are
// included so callers can decide whether to defer them.
func findMissingManifestEnvRefs(manifest []byte, lookup func(string) (string, bool)) []string {
	var missing []string
	seen := map[string]bool{}
	for _, m := range manifestEnvRef.FindAll(manifest, -1) {
		if len(m) >= 2 && m[0] == '$' && m[1] == '$' {
			continue
		}
		name := string(m[2 : len(m)-1])
		if _, ok := lookup(name); ok {
			continue
		}
		if seen[name] {
			continue
		}
		seen[name] = true
		missing = append(missing, name)
	}
	return missing
}

// ensureManifestEnvVars prompts (when interactive) for any missing ${VAR}
// references in the manifest and exports them into the process environment so
// later expand/attach steps see them. Server-provided placeholders like ENV_ID
// are skipped here — they must arrive via the caller's overlay.
func ensureManifestEnvVars(manifest []byte, lookup func(string) (string, bool)) error {
	missing := findMissingManifestEnvRefs(manifest, lookup)
	var need []string
	for _, name := range missing {
		if serverProvidedEnvPlaceholders[name] {
			continue
		}
		need = append(need, name)
	}
	if len(need) == 0 {
		return nil
	}

	if !canPromptForEnv() {
		return fmt.Errorf("manifest references environment variable(s) not set locally: %s (set them in your environment, escape a literal with $${...}, or re-run in an interactive terminal to be prompted)", strings.Join(need, ", "))
	}

	for _, name := range need {
		if _, err := ensureEnvVar(name); err != nil {
			return err
		}
	}
	return nil
}

// expandManifestEnvCollect resolves ${VAR} references after progressively
// collecting any missing values from the developer when possible.
func expandManifestEnvCollect(manifest []byte, lookup func(string) (string, bool)) ([]byte, error) {
	if err := ensureManifestEnvVars(manifest, lookup); err != nil {
		return nil, err
	}
	// Re-read env after prompts so Setenv'd values are visible even when the
	// original lookup closed over a snapshot.
	live := func(name string) (string, bool) {
		if lookup != nil {
			if v, ok := lookup(name); ok {
				return v, true
			}
		}
		return os.LookupEnv(name)
	}
	return expandManifestEnvLookup(manifest, live)
}

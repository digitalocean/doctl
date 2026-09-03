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
	"sort"
	"strings"

	"github.com/digitalocean/doctl"
	yaml "gopkg.in/yaml.v2"
)

// A tenant secret has no standalone API: its value only ever exists as a
// write-only `value` on a manifest secret slot, extracted server-side at create
// time and never returned. So `--secret NAME=VALUE` is manifest surgery, not an
// API call — it writes the slot the manifest declares, which lets a checked-in
// agents.yaml name the credentials it needs without carrying them.

// redactedSecretValue replaces secret values anywhere doctl echoes a manifest
// back to the user. It is deliberately a plausible value rather than an empty
// string so a redacted manifest stays structurally valid and can be piped into
// `config create --secret NAME=...`, which overrides it with the real value.
const redactedSecretValue = "REDACTED"

// tenantSecretSource is the only secret source doctl can populate client-side;
// other sources (e.g. OAuth assignments) are resolved server-side.
const tenantSecretSource = "tenantSecret"

// agentSecretFlags reads the repeatable --secret flag into name -> value,
// resolving @file and - the same way `doctl secrets create --value` does.
func agentSecretFlags(c *CmdConfig) (map[string]string, error) {
	pairs, err := c.Doit.GetStringSlice(c.NS, doctl.ArgAgentSecret)
	if err != nil {
		return nil, err
	}
	return parseAgentSecretPairs(pairs)
}

// parseAgentSecretPairs turns already-read --secret flag values into a map.
func parseAgentSecretPairs(pairs []string) (map[string]string, error) {
	if len(pairs) == 0 {
		return nil, nil
	}
	secrets, err := parseKeyValueInputs(pairs)
	if err != nil {
		return nil, fmt.Errorf("parsing --%s: %w", doctl.ArgAgentSecret, err)
	}
	for name, value := range secrets {
		if strings.TrimSpace(value) == "" {
			return nil, fmt.Errorf("--%s %s is empty (the @file or stdin value was blank). Pass a real key — an unset $VAR writes nothing",
				doctl.ArgAgentSecret, name)
		}
	}
	return secrets, nil
}

// rejectStdinSecretWithStdinManifest catches --secret NAME=- combined with a
// manifest on stdin. Both read os.Stdin, so the second one would see EOF and
// look like a mysterious empty-manifest or empty-secret failure.
func rejectStdinSecretWithStdinManifest(manifestPath string, pairs []string) error {
	if manifestPath != "-" {
		return nil
	}
	for _, pair := range pairs {
		_, value, err := parseKeyValueLine(pair)
		if err != nil {
			continue
		}
		if value == "-" {
			return fmt.Errorf("--%s NAME=- and a manifest on stdin (--%s -) cannot be combined: both read stdin. Pass the secret as NAME=@path or NAME=VALUE, or pass the manifest as a file",
				doctl.ArgAgentSecret, doctl.ArgAgentSpec)
		}
	}
	return nil
}

// manifestSecretValues returns name -> value for every secret slot that
// already has a value. Used so --secret ANTHROPIC_API_KEY=… satisfies the
// claude-code key check and ${ANTHROPIC_API_KEY} expansion, instead of still
// demanding the process environment.
func manifestSecretValues(manifest []byte) map[string]string {
	var doc map[string]any
	if err := yaml.Unmarshal(manifest, &doc); err != nil || doc == nil {
		return nil
	}

	container := doc
	if _, legacy := doc["apiVersion"]; legacy {
		spec, ok := yamlMap(doc["spec"])
		if !ok {
			return nil
		}
		container = spec
	}

	raw, present := container["secrets"]
	if !present || raw == nil {
		return nil
	}

	out := map[string]string{}
	if isYAMLMapping(raw) {
		shorthand, _ := yamlStringMap(raw)
		for name, value := range shorthand {
			if v := strings.TrimSpace(value); v != "" && v != redactedSecretValue {
				out[name] = v
			}
		}
		return out
	}
	for _, slot := range yamlListOrNil(raw) {
		m, ok := yamlMap(slot)
		if !ok {
			continue
		}
		name, _ := yamlString(m["name"])
		value, _ := yamlString(m["value"])
		name = strings.TrimSpace(name)
		value = strings.TrimSpace(value)
		if name == "" || value == "" || value == redactedSecretValue {
			continue
		}
		out[name] = value
	}
	return out
}

func yamlListOrNil(v any) []any {
	list, ok := yamlList(v)
	if !ok {
		return nil
	}
	return list
}

// dropEnvKeysCoveredBySecrets removes env entries that duplicate a secret
// slot. --harness claude-code writes ANTHROPIC_API_KEY: ${ANTHROPIC_API_KEY}
// into env so a missing key can be prompted; once --secret fills the slot,
// leaving both makes durable validation fail ("supply the value through
// secrets, not env") and is what made NAME=- look documented-but-broken.
func dropEnvKeysCoveredBySecrets(manifest []byte) []byte {
	secrets := manifestSecretValues(manifest)
	if len(secrets) == 0 {
		return manifest
	}

	var doc map[string]any
	if err := yaml.Unmarshal(manifest, &doc); err != nil || doc == nil {
		return manifest
	}

	container := doc
	legacy := false
	if _, ok := doc["apiVersion"]; ok {
		spec, ok := yamlMap(doc["spec"])
		if !ok {
			return manifest
		}
		container = spec
		legacy = true
	}

	env, ok := yamlStringMap(container["env"])
	if !ok || len(env) == 0 {
		return manifest
	}

	changed := false
	for name := range secrets {
		if _, exists := env[name]; exists {
			delete(env, name)
			changed = true
		}
	}
	if !changed {
		return manifest
	}

	if len(env) == 0 {
		delete(container, "env")
	} else {
		out := make(map[string]any, len(env))
		for k, v := range env {
			out[k] = v
		}
		container["env"] = out
	}
	if legacy {
		doc["spec"] = container
	}

	raw, err := yaml.Marshal(doc)
	if err != nil {
		return manifest
	}
	return raw
}

// injectManifestSecrets writes each name/value pair into the manifest's secret
// slots as a tenantSecret, overriding a same-named slot already declared in the
// file. Legacy envelopes take them under spec.secrets, flat manifests under a
// top-level secrets. An empty map returns the manifest untouched, so the
// no-flag path is byte-for-byte what the user wrote.
func injectManifestSecrets(manifest []byte, secrets map[string]string) ([]byte, error) {
	if len(secrets) == 0 {
		return manifest, nil
	}

	var doc map[string]any
	if err := yaml.Unmarshal(manifest, &doc); err != nil {
		return nil, fmt.Errorf("parsing manifest to apply --%s: %w", doctl.ArgAgentSecret, err)
	}
	if doc == nil {
		doc = map[string]any{}
	}

	// Legacy envelopes nest everything under spec; flat manifests are the doc
	// itself. Either way we mutate one map and write it back where it came from.
	container := doc
	_, legacy := doc["apiVersion"]
	if legacy {
		spec, ok := yamlMap(doc["spec"])
		if !ok {
			spec = map[string]any{}
		}
		container = spec
	}

	merged, err := mergeSecretSlots(container["secrets"], secrets)
	if err != nil {
		return nil, err
	}
	container["secrets"] = merged

	if legacy {
		doc["spec"] = container
	}

	out, err := yaml.Marshal(doc)
	if err != nil {
		return nil, fmt.Errorf("applying --%s to manifest: %w", doctl.ArgAgentSecret, err)
	}
	return out, nil
}

// mergeSecretSlots folds secrets into whatever shape the manifest already
// uses. The map shorthand (`secrets: {NAME: value}`) is preserved when present
// so we don't rewrite a user's file into a different style; everything else
// becomes the canonical list of {name, source, value} slots.
func mergeSecretSlots(existing any, secrets map[string]string) (any, error) {
	if shorthand, ok := yamlStringMap(existing); ok && existing != nil {
		out := make(map[string]any, len(shorthand)+len(secrets))
		for name, value := range shorthand {
			out[name] = value
		}
		for name, value := range secrets {
			out[name] = value
		}
		return out, nil
	}

	slots, ok := yamlList(existing)
	if existing != nil && !ok {
		return nil, fmt.Errorf("manifest secrets must be a list of slots or a name/value mapping to apply --%s", doctl.ArgAgentSecret)
	}

	out := make([]any, 0, len(slots)+len(secrets))
	applied := make(map[string]bool, len(secrets))
	for _, slot := range slots {
		m, ok := yamlMap(slot)
		if !ok {
			out = append(out, slot)
			continue
		}
		name, _ := yamlString(m["name"])
		value, override := secrets[name]
		if !override {
			out = append(out, slot)
			continue
		}
		// Keep the declared slot (it may carry other fields) and only swap in
		// the value, so `--secret` reads as "fill this in", not "replace it".
		m["value"] = value
		if source, _ := yamlString(m["source"]); source == "" {
			m["source"] = tenantSecretSource
		}
		out = append(out, m)
		applied[name] = true
	}

	// Deterministic order keeps --dry-run output stable between invocations,
	// which matters because it is meant to be diffed and piped.
	names := make([]string, 0, len(secrets))
	for name := range secrets {
		if !applied[name] {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	for _, name := range names {
		out = append(out, map[string]any{
			"name":   name,
			"source": tenantSecretSource,
			"value":  secrets[name],
		})
	}

	return out, nil
}

// redactManifestSecrets replaces every secret slot value with
// redactedSecretValue. Every path that echoes manifest bytes back to a user
// (--dry-run today) goes through this, so a value supplied by --secret or
// expanded from ${VAR} is never printed in plaintext. Returns the manifest
// unchanged when it declares no secrets or cannot be parsed as YAML — the
// server is the authority on manifest validity, and a parse failure here must
// not swallow the output the user asked to see.
func redactManifestSecrets(manifest []byte) []byte {
	var doc map[string]any
	if err := yaml.Unmarshal(manifest, &doc); err != nil || doc == nil {
		return manifest
	}

	container := doc
	_, legacy := doc["apiVersion"]
	if legacy {
		spec, ok := yamlMap(doc["spec"])
		if !ok {
			return manifest
		}
		container = spec
	}

	raw, present := container["secrets"]
	if !present || raw == nil {
		return manifest
	}

	switch {
	case isYAMLMapping(raw):
		shorthand, _ := yamlStringMap(raw)
		out := make(map[string]any, len(shorthand))
		for name := range shorthand {
			out[name] = redactedSecretValue
		}
		container["secrets"] = out
	default:
		slots, ok := yamlList(raw)
		if !ok {
			return manifest
		}
		out := make([]any, 0, len(slots))
		for _, slot := range slots {
			m, ok := yamlMap(slot)
			if !ok {
				out = append(out, slot)
				continue
			}
			if _, hasValue := m["value"]; hasValue {
				m["value"] = redactedSecretValue
			}
			out = append(out, m)
		}
		container["secrets"] = out
	}

	if legacy {
		doc["spec"] = container
	}
	out, err := yaml.Marshal(doc)
	if err != nil {
		return manifest
	}
	return out
}

// errRedactedSecretNames lists secret slots still holding the redaction
// sentinel. --dry-run deliberately emits a structurally valid manifest so it can
// be piped, which means the sentinel is one --secret away from being stored as
// a real credential:
//
//	create --dry-run | config create --spec -      # no --secret
//
// That would succeed and produce a config whose API key is the literal string
// "REDACTED", failing much later inside a sandbox with an auth error that points
// nowhere near this command. Callers check this before sending a manifest.
func errRedactedSecretNames(manifest []byte) []string {
	var doc map[string]any
	if err := yaml.Unmarshal(manifest, &doc); err != nil || doc == nil {
		return nil
	}

	container := doc
	if _, legacy := doc["apiVersion"]; legacy {
		spec, ok := yamlMap(doc["spec"])
		if !ok {
			return nil
		}
		container = spec
	}

	raw, present := container["secrets"]
	if !present || raw == nil {
		return nil
	}

	var names []string
	if isYAMLMapping(raw) {
		shorthand, _ := yamlStringMap(raw)
		for name, value := range shorthand {
			if value == redactedSecretValue {
				names = append(names, name)
			}
		}
	} else {
		slots, _ := yamlList(raw)
		for _, slot := range slots {
			m, ok := yamlMap(slot)
			if !ok {
				continue
			}
			if value, _ := yamlString(m["value"]); value == redactedSecretValue {
				name, _ := yamlString(m["name"])
				if name == "" {
					name = "(unnamed slot)"
				}
				names = append(names, name)
			}
		}
	}
	sort.Strings(names)
	return names
}

// rejectRedactedSecrets stops a redacted placeholder from being stored as a real
// credential, naming the flag that supplies the missing value.
func rejectRedactedSecrets(manifest []byte) error {
	names := errRedactedSecretNames(manifest)
	if len(names) == 0 {
		return nil
	}
	return fmt.Errorf(
		"secret %s still %s the redacted placeholder %q, which would be stored as the real value. "+
			"This manifest came from --%s; supply the value with --%s %s=VALUE (or NAME=@path)",
		strings.Join(names, ", "), map[bool]string{true: "holds", false: "hold"}[len(names) == 1],
		redactedSecretValue, doctl.ArgAgentDryRun, doctl.ArgAgentSecret, names[0])
}

// isYAMLMapping distinguishes the `secrets: {NAME: value}` shorthand from the
// canonical list form. yamlStringMap alone cannot: it coerces, so it would
// happily accept a list element too.
func isYAMLMapping(v any) bool {
	switch v.(type) {
	case map[string]any, map[any]any:
		return true
	default:
		return false
	}
}

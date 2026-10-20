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

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/godo"
	"github.com/spf13/viper"
)

// Regional routing for the harness-runtime command group (MARSOHS-2084).
//
// A MARS resource lives in exactly one region and is reachable only on that
// region's API host: there is no redirect, no forwarding and no cross-region
// lookup, so a request sent to the wrong region 404s. The region is carried by
// the hostname — `api-<region>.digitalocean.com` — and the region-less host
// `api.digitalocean.com` is served by the default region.
//
// `--region` is therefore a host selector. It is persistent on the group, so
// every subcommand accepts it either side of the subcommand name
// (`doctl harness-runtime --region mem1 create` and
// `doctl harness-runtime create --region mem1` are the same command).
//
// Three parts of the regional contract are not here, each waiting on something
// outside this repo:
//
//   - The create body carries no region. godo's session and config create
//     requests have no region field, so there is nothing to set and the host
//     is the only signal doctl sends (MARSOHS-2085).
//   - A create response's Location header is not followed for the wait and
//     attach that come after it. harness-api does not send one yet, and the
//     service layer in do/agents.go drops the response that would carry it
//     (MARSOHS-2078).
//   - A session id is not read for a region prefix. Whether ids outside the
//     default region carry one is an open question in the design, so a
//     follow-up command takes --region or the default region.

const (
	// defaultAgentRegion serves the region-less API host, so omitting --region
	// is exactly today's behavior and needs no host rewrite.
	defaultAgentRegion = "ric1"

	// agentRegionFlagKey is the viper key the group-level --region flag binds
	// to. Group-level rather than the per-command `<parent>.<command>.<flag>`
	// key AddStringFlag would produce, since one flag serves every subcommand.
	agentRegionFlagKey = agentCmdName + "." + doctl.ArgRegionSlug
)

// agentRegionPattern bounds what may be interpolated into a hostname. doctl
// deliberately does not carry a list of known regions — harness-api validates
// the region against its own registry and a local list would go stale the day
// a region opens — so this checks shape only.
var agentRegionPattern = regexp.MustCompile(`^[a-z0-9]+$`)

// addAgentRegionFlag registers --region on the group's persistent flag set.
func addAgentRegionFlag(cmd *Command) {
	cmd.PersistentFlags().String(doctl.ArgRegionSlug, "",
		fmt.Sprintf("Datacenter region to address, for example 'mem1'. Selects the regional API host; defaults to %s, which the region-less host serves. A resource is reachable only in the region that holds it.", defaultAgentRegion))
	viper.BindPFlag(agentRegionFlagKey, cmd.PersistentFlags().Lookup(doctl.ArgRegionSlug))
}

// agentRegionFlag returns the validated --region value, or "" when unset.
func agentRegionFlag() (string, error) {
	region := viper.GetString(agentRegionFlagKey)
	if region == "" {
		return "", nil
	}
	if !agentRegionPattern.MatchString(region) {
		return "", fmt.Errorf("invalid region %q: expected a region slug such as %s", region, defaultAgentRegion)
	}
	return region, nil
}

// resolvedAgentRegion is the region a command is addressing, for display. It
// falls back to the default region, which is what an unset flag means.
func resolvedAgentRegion() string {
	region, err := agentRegionFlag()
	if err != nil || region == "" {
		return defaultAgentRegion
	}
	return region
}

// regionAPIURL maps a region to its API base URL. The default region is served
// by the region-less host, so it returns "" and leaves the host alone.
func regionAPIURL(region string) string {
	if region == "" || region == defaultAgentRegion {
		return ""
	}
	return fmt.Sprintf("https://api-%s.digitalocean.com/", region)
}

// agentRegionBaseURL is the base URL agents traffic should use, honoring an
// explicit --api-url / DIGITALOCEAN_API_URL first. For callers that build a URL
// themselves instead of going through the godo client.
func agentRegionBaseURL() (string, error) {
	if raw := viper.GetString("api-url"); raw != "" {
		return raw, nil
	}
	region, err := agentRegionFlag()
	if err != nil {
		return "", err
	}
	if base := regionAPIURL(region); base != "" {
		return base, nil
	}
	return defaultAPIURL, nil
}

// agentRegionClientOpts returns the godo options that point the client at the
// requested region's host, and nil when there is nothing to change.
//
// GetGodoClient applies these before the --api-url override, so a host the user
// chose explicitly still wins and doctl only ever rewrites a default. Only the
// harness-runtime group registers --region, so this is nil for every other
// command.
func agentRegionClientOpts() ([]godo.ClientOpt, error) {
	region, err := agentRegionFlag()
	if err != nil {
		return nil, err
	}
	base := regionAPIURL(region)
	if base == "" {
		return nil, nil
	}
	return []godo.ClientOpt{godo.SetBaseURL(base)}, nil
}

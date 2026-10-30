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
	"net/url"
	"regexp"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
	"github.com/spf13/viper"
)

// Regional routing for the harness-runtime command group (MARSOHS-2084).
//
// A MARS resource lives in exactly one region and is reachable only on that
// region's API host. There is no redirect, no forwarding and no cross-region
// lookup: asking one region for a resource another region holds is a 404, and
// a regional host whose region the server does not serve is refused with 421.
//
// The hostname is the whole signal — `api-<region>.digitalocean.com`, with the
// region-less `api.digitalocean.com` served by the default region. harness-api
// reads the forwarded Host itself; nothing carries the region in a request
// header.
//
// `--region` is therefore a host selector. It is persistent on the group, so
// every subcommand accepts it either side of the subcommand name
// (`doctl harness-runtime --region mem1 create` and
// `doctl harness-runtime create --region mem1` are the same command).

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

// regionAPIURL maps a region to its API base URL. An unset region leaves the
// host alone, so a command without the flag calls the region-less host exactly
// as before. A region given explicitly always selects that region's host, the
// default region included: --region ric1 addresses api-ric1 rather than
// silently falling back to the region-less name.
func regionAPIURL(region string) string {
	if region == "" {
		return ""
	}
	return godo.HostedAgentsRegionalBaseURL(region)
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

// locationBaseURL reduces a session's canonical URL to the base URL of the host
// serving it. A Location that cannot be read yields "", which leaves the host
// doctl already called in use: an unreadable header is not worth failing a
// session that was created successfully.
func locationBaseURL(location string) string {
	u, err := url.Parse(location)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return ""
	}
	return (&url.URL{Scheme: u.Scheme, Host: u.Host, Path: "/"}).String()
}

// useSessionLocationHost repoints the hosted-agents service at the host named
// by a created session's Location header, so the wait, prompt and attach that
// follow a create land in the region that actually holds the session rather
// than the one doctl happened to call.
//
// It is a no-op unless the server sent a Location — it does so only for teams
// on the regional routing rollout — and when --api-url names a host the user
// chose, which doctl does not override.
func useSessionLocationHost(c *CmdConfig, sess *do.HostedAgentSession) error {
	if sess == nil || sess.HostedAgentSession == nil || sess.Location == "" {
		return nil
	}
	if viper.GetString("api-url") != "" {
		return nil
	}
	base := locationBaseURL(sess.Location)
	if base == "" {
		return nil
	}

	godoClient, err := c.Doit.GetGodoClient(Trace, true, c.getContextAccessToken(), godo.SetBaseURL(base))
	if err != nil {
		return fmt.Errorf("unable to initialize DigitalOcean API client: %w", err)
	}
	c.HostedAgents = func() do.HostedAgentsService { return do.NewHostedAgentsService(godoClient) }
	return nil
}

// reportSessionRegion names the region a new session was placed in. The server
// reports it only for teams on the regional routing rollout, and structured
// output carries it as a field, so neither case prints anything here.
func reportSessionRegion(c *CmdConfig, sess *do.HostedAgentSession) {
	if sess == nil || sess.HostedAgentSession == nil || sess.Region == "" {
		return
	}
	if agentStructuredOutput(c) {
		return
	}
	fmt.Fprintln(c.Out, colorize("Region: "+sess.Region, colMuted))
}

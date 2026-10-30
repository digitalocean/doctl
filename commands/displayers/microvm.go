/*
Copyright 2025 The Doctl Authors All rights reserved.
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

package displayers

import (
	"fmt"
	"io"
	"strings"

	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
)

type MicroVM struct {
	MicroVMs do.MicroVMs
}

var _ Displayable = &MicroVM{}

func (m *MicroVM) JSON(out io.Writer) error {
	return writeJSON(m.MicroVMs, out)
}

func (m *MicroVM) Cols() []string {
	return []string{
		"ID", "Name", "Region", "State", "Size", "Networking", "Source", "Endpoint", "Ports", "Protocol", "Tags", "FailureReason", "Created",
	}
}

func (m *MicroVM) ColMap() map[string]string {
	return map[string]string{
		"ID":            "ID",
		"Name":          "Name",
		"Region":        "Region",
		"State":         "State",
		"Size":          "Size",
		"Networking":    "Networking",
		"Source":        "Source",
		"Endpoint":      "Endpoint",
		"Ports":         "Ports",
		"Protocol":      "Protocol",
		"Tags":          "Tags",
		"FailureReason": "Failure Reason",
		"Created":       "Created At",
		"Egress":        "Egress",
	}
}

func (m *MicroVM) KV() []map[string]any {
	out := make([]map[string]any, 0, len(m.MicroVMs))
	for _, md := range m.MicroVMs {
		out = append(out, map[string]any{
			"ID":            md.ID,
			"Name":          md.Name,
			"Region":        md.Region,
			"State":         string(md.State),
			"Size":          formatMicroVMSize(md.Size),
			"Networking":    string(md.Networking),
			"Source":        formatMicroVMSource(md.Source),
			"Endpoint":      defaultMicroVMHostname(md.URLs),
			"Ports":         formatPorts(md.Ports),
			"Protocol":      string(md.HTTPProtocol),
			"Tags":          strings.Join(md.Tags, ","),
			"FailureReason": md.FailureReason,
			"Created":       md.Created,
			"Egress":        formatMicroVMEgress(md.EgressPolicy),
		})
	}
	return out
}

type MicroVMURL struct {
	URLs []godo.MicroVMURL
}

var _ Displayable = &MicroVMURL{}

func (u *MicroVMURL) JSON(out io.Writer) error {
	return writeJSON(u.URLs, out)
}

func (u *MicroVMURL) Cols() []string {
	return []string{"ID", "Hostname", "Port", "Default", "Status"}
}

func (u *MicroVMURL) ColMap() map[string]string {
	return map[string]string{
		"ID":       "ID",
		"Hostname": "Hostname",
		"Port":     "Port",
		"Default":  "Default",
		"Status":   "Status",
	}
}

func (u *MicroVMURL) KV() []map[string]any {
	out := make([]map[string]any, 0, len(u.URLs))
	for _, url := range u.URLs {
		out = append(out, map[string]any{
			"ID":       url.ID,
			"Hostname": url.Hostname,
			"Port":     url.Port,
			"Default":  url.Default,
			"Status":   string(url.Status),
		})
	}
	return out
}

// MicroVMAccessToken displays MicroVM access tokens. Set Created for the
// create response so the one-time token value is shown; set AccessTokens
// for a list.
type MicroVMAccessToken struct {
	AccessTokens []godo.MicroVMAccessToken
	Created      *godo.MicroVMAccessTokenCreated
}

var _ Displayable = &MicroVMAccessToken{}

func (t *MicroVMAccessToken) JSON(out io.Writer) error {
	if t.Created != nil {
		return writeJSON(t.Created, out)
	}
	return writeJSON(t.AccessTokens, out)
}

func (t *MicroVMAccessToken) Cols() []string {
	cols := []string{"ID", "Name", "Created", "ExpiresAt"}
	if t.Created != nil {
		cols = append(cols, "Token")
	}
	return cols
}

func (t *MicroVMAccessToken) ColMap() map[string]string {
	return map[string]string{
		"ID":        "ID",
		"Name":      "Name",
		"Created":   "Created At",
		"ExpiresAt": "Expires At",
		"Token":     "Token",
	}
}

func (t *MicroVMAccessToken) KV() []map[string]any {
	if t.Created != nil {
		kv := microVMAccessTokenKV(t.Created.MicroVMAccessToken)
		kv["Token"] = t.Created.Token
		return []map[string]any{kv}
	}
	out := make([]map[string]any, 0, len(t.AccessTokens))
	for _, tok := range t.AccessTokens {
		out = append(out, microVMAccessTokenKV(tok))
	}
	return out
}

func microVMAccessTokenKV(tok godo.MicroVMAccessToken) map[string]any {
	expires := tok.ExpiresAt
	if expires == "" {
		expires = "-"
	}
	return map[string]any{
		"ID":        tok.ID,
		"Name":      tok.Name,
		"Created":   tok.Created,
		"ExpiresAt": expires,
	}
}

type MicroVMFile struct {
	Info *godo.MicroVMFileInfo
}

var _ Displayable = &MicroVMFile{}

func (f *MicroVMFile) JSON(out io.Writer) error {
	return writeJSON(f.Info, out)
}

func (f *MicroVMFile) Cols() []string {
	return []string{"Path", "SizeBytes", "SHA256", "Mode", "UID", "GID"}
}

func (f *MicroVMFile) ColMap() map[string]string {
	return map[string]string{
		"Path":      "Path",
		"SizeBytes": "Size (bytes)",
		"SHA256":    "SHA256",
		"Mode":      "Mode",
		"UID":       "UID",
		"GID":       "GID",
	}
}

func (f *MicroVMFile) KV() []map[string]any {
	if f.Info == nil {
		return nil
	}
	return []map[string]any{{
		"Path":      f.Info.Path,
		"SizeBytes": f.Info.SizeBytes,
		"SHA256":    f.Info.SHA256,
		"Mode":      f.Info.Mode,
		"UID":       f.Info.UID,
		"GID":       f.Info.GID,
	}}
}

type MicroVMCheckpoint struct {
	Checkpoints do.MicroVMCheckpoints
}

var _ Displayable = &MicroVMCheckpoint{}

func (c *MicroVMCheckpoint) JSON(out io.Writer) error {
	return writeJSON(c.Checkpoints, out)
}

func (c *MicroVMCheckpoint) Cols() []string {
	return []string{
		"ID", "MicroVMID", "MicroVMName", "Name", "Region", "Size", "Status", "MemoryBytes", "DiskBytes", "Created",
	}
}

func (c *MicroVMCheckpoint) ColMap() map[string]string {
	return map[string]string{
		"ID":          "ID",
		"MicroVMID":   "MicroVM ID",
		"MicroVMName": "MicroVM Name",
		"Name":        "Name",
		"Region":      "Region",
		"Size":        "Size",
		"Status":      "Status",
		"MemoryBytes": "Memory Bytes",
		"DiskBytes":   "Disk Bytes",
		"Created":     "Created At",
	}
}

func (c *MicroVMCheckpoint) KV() []map[string]any {
	out := make([]map[string]any, 0, len(c.Checkpoints))
	for _, cp := range c.Checkpoints {
		out = append(out, map[string]any{
			"ID":          cp.ID,
			"MicroVMID":   cp.MicroVMID,
			"MicroVMName": cp.MicroVMName,
			"Name":        cp.Name,
			"Region":      cp.Region,
			"Size":        formatMicroVMSize(cp.Size),
			"Status":      string(cp.Status),
			"MemoryBytes": cp.MemoryBytes,
			"DiskBytes":   cp.DiskBytes,
			"Created":     cp.Created,
		})
	}
	return out
}

type MicroVMCreateOptions struct {
	Options *godo.MicroVMCreateOptions
}

var _ Displayable = &MicroVMCreateOptions{}

func (o *MicroVMCreateOptions) JSON(out io.Writer) error {
	return writeJSON(o.Options, out)
}

func (o *MicroVMCreateOptions) Cols() []string {
	return []string{"DefaultRegion", "Sizes", "Features", "AccountLimits"}
}

func (o *MicroVMCreateOptions) ColMap() map[string]string {
	return map[string]string{
		"DefaultRegion": "Default Region",
		"Sizes":         "Sizes",
		"Features":      "Features",
		"AccountLimits": "Account Limits",
	}
}

func (o *MicroVMCreateOptions) KV() []map[string]any {
	if o.Options == nil {
		return nil
	}
	sizes := make([]string, 0, len(o.Options.Sizes))
	for _, s := range o.Options.Sizes {
		sizes = append(sizes, formatMicroVMSizeOption(s))
	}
	features := make([]string, 0, len(o.Options.Features))
	for _, f := range o.Options.Features {
		state := "off"
		if f.Enabled {
			state = "on"
		}
		features = append(features, fmt.Sprintf("%s=%s", f.Name, state))
	}
	return []map[string]any{{
		"DefaultRegion": o.Options.DefaultRegion,
		"Sizes":         strings.Join(sizes, ","),
		"Features":      strings.Join(features, ","),
		"AccountLimits": formatAccountLimits(o.Options.AccountLimits),
	}}
}

func formatMicroVMSize(size *godo.MicroVMSize) string {
	if size == nil {
		return ""
	}
	return fmt.Sprintf("%dvCPU/%dMiB/%dGB", size.CPU, size.Memory, size.Disk)
}

func formatMicroVMSizeOption(s godo.MicroVMSizeOption) string {
	size := fmt.Sprintf("%dvCPU/%dMiB/%dGB", s.CPU, s.Memory, s.Disk)
	if s.Pricing != nil {
		size = fmt.Sprintf("%s $%g/hr", size, s.Pricing.PricePerHour)
	}
	if !s.Available {
		size = "unavailable " + size
	}
	if len(s.Regions) == 0 {
		return size
	}
	return fmt.Sprintf("%s[%s]", size, strings.Join(s.Regions, ","))
}

func formatAccountLimits(l *godo.MicroVMAccountLimits) string {
	if l == nil {
		return ""
	}
	return fmt.Sprintf("running=%d total=%d mem=%d disk=%d idle=%ds",
		l.MaxConcurrentRunning, l.MaxTotalCount, l.MaxMemoryBytes, l.MaxDiskBytes, l.MaxIdleTimeoutSeconds)
}

func formatMicroVMSource(src *godo.MicroVMSource) string {
	if src == nil {
		return ""
	}
	if src.OCIRef != "" {
		return src.OCIRef
	}
	return src.CheckpointID
}

func defaultMicroVMHostname(urls []godo.MicroVMURL) string {
	for _, u := range urls {
		if u.Default {
			return u.Hostname
		}
	}
	if len(urls) > 0 {
		return urls[0].Hostname
	}
	return ""
}

// formatMicroVMEgress renders a policy as "deny: a.example.com,203.0.113.5".
// An empty string means the MicroVM has no policy and egress is unrestricted.
func formatMicroVMEgress(p *godo.MicroVMEgressPolicy) string {
	if p == nil {
		return ""
	}
	action := string(p.DefaultAction)
	if action == "" {
		action = string(godo.MicroVMEgressActionDeny)
	}
	hosts := make([]string, 0, len(p.Rules))
	for _, r := range p.Rules {
		if r.Domain != "" {
			hosts = append(hosts, r.Domain)
		} else {
			hosts = append(hosts, r.IP)
		}
	}
	return action + ": " + strings.Join(hosts, ",")
}

func formatPorts(ports []uint32) string {
	if len(ports) == 0 {
		return ""
	}
	parts := make([]string, len(ports))
	for i, p := range ports {
		parts[i] = fmt.Sprintf("%d", p)
	}
	return strings.Join(parts, ",")
}

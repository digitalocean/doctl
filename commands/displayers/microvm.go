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
		})
	}
	return out
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

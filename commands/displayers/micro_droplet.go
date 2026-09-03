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

type MicroDroplet struct {
	MicroDroplets do.MicroDroplets
}

var _ Displayable = &MicroDroplet{}

func (m *MicroDroplet) JSON(out io.Writer) error {
	return writeJSON(m.MicroDroplets, out)
}

func (m *MicroDroplet) Cols() []string {
	return []string{
		"ID", "Name", "Region", "State", "Size", "Networking", "Source", "Endpoint", "Ports", "Created",
	}
}

func (m *MicroDroplet) ColMap() map[string]string {
	return map[string]string{
		"ID":         "ID",
		"Name":       "Name",
		"Region":     "Region",
		"State":      "State",
		"Size":       "Size",
		"Networking": "Networking",
		"Source":     "Source",
		"Endpoint":   "Endpoint",
		"Ports":      "Ports",
		"Created":    "Created At",
	}
}

func (m *MicroDroplet) KV() []map[string]any {
	out := make([]map[string]any, 0, len(m.MicroDroplets))
	for _, md := range m.MicroDroplets {
		out = append(out, map[string]any{
			"ID":         md.ID,
			"Name":       md.Name,
			"Region":     md.Region,
			"State":      string(md.State),
			"Size":       formatMicroDropletSize(md.Size),
			"Networking": string(md.Networking),
			"Source":     formatMicroDropletSource(md.Source),
			"Endpoint":   defaultMicroDropletHostname(md.URLs),
			"Ports":      formatPorts(md.Ports),
			"Created":    md.Created,
		})
	}
	return out
}

type MicroDropletCheckpoint struct {
	Checkpoints do.MicroDropletCheckpoints
}

var _ Displayable = &MicroDropletCheckpoint{}

func (c *MicroDropletCheckpoint) JSON(out io.Writer) error {
	return writeJSON(c.Checkpoints, out)
}

func (c *MicroDropletCheckpoint) Cols() []string {
	return []string{
		"ID", "MicroDropletID", "MicroDropletName", "Name", "Region", "Status", "MemoryBytes", "DiskBytes", "Created",
	}
}

func (c *MicroDropletCheckpoint) ColMap() map[string]string {
	return map[string]string{
		"ID":               "ID",
		"MicroDropletID":   "MicroDroplet ID",
		"MicroDropletName": "MicroDroplet Name",
		"Name":             "Name",
		"Region":           "Region",
		"Status":           "Status",
		"MemoryBytes":      "Memory Bytes",
		"DiskBytes":        "Disk Bytes",
		"Created":          "Created At",
	}
}

func (c *MicroDropletCheckpoint) KV() []map[string]any {
	out := make([]map[string]any, 0, len(c.Checkpoints))
	for _, cp := range c.Checkpoints {
		out = append(out, map[string]any{
			"ID":               cp.ID,
			"MicroDropletID":   cp.MicroDropletID,
			"MicroDropletName": cp.MicroDropletName,
			"Name":             cp.Name,
			"Region":           cp.Region,
			"Status":           string(cp.Status),
			"MemoryBytes":      cp.MemoryBytes,
			"DiskBytes":        cp.DiskBytes,
			"Created":          cp.Created,
		})
	}
	return out
}

type MicroDropletCreateOptions struct {
	Options *godo.MicroDropletCreateOptions
}

var _ Displayable = &MicroDropletCreateOptions{}

func (o *MicroDropletCreateOptions) JSON(out io.Writer) error {
	return writeJSON(o.Options, out)
}

func (o *MicroDropletCreateOptions) Cols() []string {
	return []string{"DefaultRegion", "Regions", "Sizes", "Features"}
}

func (o *MicroDropletCreateOptions) ColMap() map[string]string {
	return map[string]string{
		"DefaultRegion": "Default Region",
		"Regions":       "Regions",
		"Sizes":         "Sizes",
		"Features":      "Features",
	}
}

func (o *MicroDropletCreateOptions) KV() []map[string]any {
	if o.Options == nil {
		return nil
	}
	regions := make([]string, 0, len(o.Options.Regions))
	for _, r := range o.Options.Regions {
		mark := "unavailable"
		if r.Available {
			mark = "available"
		}
		regions = append(regions, fmt.Sprintf("%s(%s)", r.Slug, mark))
	}
	sizes := make([]string, 0, len(o.Options.Sizes))
	for _, s := range o.Options.Sizes {
		sizes = append(sizes, formatMicroDropletSize(&s.Size))
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
		"Regions":       strings.Join(regions, ","),
		"Sizes":         strings.Join(sizes, ","),
		"Features":      strings.Join(features, ","),
	}}
}

func formatMicroDropletSize(size *godo.MicroDropletSize) string {
	if size == nil {
		return ""
	}
	return fmt.Sprintf("%dvCPU/%dMiB/%dGB", size.CPU, size.Memory, size.Disk)
}

func formatMicroDropletSource(src *godo.MicroDropletSource) string {
	if src == nil {
		return ""
	}
	if src.OCIRef != "" {
		return src.OCIRef
	}
	return src.CheckpointID
}

func defaultMicroDropletHostname(urls []godo.MicroDropletURL) string {
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

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
	"io"

	"github.com/digitalocean/doctl/do"
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
		"ID", "Name", "Region", "State", "Size", "Networking", "Image", "Endpoint", "Created",
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
		"Image":      "Image",
		"Endpoint":   "Endpoint",
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
			"Size":       md.Size,
			"Networking": string(md.Networking),
			"Image":      md.Image,
			"Endpoint":   md.Endpoint,
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
		"ID", "MicroDropletID", "Name", "Status", "MemoryBytes", "DiskBytes", "Created",
	}
}

func (c *MicroDropletCheckpoint) ColMap() map[string]string {
	return map[string]string{
		"ID":             "ID",
		"MicroDropletID": "MicroDroplet ID",
		"Name":           "Name",
		"Status":         "Status",
		"MemoryBytes":    "Memory Bytes",
		"DiskBytes":      "Disk Bytes",
		"Created":        "Created At",
	}
}

func (c *MicroDropletCheckpoint) KV() []map[string]any {
	out := make([]map[string]any, 0, len(c.Checkpoints))
	for _, cp := range c.Checkpoints {
		out = append(out, map[string]any{
			"ID":             cp.ID,
			"MicroDropletID": cp.MicroDropletID,
			"Name":           cp.Name,
			"Status":         string(cp.Status),
			"MemoryBytes":    cp.MemoryBytes,
			"DiskBytes":      cp.DiskBytes,
			"Created":        cp.Created,
		})
	}
	return out
}

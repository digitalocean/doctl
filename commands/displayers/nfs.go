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
	"strconv"
	"strings"

	"github.com/digitalocean/doctl/do"
)

type Nfs struct {
	NfsShares []do.Nfs
}

var _ Displayable = &Nfs{}

func (n *Nfs) JSON(out io.Writer) error {
	return writeJSON(n.NfsShares, out)
}

func (n *Nfs) Cols() []string {
	return []string{
		"ID", "Name", "Size", "Region", "Status", "CreatedAt", "VpcIDs",
	}
}

func (n *Nfs) ColMap() map[string]string {
	return map[string]string{
		"ID":        "ID",
		"Name":      "Name",
		"Size":      "Size",
		"Region":    "Region",
		"Status":    "Status",
		"CreatedAt": "Created At",
		"VpcIDs":    "VPC IDs",
	}
}

func (n *Nfs) KV() []map[string]any {
	out := make([]map[string]any, 0, len(n.NfsShares))
	for _, nfs := range n.NfsShares {
		m := map[string]any{
			"ID":        nfs.ID,
			"Name":      nfs.Name,
			"Size":      strconv.Itoa(nfs.SizeGib) + " GiB",
			"Region":    nfs.Region,
			"Status":    nfs.Status,
			"CreatedAt": nfs.CreatedAt,
			"VpcIDs":    strings.Join(nfs.VpcIDs, ", "),
		}
		out = append(out, m)
	}
	return out
}

type NfsSnapshot struct {
	NfsSnapshots []do.NfsSnapshot
}

var _ Displayable = &NfsSnapshot{}

func (n *NfsSnapshot) JSON(out io.Writer) error {
	return writeJSON(n.NfsSnapshots, out)
}

func (n *NfsSnapshot) Cols() []string {
	return []string{
		"ID", "Name", "Size", "Region", "Status", "CreatedAt", "ShareID",
	}
}

func (n *NfsSnapshot) ColMap() map[string]string {
	return map[string]string{
		"ID":        "ID",
		"Name":      "Name",
		"Size":      "Size",
		"Region":    "Region",
		"Status":    "Status",
		"CreatedAt": "Created At",
		"ShareID":   "Share ID",
	}
}

func (n *NfsSnapshot) KV() []map[string]any {
	out := make([]map[string]any, 0, len(n.NfsSnapshots))
	for _, snapshot := range n.NfsSnapshots {
		m := map[string]any{
			"ID":        snapshot.ID,
			"Name":      snapshot.Name,
			"Size":      strconv.Itoa(snapshot.SizeGib) + " GiB",
			"Region":    snapshot.Region,
			"Status":    snapshot.Status,
			"CreatedAt": snapshot.CreatedAt,
			"ShareID":   snapshot.ShareID,
		}
		out = append(out, m)
	}
	return out
}

type NfsAction struct {
	NfsActions []do.NfsAction
}

var _ Displayable = &NfsAction{}

func (na *NfsAction) JSON(out io.Writer) error {
	return writeJSON(na.NfsActions, out)
}

func (na *NfsAction) Cols() []string {
	return []string{
		"Status", "Type", "StartedAt", "ResourceID", "ResourceType", "Region",
	}
}

func (na *NfsAction) ColMap() map[string]string {
	return map[string]string{
		"Status":       "Status",
		"Type":         "Type",
		"StartedAt":    "Started At",
		"ResourceID":   "Resource ID",
		"ResourceType": "Resource Type",
		"Region":       "Region",
	}
}

func (na *NfsAction) KV() []map[string]any {
	out := make([]map[string]any, 0, len(na.NfsActions))
	for _, x := range na.NfsActions {
		region := ""
		if x.Region != nil {
			region = x.Region.Slug
		} else if x.RegionSlug != "" {
			region = x.RegionSlug
		}
		o := map[string]any{
			"Status": x.Status, "Type": x.Type,
			"StartedAt":  x.StartedAt,
			"ResourceID": x.ResourceID, "ResourceType": x.ResourceType,
			"Region": region,
		}
		out = append(out, o)
	}

	return out
}

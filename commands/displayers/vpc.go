/*
Copyright 2018 The Doctl Authors All rights reserved.
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

type VPC struct {
	VPCs do.VPCs
}

var _ Displayable = &VPC{}

func (v *VPC) JSON(out io.Writer) error {
	return writeJSON(v.VPCs, out)
}

func (v *VPC) Cols() []string {
	return []string{
		"ID",
		"URN",
		"Name",
		"Description",
		"IPRange",
		"Region",
		"Created",
		"Default",
	}
}

func (v *VPC) ColMap() map[string]string {
	return map[string]string{
		"ID":          "ID",
		"URN":         "URN",
		"Name":        "Name",
		"Description": "Description",
		"IPRange":     "IP Range",
		"Region":      "Region",
		"Created":     "Created At",
		"Default":     "Default",
	}
}

func (v *VPC) KV() []map[string]interface{} {
	out := make([]map[string]interface{}, 0, len(v.VPCs))

	for _, v := range v.VPCs {
		o := map[string]interface{}{
			"ID":          v.ID,
			"URN":         v.URN,
			"Name":        v.Name,
			"Description": v.Description,
			"IPRange":     v.IPRange,
			"Created":     v.CreatedAt,
			"Region":      v.RegionSlug,
			"Default":     v.Default,
		}
		out = append(out, o)
	}

	return out
}

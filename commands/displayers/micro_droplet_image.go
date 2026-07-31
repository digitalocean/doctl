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

type MicroDropletImage struct {
	Images do.MicroDropletImages
}

var _ Displayable = &MicroDropletImage{}

func (i *MicroDropletImage) JSON(out io.Writer) error {
	return writeJSON(i.Images, out)
}

func (i *MicroDropletImage) Cols() []string {
	return []string{"ID", "Name", "Source", "Status", "Created"}
}

func (i *MicroDropletImage) ColMap() map[string]string {
	return map[string]string{
		"ID":      "ID",
		"Name":    "Name",
		"Source":  "Source",
		"Status":  "Status",
		"Created": "Created At",
	}
}

func (i *MicroDropletImage) KV() []map[string]any {
	out := make([]map[string]any, 0, len(i.Images))
	for _, img := range i.Images {
		out = append(out, map[string]any{
			"ID":      img.ID,
			"Name":    img.Name,
			"Source":  img.Source,
			"Status":  string(img.Status),
			"Created": img.Created,
		})
	}
	return out
}

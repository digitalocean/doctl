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

type Image struct {
	Images do.Images
}

var _ Displayable = &Image{}

func (gi *Image) JSON(out io.Writer) error {
	return writeJSON(gi.Images, out)
}

func (gi *Image) Cols() []string {
	return []string{
		"ID", "Name", "Type", "Distribution", "Slug", "Public", "MinDisk",
	}
}

func (gi *Image) ColMap() map[string]string {
	return map[string]string{
		"ID": "ID", "Name": "Name", "Type": "Type", "Distribution": "Distribution",
		"Slug": "Slug", "Public": "Public", "MinDisk": "Min Disk",
	}
}

func (gi *Image) KV() []map[string]interface{} {
	out := make([]map[string]interface{}, 0, len(gi.Images))

	for _, i := range gi.Images {
		publicStatus := false
		if i.Public {
			publicStatus = true
		}

		o := map[string]interface{}{
			"ID": i.ID, "Name": i.Name, "Type": i.Type, "Distribution": i.Distribution,
			"Slug": i.Slug, "Public": publicStatus, "MinDisk": i.MinDiskSize,
		}

		out = append(out, o)
	}

	return out
}

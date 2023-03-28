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
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/digitalocean/doctl/do"
)

type Volume struct {
	Volumes []do.Volume
}

var _ Displayable = &Volume{}

func (a *Volume) JSON(out io.Writer) error {
	return writeJSON(a.Volumes, out)

}

func (a *Volume) Cols() []string {
	return []string{
		"ID", "Name", "Size", "Region", "Filesystem Type", "Filesystem Label", "DropletIDs", "Tags",
	}
}

func (a *Volume) ColMap() map[string]string {
	return map[string]string{
		"ID":               "ID",
		"Name":             "Name",
		"Size":             "Size",
		"Region":           "Region",
		"Filesystem Type":  "Filesystem Type",
		"Filesystem Label": "Filesystem Label",
		"DropletIDs":       "Droplet IDs",
		"Tags":             "Tags",
	}

}

func (a *Volume) KV() []map[string]interface{} {
	out := make([]map[string]interface{}, 0, len(a.Volumes))
	for _, volume := range a.Volumes {
		m := map[string]interface{}{
			"ID":               volume.ID,
			"Name":             volume.Name,
			"Size":             strconv.FormatInt(volume.SizeGigaBytes, 10) + " GiB",
			"Filesystem Type":  volume.FilesystemType,
			"Filesystem Label": volume.FilesystemLabel,
			"Tags":             strings.Join(volume.Tags, ","),
		}
		m["Region"] = ""
		if volume.Region != nil {
			m["Region"] = volume.Region.Slug
		}
		m["DropletIDs"] = ""
		if len(volume.DropletIDs) != 0 {
			m["DropletIDs"] = fmt.Sprintf("%v", volume.DropletIDs)
		}
		out = append(out, m)

	}
	return out
}

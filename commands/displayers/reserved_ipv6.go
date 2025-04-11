/*
Copyright 2024 The Doctl Authors All rights reserved.
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

	"github.com/digitalocean/doctl/do"
)

type ReservedIPv6 struct {
	ReservedIPv6s do.ReservedIPv6s
}

var _ Displayable = &ReservedIPv6{}

func (rip *ReservedIPv6) JSON(out io.Writer) error {
	return writeJSON(rip.ReservedIPv6s, out)
}

func (rip *ReservedIPv6) Cols() []string {
	return []string{
		"IP", "Region", "DropletID", "DropletName",
	}
}

func (rip *ReservedIPv6) ColMap() map[string]string {
	return map[string]string{
		"IP": "IP", "Region": "Region", "DropletID": "Droplet ID", "DropletName": "Droplet Name",
	}
}

func (rip *ReservedIPv6) KV() []map[string]any {
	out := make([]map[string]any, 0, len(rip.ReservedIPv6s))

	for _, f := range rip.ReservedIPv6s {
		var dropletID, dropletName string
		if f.Droplet != nil {
			dropletID = fmt.Sprintf("%d", f.Droplet.ID)
			dropletName = f.Droplet.Name
		}

		o := map[string]any{
			"IP":          f.IP,
			"Region":      f.RegionSlug,
			"DropletID":   dropletID,
			"DropletName": dropletName,
		}

		out = append(out, o)
	}

	return out
}

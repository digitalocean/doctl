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

	"github.com/digitalocean/doctl/do"
)

type ReservedIP struct {
	ReservedIPs do.ReservedIPs
}

var _ Displayable = &ReservedIP{}

func (rip *ReservedIP) JSON(out io.Writer) error {
	return writeJSON(rip.ReservedIPs, out)
}

func (rip *ReservedIP) Cols() []string {
	return []string{
		"IP", "Region", "DropletID", "DropletName", "ProjectID",
	}
}

func (rip *ReservedIP) ColMap() map[string]string {
	return map[string]string{
		"IP": "IP", "Region": "Region", "DropletID": "Droplet ID", "DropletName": "Droplet Name", "ProjectID": "Project ID",
	}
}

func (rip *ReservedIP) KV() []map[string]interface{} {
	out := make([]map[string]interface{}, 0, len(rip.ReservedIPs))

	for _, f := range rip.ReservedIPs {
		var dropletID, dropletName string
		if f.Droplet != nil {
			dropletID = fmt.Sprintf("%d", f.Droplet.ID)
			dropletName = f.Droplet.Name
		}

		o := map[string]interface{}{
			"IP": f.IP, "Region": f.Region.Slug,
			"DropletID": dropletID, "DropletName": dropletName,
			"ProjectID": f.ProjectID,
		}

		out = append(out, o)
	}

	return out
}

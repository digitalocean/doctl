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
	"strconv"
	"strings"

	"github.com/digitalocean/doctl/do"
)

type Snapshot struct {
	Snapshots do.Snapshots
}

var _ Displayable = &Snapshot{}

func (s *Snapshot) JSON(out io.Writer) error {
	return writeJSON(s.Snapshots, out)
}

func (s *Snapshot) Cols() []string {
	return []string{"ID", "Name", "CreatedAt", "Regions", "ResourceId",
		"ResourceType", "MinDiskSize", "Size", "Tags"}
}

func (s *Snapshot) ColMap() map[string]string {
	return map[string]string{
		"ID": "ID", "Name": "Name", "CreatedAt": "Created at", "Regions": "Regions",
		"ResourceId": "Resource ID", "ResourceType": "Resource Type", "MinDiskSize": "Min Disk Size", "Size": "Size", "Tags": "Tags"}
}

func (s *Snapshot) KV() []map[string]interface{} {
	out := make([]map[string]interface{}, 0, len(s.Snapshots))

	for _, ss := range s.Snapshots {
		o := map[string]interface{}{
			"ID": ss.ID, "Name": ss.Name, "ResourceId": ss.ResourceID,
			"ResourceType": ss.ResourceType, "Regions": ss.Regions, "MinDiskSize": ss.MinDiskSize,
			"Size": strconv.FormatFloat(ss.SizeGigaBytes, 'f', 2, 64) + " GiB", "CreatedAt": ss.Created,
			"Tags": strings.Join(ss.Tags, ","),
		}
		out = append(out, o)
	}

	return out
}

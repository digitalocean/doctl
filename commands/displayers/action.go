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

type Action struct {
	Actions do.Actions
}

var _ Displayable = &Action{}

func (a *Action) JSON(out io.Writer) error {
	return writeJSON(a.Actions, out)
}

func (a *Action) Cols() []string {
	return []string{
		"ID", "Status", "Type", "StartedAt", "CompletedAt", "ResourceID", "ResourceType", "Region",
	}
}

func (a *Action) ColMap() map[string]string {
	return map[string]string{
		"ID": "ID", "Status": "Status", "Type": "Type", "StartedAt": "Started At",
		"CompletedAt": "Completed At", "ResourceID": "Resource ID",
		"ResourceType": "Resource Type", "Region": "Region",
	}
}

func (a *Action) KV() []map[string]interface{} {
	out := make([]map[string]interface{}, 0, len(a.Actions))

	for _, x := range a.Actions {
		region := ""
		if x.Region != nil {
			region = x.Region.Slug
		}
		o := map[string]interface{}{
			"ID": x.ID, "Status": x.Status, "Type": x.Type,
			"StartedAt": x.StartedAt, "CompletedAt": x.CompletedAt,
			"ResourceID": x.ResourceID, "ResourceType": x.ResourceType,
			"Region": region,
		}
		out = append(out, o)
	}

	return out
}

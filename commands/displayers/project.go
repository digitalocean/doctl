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

type Project struct {
	Projects do.Projects
}

var _ Displayable = &Project{}

func (p *Project) JSON(out io.Writer) error {
	return writeJSON(p.Projects, out)
}

func (p *Project) Cols() []string {
	return []string{
		"ID",
		"OwnerUUID",
		"OwnerID",
		"Name",
		"Description",
		"Purpose",
		"Environment",
		"IsDefault",
		"CreatedAt",
		"UpdatedAt",
	}
}

func (p *Project) ColMap() map[string]string {
	return map[string]string{
		"ID":          "ID",
		"OwnerUUID":   "Owner UUID",
		"OwnerID":     "Owner ID",
		"Name":        "Name",
		"Description": "Description",
		"Purpose":     "Purpose",
		"Environment": "Environment",
		"IsDefault":   "Is Default?",
		"CreatedAt":   "Created At",
		"UpdatedAt":   "Updated At",
	}
}

func (p *Project) KV() []map[string]interface{} {
	out := make([]map[string]interface{}, 0, len(p.Projects))

	for _, pr := range p.Projects {
		o := map[string]interface{}{
			"ID":          pr.ID,
			"OwnerUUID":   pr.OwnerUUID,
			"OwnerID":     pr.OwnerID,
			"Name":        pr.Name,
			"Description": pr.Description,
			"Purpose":     pr.Purpose,
			"Environment": pr.Environment,
			"IsDefault":   pr.IsDefault,
			"CreatedAt":   pr.CreatedAt,
			"UpdatedAt":   pr.UpdatedAt,
		}
		out = append(out, o)
	}

	return out
}

type ProjectResource struct {
	ProjectResources do.ProjectResources
}

var _ Displayable = &ProjectResource{}

func (p *ProjectResource) JSON(out io.Writer) error {
	return writeJSON(p.ProjectResources, out)
}

func (p *ProjectResource) Cols() []string {
	return []string{
		"URN",
		"AssignedAt",
		"Status",
	}
}

func (p *ProjectResource) ColMap() map[string]string {
	return map[string]string{
		"URN":        "URN",
		"AssignedAt": "Assigned At",
		"Status":     "Status",
	}
}

func (p *ProjectResource) KV() []map[string]interface{} {
	out := make([]map[string]interface{}, 0, len(p.ProjectResources))

	for _, pr := range p.ProjectResources {
		assignedAt := pr.AssignedAt
		if assignedAt == "" {
			assignedAt = "N/A"
		}

		o := map[string]interface{}{
			"URN":        pr.URN,
			"AssignedAt": assignedAt,
			"Status":     pr.Status,
		}
		out = append(out, o)
	}

	return out
}

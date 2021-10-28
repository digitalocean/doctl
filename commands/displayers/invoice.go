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
	"time"

	"github.com/digitalocean/doctl/do"
)

type Invoice struct {
	*do.Invoice
}

var _ Displayable = &Invoice{}

func (i *Invoice) JSON(out io.Writer) error {
	return writeJSON(i.Invoice, out)
}

func (i *Invoice) Cols() []string {
	return []string{
		"ResourceID", "ResourceUUID", "Product", "Description", "GroupDescription", "Amount", "Duration", "DurationUnit", "StartTime", "EndTime", "ProjectName", "Category",
	}
}

func (i *Invoice) ColMap() map[string]string {
	return map[string]string{
		"ResourceID":       "Resource ID",
		"ResourceUUID":     "Resource UUID",
		"Product":          "Product",
		"Description":      "Description",
		"GroupDescription": "Group Description",
		"Amount":           "Amount",
		"Duration":         "Duration",
		"DurationUnit":     "Duration Unit",
		"StartTime":        "Start Time",
		"EndTime":          "End Time",
		"ProjectName":      "Project Name",
		"Category":         "Category",
	}
}

func (i *Invoice) KV() []map[string]interface{} {
	out := make([]map[string]interface{}, 0, len(i.Invoice.Invoice.InvoiceItems))
	for _, ii := range i.Invoice.Invoice.InvoiceItems {
		x := map[string]interface{}{
			"ResourceID":       ii.ResourceID,
			"ResourceUUID":     ii.ResourceUUID,
			"Product":          ii.Product,
			"Description":      ii.Description,
			"GroupDescription": ii.GroupDescription,
			"Amount":           ii.Amount,
			"Duration":         ii.Duration,
			"DurationUnit":     ii.DurationUnit,
			"StartTime":        ii.StartTime.Format(time.RFC3339),
			"EndTime":          ii.EndTime.Format(time.RFC3339),
			"ProjectName":      ii.ProjectName,
			"Category":         ii.Category,
		}
		out = append(out, x)
	}

	return out
}

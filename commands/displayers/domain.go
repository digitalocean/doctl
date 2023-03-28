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

type Domain struct {
	Domains do.Domains
}

var _ Displayable = &Domain{}

func (d *Domain) JSON(out io.Writer) error {
	return writeJSON(d.Domains, out)
}

func (d *Domain) Cols() []string {
	return []string{"Domain", "TTL"}
}

func (d *Domain) ColMap() map[string]string {
	return map[string]string{
		"Domain": "Domain", "TTL": "TTL",
	}
}

func (d *Domain) KV() []map[string]interface{} {
	out := make([]map[string]interface{}, 0, len(d.Domains))

	for _, do := range d.Domains {
		o := map[string]interface{}{
			"Domain": do.Name, "TTL": do.TTL,
		}
		out = append(out, o)
	}

	return out
}

type DomainRecord struct {
	DomainRecords do.DomainRecords
}

func (dr *DomainRecord) JSON(out io.Writer) error {
	return writeJSON(dr.DomainRecords, out)
}

func (dr *DomainRecord) Cols() []string {
	return []string{
		"ID", "Type", "Name", "Data", "Priority", "Port", "TTL", "Weight",
	}
}

func (dr *DomainRecord) ColMap() map[string]string {
	return map[string]string{
		"ID": "ID", "Type": "Type", "Name": "Name", "Data": "Data",
		"Priority": "Priority", "Port": "Port", "TTL": "TTL", "Weight": "Weight",
	}
}

func (dr *DomainRecord) KV() []map[string]interface{} {
	out := make([]map[string]interface{}, 0, len(dr.DomainRecords))

	for _, d := range dr.DomainRecords {
		o := map[string]interface{}{
			"ID": d.ID, "Type": d.Type, "Name": d.Name,
			"Data": d.Data, "Priority": d.Priority,
			"Port": d.Port, "TTL": d.TTL, "Weight": d.Weight,
		}
		out = append(out, o)
	}

	return out
}

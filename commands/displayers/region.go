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

type Region struct {
	Regions do.Regions
}

var _ Displayable = &Region{}

func (re *Region) JSON(out io.Writer) error {
	return writeJSON(re.Regions, out)
}

func (re *Region) Cols() []string {
	return []string{
		"Slug", "Name", "Available",
	}
}

func (re *Region) ColMap() map[string]string {
	return map[string]string{
		"Slug": "Slug", "Name": "Name", "Available": "Available",
	}
}

func (re *Region) KV() []map[string]interface{} {
	out := make([]map[string]interface{}, 0, len(re.Regions))

	for _, r := range re.Regions {
		o := map[string]interface{}{
			"Slug": r.Slug, "Name": r.Name, "Available": r.Available,
		}

		out = append(out, o)
	}

	return out
}

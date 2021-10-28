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

type Size struct {
	Sizes do.Sizes
}

var _ Displayable = &Size{}

func (si *Size) JSON(out io.Writer) error {
	return writeJSON(si.Sizes, out)
}

func (si *Size) Cols() []string {
	return []string{
		"Slug", "Memory", "VCPUs", "Disk", "PriceMonthly", "PriceHourly",
	}
}

func (si *Size) ColMap() map[string]string {
	return map[string]string{
		"Slug": "Slug", "Memory": "Memory", "VCPUs": "VCPUs",
		"Disk": "Disk", "PriceMonthly": "Price Monthly",
		"PriceHourly": "Price Hourly",
	}
}

func (si *Size) KV() []map[string]interface{} {
	out := make([]map[string]interface{}, 0, len(si.Sizes))

	for _, s := range si.Sizes {
		o := map[string]interface{}{
			"Slug": s.Slug, "Memory": s.Memory, "VCPUs": s.Vcpus,
			"Disk": s.Disk, "PriceMonthly": fmt.Sprintf("%0.2f", s.PriceMonthly),
			"PriceHourly": s.PriceHourly,
		}

		out = append(out, o)
	}

	return out
}

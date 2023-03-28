/*
Copyright 2020 The Doctl Authors All rights reserved.
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

// OneClick is the struct with the OneClickService on it.
type OneClick struct {
	OneClicks do.OneClicks
}

var _ Displayable = &OneClick{}

// JSON handles writing the json
func (oc *OneClick) JSON(out io.Writer) error {
	return writeJSON(oc.OneClicks, out)
}

// Cols are the columns returned in the json
func (oc *OneClick) Cols() []string {
	return []string{
		"SLUG",
		"TYPE",
	}
}

// ColMap maps the column names
func (oc *OneClick) ColMap() map[string]string {
	return map[string]string{
		"SLUG": "SLUG",
		"TYPE": "TYPE",
	}
}

// KV maps the values of a 1-click to an output
func (oc *OneClick) KV() []map[string]interface{} {
	out := make([]map[string]interface{}, 0, len(oc.OneClicks))

	for _, oneClick := range oc.OneClicks {
		o := map[string]interface{}{
			"SLUG": oneClick.Slug,
			"TYPE": oneClick.Type,
		}
		out = append(out, o)
	}

	return out
}

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

// Namespaces is the type of the displayer for namespaces list
type Namespaces struct {
	Info []do.OutputNamespace
}

var _ Displayable = &Namespaces{}

// JSON is the displayer JSON method specialized for namespaces list
func (i *Namespaces) JSON(out io.Writer) error {
	return writeJSON(i.Info, out)
}

// Cols is the displayer Cols method specialized for namespaces list
func (i *Namespaces) Cols() []string {
	return []string{
		"Label", "Region", "ID", "Host",
	}
}

// ColMap is the displayer ColMap method specialized for namespaces list
func (i *Namespaces) ColMap() map[string]string {
	return map[string]string{
		"Label":  "Label",
		"Region": "Region",
		"ID":     "Namespace ID",
		"Host":   "API Host",
	}
}

// KV is the displayer KV method specialized for namespaces list
func (i *Namespaces) KV() []map[string]interface{} {
	out := make([]map[string]interface{}, 0, len(i.Info))
	for _, ii := range i.Info {
		x := map[string]interface{}{
			"Label":  ii.Label,
			"Region": ii.Region,
			"ID":     ii.Namespace,
			"Host":   ii.APIHost,
		}
		out = append(out, x)
	}

	return out
}

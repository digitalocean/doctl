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

type Kernel struct {
	Kernels do.Kernels
}

var _ Displayable = &Kernel{}

func (ke *Kernel) JSON(out io.Writer) error {
	return writeJSON(ke.Kernels, out)
}

func (ke *Kernel) Cols() []string {
	return []string{
		"ID", "Name", "Version",
	}
}

func (ke *Kernel) ColMap() map[string]string {
	return map[string]string{
		"ID": "ID", "Name": "Name", "Version": "Version",
	}
}

func (ke *Kernel) KV() []map[string]interface{} {
	out := make([]map[string]interface{}, 0, len(ke.Kernels))

	for _, k := range ke.Kernels {
		o := map[string]interface{}{
			"ID": k.ID, "Name": k.Name, "Version": k.Version,
		}

		out = append(out, o)
	}

	return out
}

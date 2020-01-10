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

type Registry struct {
	Registries []do.Registry
}

var _ Displayable = &Registry{}

func (r *Registry) JSON(out io.Writer) error {
	return writeJSON(r.Registries, out)
}

func (r *Registry) Cols() []string {
	return []string{
		"Name",
		"Endpoint",
	}
}

func (r *Registry) ColMap() map[string]string {
	return map[string]string{
		"Name":     "Name",
		"Endpoint": "Endpoint",
	}
}

func (r *Registry) KV() []map[string]interface{} {
	out := []map[string]interface{}{}

	for _, reg := range r.Registries {
		m := map[string]interface{}{
			"Name":     reg.Name,
			"Endpoint": reg.Endpoint(),
		}

		out = append(out, m)
	}

	return out
}

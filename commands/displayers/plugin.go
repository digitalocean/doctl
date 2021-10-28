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

import "io"

type PlugDesc struct {
	Path string `json:"path"`
	Name string `json:"name"`
}

type Plugin struct {
	Plugins []PlugDesc
}

var _ Displayable = &Plugin{}

func (p *Plugin) JSON(out io.Writer) error {
	return writeJSON(p.Plugins, out)
}

func (p *Plugin) Cols() []string {
	return []string{
		"Name",
	}
}

func (p *Plugin) ColMap() map[string]string {
	return map[string]string{
		"Name": "Name",
	}
}

func (p *Plugin) KV() []map[string]interface{} {
	out := make([]map[string]interface{}, 0, len(p.Plugins))

	for _, plug := range p.Plugins {
		o := map[string]interface{}{
			"Name": plug.Name,
		}

		out = append(out, o)
	}

	return out
}

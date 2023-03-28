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
	"strings"
	"time"

	"github.com/apache/openwhisk-client-go/whisk"
)

// Functions is the type of the displayer for functions list
type Functions struct {
	Info []whisk.Action
}

var _ Displayable = &Functions{}

// JSON is the displayer JSON method specialized for functions list
func (i *Functions) JSON(out io.Writer) error {
	return writeJSON(i.Info, out)
}

// Cols is the displayer Cols method specialized for functions list
func (i *Functions) Cols() []string {
	return []string{
		"Update", "Version", "Runtime", "Function",
	}
}

// ColMap is the displayer ColMap method specialized for functions list
func (i *Functions) ColMap() map[string]string {
	return map[string]string{
		"Update":   "Latest Update",
		"Runtime":  "Runtime Kind",
		"Version":  "Latest Version",
		"Function": "Function Name",
	}
}

// KV is the displayer KV method specialized for functions list
func (i *Functions) KV() []map[string]interface{} {
	out := make([]map[string]interface{}, 0, len(i.Info))
	for _, ii := range i.Info {
		x := map[string]interface{}{
			"Update":   time.UnixMilli(ii.Updated).Format("01/02 03:04:05"),
			"Runtime":  findRuntime(ii.Annotations),
			"Version":  ii.Version,
			"Function": computeFunctionName(ii.Name, ii.Namespace),
		}
		out = append(out, x)
	}

	return out
}

// findRuntime finds the runtime string amongst the annotations of a function
func findRuntime(annots whisk.KeyValueArr) string {
	for i := range annots {
		if annots[i].Key == "exec" {
			return annots[i].Value.(string)
		}
	}
	return "<unknown>"
}

// computeFunctionName computes the effective name of a function from its simple name and the 'namespace' field
// (which actually encodes both namespace and package).
func computeFunctionName(simpleName string, namespace string) string {
	nsparts := strings.Split(namespace, "/")
	if len(nsparts) > 1 {
		return nsparts[1] + "/" + simpleName
	}
	return simpleName
}

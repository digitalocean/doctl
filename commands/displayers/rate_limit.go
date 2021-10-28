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

type RateLimit struct {
	*do.RateLimit
}

var _ Displayable = &RateLimit{}

func (rl *RateLimit) JSON(out io.Writer) error {
	return writeJSON(rl.Rate, out)
}

func (rl *RateLimit) Cols() []string {
	return []string{
		"Limit", "Remaining", "Reset",
	}
}

func (rl *RateLimit) ColMap() map[string]string {
	return map[string]string{
		"Limit": "Limit", "Remaining": "Remaining", "Reset": "Reset",
	}
}

func (rl *RateLimit) KV() []map[string]interface{} {
	x := map[string]interface{}{
		"Limit": rl.Limit, "Remaining": rl.Remaining, "Reset": rl.Reset,
	}

	return []map[string]interface{}{x}
}

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

type CDN struct {
	CDNs []do.CDN
}

var _ Displayable = &CDN{}

func (c *CDN) JSON(out io.Writer) error {
	return writeJSON(c.CDNs, out)
}

func (c *CDN) Cols() []string {
	return []string{
		"ID", "Origin", "Endpoint", "TTL", "CustomDomain", "CertificateID", "CreatedAt",
	}
}

func (c *CDN) ColMap() map[string]string {
	return map[string]string{
		"ID":            "ID",
		"Origin":        "Origin",
		"Endpoint":      "Endpoint",
		"TTL":           "TTL",
		"CustomDomain":  "CustomDomain",
		"CertificateID": "CertificateID",
		"CreatedAt":     "CreatedAt",
	}
}

func (c *CDN) KV() []map[string]interface{} {
	out := make([]map[string]interface{}, 0, len(c.CDNs))

	for _, cdn := range c.CDNs {
		m := map[string]interface{}{
			"ID":            cdn.ID,
			"Origin":        cdn.Origin,
			"Endpoint":      cdn.Endpoint,
			"TTL":           cdn.TTL,
			"CustomDomain":  cdn.CustomDomain,
			"CertificateID": cdn.CertificateID,
			"CreatedAt":     cdn.CreatedAt,
		}

		out = append(out, m)
	}

	return out
}

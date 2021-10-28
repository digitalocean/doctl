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

	"github.com/digitalocean/doctl/do"
)

type Certificate struct {
	Certificates do.Certificates
}

var _ Displayable = &Certificate{}

func (c *Certificate) JSON(out io.Writer) error {
	return writeJSON(c.Certificates, out)
}

func (c *Certificate) Cols() []string {
	return []string{
		"ID",
		"Name",
		"DNSNames",
		"SHA1Fingerprint",
		"NotAfter",
		"Created",
		"Type",
		"State",
	}
}

func (c *Certificate) ColMap() map[string]string {
	return map[string]string{
		"ID":              "ID",
		"Name":            "Name",
		"DNSNames":        "DNS Names",
		"SHA1Fingerprint": "SHA-1 Fingerprint",
		"NotAfter":        "Expiration Date",
		"Created":         "Created At",
		"Type":            "Type",
		"State":           "State",
	}
}

func (c *Certificate) KV() []map[string]interface{} {
	out := make([]map[string]interface{}, 0, len(c.Certificates))

	for _, c := range c.Certificates {
		o := map[string]interface{}{
			"ID":              c.ID,
			"Name":            c.Name,
			"DNSNames":        strings.Join(c.DNSNames, ","),
			"SHA1Fingerprint": c.SHA1Fingerprint,
			"NotAfter":        c.NotAfter,
			"Created":         c.Created,
			"Type":            c.Type,
			"State":           c.State,
		}
		out = append(out, o)
	}

	return out
}

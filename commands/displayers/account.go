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

type Account struct {
	*do.Account
}

var _ Displayable = &Account{}

func (a *Account) JSON(out io.Writer) error {
	return writeJSON(a.Account, out)
}

func (a *Account) Cols() []string {
	return []string{
		"Email", "DropletLimit", "EmailVerified", "UUID", "Status",
	}
}

func (a *Account) ColMap() map[string]string {
	return map[string]string{
		"Email": "Email", "DropletLimit": "Droplet Limit", "EmailVerified": "Email Verified",
		"UUID": "UUID", "Status": "Status",
	}
}

func (a *Account) KV() []map[string]interface{} {
	x := map[string]interface{}{
		"Email": a.Email, "DropletLimit": a.DropletLimit,
		"EmailVerified": a.EmailVerified, "UUID": a.UUID,
		"Status": a.Status,
	}

	return []map[string]interface{}{x}
}

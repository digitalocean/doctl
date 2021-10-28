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

// Key is used to display the SSH Key results from a `list` operation.
type Key struct {
	Keys do.SSHKeys
}

var _ Displayable = &Key{}

func (ke *Key) JSON(out io.Writer) error {
	return writeJSON(ke.Keys, out)
}

func (ke *Key) Cols() []string {
	return []string{
		"ID", "Name", "FingerPrint",
	}
}

func (ke *Key) ColMap() map[string]string {
	return map[string]string{
		"ID": "ID", "Name": "Name", "FingerPrint": "FingerPrint",
	}
}

func (ke *Key) KV() []map[string]interface{} {
	out := make([]map[string]interface{}, 0, len(ke.Keys))

	for _, k := range ke.Keys {
		o := map[string]interface{}{
			"ID": k.ID, "Name": k.Name, "FingerPrint": k.Fingerprint,
		}

		out = append(out, o)
	}

	return out
}

// KeyGet is used to display the SSH Key results from a `get` operation. This
// separate displayer is required in order to include the public key in this
// operation.
type KeyGet struct {
	Keys do.SSHKeys
}

var _ Displayable = &KeyGet{}

func (ke *KeyGet) JSON(out io.Writer) error {
	return writeJSON(ke.Keys, out)
}

func (ke *KeyGet) Cols() []string {
	return []string{
		"ID", "Name", "FingerPrint", "PublicKey",
	}
}

func (ke *KeyGet) ColMap() map[string]string {
	return map[string]string{
		"ID": "ID", "Name": "Name", "FingerPrint": "FingerPrint", "PublicKey": "Public Key",
	}
}

func (ke *KeyGet) KV() []map[string]interface{} {
	out := make([]map[string]interface{}, 0, len(ke.Keys))

	for _, k := range ke.Keys {
		o := map[string]interface{}{
			"ID": k.ID, "Name": k.Name, "FingerPrint": k.Fingerprint, "PublicKey": k.PublicKey,
		}

		out = append(out, o)
	}

	return out
}

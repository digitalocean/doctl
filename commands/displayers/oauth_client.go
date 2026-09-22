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
)

// OAuthClient renders the OAuth application doctl registered for itself. It
// never carries the registration access token: that credential manages the
// registration and is kept out of command output.
type OAuthClient struct {
	ClientID      string
	Issuer        string
	RedirectURIs  []string
	RegisteredAt  time.Time
	ManagementURL string
}

var _ Displayable = &OAuthClient{}

func (c *OAuthClient) JSON(out io.Writer) error {
	return writeJSON(struct {
		ClientID      string   `json:"client_id"`
		Issuer        string   `json:"issuer"`
		RedirectURIs  []string `json:"redirect_uris,omitempty"`
		RegisteredAt  string   `json:"registered_at,omitempty"`
		ManagementURL string   `json:"registration_client_uri,omitempty"`
	}{
		ClientID:      c.ClientID,
		Issuer:        c.Issuer,
		RedirectURIs:  c.RedirectURIs,
		RegisteredAt:  c.registeredAt(),
		ManagementURL: c.ManagementURL,
	}, out)
}

func (c *OAuthClient) Cols() []string {
	return []string{"ClientID", "Issuer", "RedirectURIs", "RegisteredAt", "ManagementURL"}
}

func (c *OAuthClient) ColMap() map[string]string {
	return map[string]string{
		"ClientID":      "Client ID",
		"Issuer":        "Issuer",
		"RedirectURIs":  "Redirect URIs",
		"RegisteredAt":  "Registered At",
		"ManagementURL": "Management URL",
	}
}

func (c *OAuthClient) KV() []map[string]any {
	return []map[string]any{{
		"ClientID":      c.ClientID,
		"Issuer":        c.Issuer,
		"RedirectURIs":  c.redirectURIs(),
		"RegisteredAt":  c.registeredAt(),
		"ManagementURL": c.ManagementURL,
	}}
}

// redirectURIs reports unknown rather than blank for a registration created
// before doctl recorded them, so an empty column is not mistaken for a client
// that accepts no redirects.
func (c *OAuthClient) redirectURIs() string {
	if len(c.RedirectURIs) == 0 {
		return "unknown"
	}
	return strings.Join(c.RedirectURIs, ", ")
}

func (c *OAuthClient) registeredAt() string {
	if c.RegisteredAt.IsZero() {
		return ""
	}
	return c.RegisteredAt.UTC().Format(time.RFC3339)
}

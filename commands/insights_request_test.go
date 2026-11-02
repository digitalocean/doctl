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

package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadInsightsRequest(t *testing.T) {
	path := filepath.Join(t.TempDir(), "channel.yaml")
	require.NoError(t, os.WriteFile(path, []byte(`
name: Operations
email:
  to: ops@example.com
`), 0o600))

	request, err := readInsightsRequest[godo.NotificationChannelRequest](strings.NewReader(""), path)
	require.NoError(t, err)
	assert.Equal(t, &godo.NotificationChannelRequest{
		Name:  "Operations",
		Email: &godo.EmailNotificationConfig{To: "ops@example.com"},
	}, request)
}

func TestReadInsightsRequestFromStdin(t *testing.T) {
	request, err := readInsightsRequest[godo.NotificationChannelRequest](strings.NewReader(
		`{"name":"Operations","email":{"to":"ops@example.com"}}`,
	), "-")
	require.NoError(t, err)
	assert.Equal(t, "Operations", request.Name)
	assert.Equal(t, "ops@example.com", request.Email.To)
}

func TestReadInsightsRequestRejectsUnknownFields(t *testing.T) {
	_, err := readInsightsRequest[godo.NotificationChannelRequest](strings.NewReader(
		`{"name":"Operations","unknown":true}`,
	), "-")
	require.Error(t, err)
	assert.Contains(t, err.Error(), `unknown field "unknown"`)
}

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
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/digitalocean/doctl/internal/oauth"
)

func TestRunAuthClientShow(t *testing.T) {
	server := newTestAuthorizationServer(t, func() int { return 1 })
	defer server.Close()

	config, _ := newOAuthTestCmdConfig(t, server.URL)
	out := &bytes.Buffer{}
	config.Out = out

	stubOAuthLogin(t, func(context.Context, oauth.LoginOptions) (*oauth.Token, error) {
		return &oauth.Token{AccessToken: "doo_v1_access"}, nil
	})
	require.NoError(t, RunAuthLogin(config))

	out.Reset()
	require.NoError(t, RunAuthClientShow(config))

	rendered := out.String()
	assert.Contains(t, rendered, "client-1")
	assert.Contains(t, rendered, server.URL)
	assert.Contains(t, rendered, "http://127.0.0.1/callback")
	assert.Contains(t, rendered, testClientIssuedAt.Format(time.RFC3339))
	assert.NotContains(t, rendered, "registration-token-1", "the credential that manages the registration is not display output")
}

func TestRunAuthClientShowWithoutARegistration(t *testing.T) {
	config, _ := newOAuthTestCmdConfig(t, "https://cloud.digitalocean.com")

	err := RunAuthClientShow(config)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "doctl auth login")
}

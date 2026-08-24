/*
Copyright 2026 The Doctl Authors All rights reserved.
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
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHTTPAnthropicClient_CheckAPIKey(t *testing.T) {
	t.Run("accepts a valid key", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/v1/models", r.URL.Path)
			assert.Equal(t, "sk-ant-good", r.Header.Get("x-api-key"))
			assert.Equal(t, anthropicAPIVersion, r.Header.Get("anthropic-version"))
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"data":[]}`))
		}))
		defer srv.Close()

		c := &httpAnthropicClient{httpClient: srv.Client(), baseURL: srv.URL}
		require.NoError(t, c.checkAPIKey(context.Background(), "sk-ant-good"))
	})

	t.Run("rejects an invalid key with a clear message", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"type":"error","error":{"type":"authentication_error","message":"invalid x-api-key"}}`))
		}))
		defer srv.Close()

		c := &httpAnthropicClient{httpClient: srv.Client(), baseURL: srv.URL}
		err := c.checkAPIKey(context.Background(), "sk-ant-bogus")
		require.Error(t, err)
		assert.Contains(t, err.Error(), anthropicAPIKeyEnv)
		assert.Contains(t, err.Error(), "rejected by Anthropic")
		assert.Contains(t, err.Error(), "invalid x-api-key")
	})

	t.Run("surfaces an unexpected status", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`boom`))
		}))
		defer srv.Close()

		c := &httpAnthropicClient{httpClient: srv.Client(), baseURL: srv.URL}
		err := c.checkAPIKey(context.Background(), "sk-ant-good")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "HTTP 500")
	})
}

func TestResolveAnthropicBaseURL(t *testing.T) {
	t.Run("defaults to the public API", func(t *testing.T) {
		t.Setenv(anthropicBaseURLEnv, "")
		assert.Equal(t, defaultAnthropicAPIBase, resolveAnthropicBaseURL())
	})

	t.Run("honors an override, trimming trailing slashes", func(t *testing.T) {
		t.Setenv(anthropicBaseURLEnv, "https://proxy.example.com/anthropic/")
		assert.Equal(t, "https://proxy.example.com/anthropic", resolveAnthropicBaseURL())
	})
}

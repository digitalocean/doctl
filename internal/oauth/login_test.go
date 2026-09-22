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

package oauth

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLogin(t *testing.T) {
	var challenge, verifier string

	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, r.ParseForm())
		verifier = r.FormValue("code_verifier")
		assert.Equal(t, "auth-code", r.FormValue("code"))

		writeJSON(t, w, http.StatusOK, map[string]any{
			"access_token":  "doo_v1_access",
			"refresh_token": "dor_v1_refresh",
			"expires_in":    3600,
		})
	}))
	defer tokenServer.Close()

	var authorizedRedirectURI string
	// Stand in for the browser: read the authorization request, then call the
	// redirect URI the way the authorization server would.
	openURL := func(authURL string) error {
		parsed, err := url.Parse(authURL)
		require.NoError(t, err)

		query := parsed.Query()
		assert.Equal(t, "client-id", query.Get("client_id"))
		assert.Equal(t, responseTypeCode, query.Get("response_type"))
		assert.Equal(t, CodeChallengeMethodS256, query.Get("code_challenge_method"))
		assert.Equal(t, "read write", query.Get("scope"))
		assert.NotEmpty(t, query.Get("state"))
		assert.Empty(t, query.Get("resource"), "resource indicators are not requested")

		challenge = query.Get("code_challenge")
		authorizedRedirectURI = query.Get("redirect_uri")

		return followRedirect(t, query.Get("redirect_uri"), url.Values{
			"code":  {"auth-code"},
			"state": {query.Get("state")},
		})
	}

	token, err := Login(context.Background(), LoginOptions{
		Metadata:   &ServerMetadata{AuthorizationEndpoint: "https://cloud.example.com/v1/oauth/authorize", TokenEndpoint: tokenServer.URL},
		ClientID:   "client-id",
		Scopes:     []string{"read", "write"},
		HTTPClient: tokenServer.Client(),
		openURL:    openURL,
	})
	require.NoError(t, err)

	assert.Equal(t, "doo_v1_access", token.AccessToken)
	assert.Equal(t, "dor_v1_refresh", token.RefreshToken)
	assert.Equal(t, challenge, codeChallengeS256(verifier), "the verifier must hash to the challenge sent to the authorization server")
	assert.Regexp(t, `^http://127\.0\.0\.1:\d+/callback$`, authorizedRedirectURI)
}

func TestLoginReportsAuthorizationErrors(t *testing.T) {
	tests := []struct {
		name          string
		params        func(state string) url.Values
		expectedError string
		invalidClient bool
	}{
		{
			name: "access denied",
			params: func(state string) url.Values {
				return url.Values{
					"error":             {"access_denied"},
					"error_description": {"the user declined the request"},
					"state":             {state},
				}
			},
			expectedError: "the user declined the request",
		},
		{
			name: "unknown client",
			params: func(state string) url.Values {
				return url.Values{"error": {"invalid_client"}, "state": {state}}
			},
			expectedError: "invalid_client",
			invalidClient: true,
		},
		{
			name: "mismatched state",
			params: func(string) url.Values {
				return url.Values{"code": {"auth-code"}, "state": {"not-the-state"}}
			},
			expectedError: "state did not match",
		},
		{
			name: "missing code",
			params: func(state string) url.Values {
				return url.Values{"state": {state}}
			},
			expectedError: "did not include an authorization code",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			openURL := func(authURL string) error {
				parsed, err := url.Parse(authURL)
				require.NoError(t, err)

				query := parsed.Query()
				return followRedirect(t, query.Get("redirect_uri"), test.params(query.Get("state")))
			}

			_, err := Login(context.Background(), LoginOptions{
				Metadata:   &ServerMetadata{AuthorizationEndpoint: "https://cloud.example.com/v1/oauth/authorize", TokenEndpoint: "https://cloud.example.com/v1/oauth/token"},
				ClientID:   "client-id",
				HTTPClient: failingClient(t),
				openURL:    openURL,
			})

			require.Error(t, err)
			assert.Contains(t, err.Error(), test.expectedError)
			if test.invalidClient {
				assert.ErrorIs(t, err, ErrInvalidClient)
			} else {
				assert.NotErrorIs(t, err, ErrInvalidClient)
			}
		})
	}
}

func TestLoginTimesOut(t *testing.T) {
	_, err := Login(context.Background(), LoginOptions{
		Metadata: &ServerMetadata{AuthorizationEndpoint: "https://cloud.example.com/v1/oauth/authorize"},
		ClientID: "client-id",
		Timeout:  50 * time.Millisecond,
		openURL:  func(string) error { return nil },
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "timed out")
}

func TestLoginRequiresClientID(t *testing.T) {
	_, err := Login(context.Background(), LoginOptions{Metadata: &ServerMetadata{}})

	assert.Error(t, err)
}

func TestLoginPrintsAuthorizationURL(t *testing.T) {
	var printed string

	_, err := Login(context.Background(), LoginOptions{
		Metadata:           &ServerMetadata{AuthorizationEndpoint: "https://cloud.example.com/v1/oauth/authorize?foo=bar"},
		ClientID:           "client-id",
		Timeout:            50 * time.Millisecond,
		NoBrowser:          true,
		OnAuthorizationURL: func(authURL string) { printed = authURL },
		openURL: func(string) error {
			return errors.New("the browser must not be opened with --no-browser")
		},
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "timed out")
	assert.Contains(t, printed, "https://cloud.example.com/v1/oauth/authorize?foo=bar&client_id=client-id")
}

func TestLoginOmitsAnEmptyScope(t *testing.T) {
	var printed string

	_, err := Login(context.Background(), LoginOptions{
		Metadata:           &ServerMetadata{AuthorizationEndpoint: "https://cloud.example.com/v1/oauth/authorize"},
		ClientID:           "client-id",
		Timeout:            50 * time.Millisecond,
		OnAuthorizationURL: func(authURL string) { printed = authURL },
		openURL:            func(string) error { return nil },
	})
	require.Error(t, err)

	parsed, err := url.Parse(printed)
	require.NoError(t, err)

	// Sending no scope leaves the granted permissions up to the authorization
	// screen rather than having doctl choose them.
	assert.False(t, parsed.Query().Has("scope"))
}

func TestCodeVerifier(t *testing.T) {
	verifier, err := generateCodeVerifier()
	require.NoError(t, err)

	// RFC 7636 section 4.1 allows 43 to 128 unreserved characters.
	assert.Len(t, verifier, 43)
	assert.Regexp(t, `^[A-Za-z0-9\-._~]+$`, verifier)

	other, err := generateCodeVerifier()
	require.NoError(t, err)
	assert.NotEqual(t, verifier, other)

	// The example verifier and challenge from RFC 7636 appendix B.
	assert.Equal(t,
		"E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM",
		codeChallengeS256("dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"),
	)
}

// followRedirect simulates the browser being redirected back to doctl's local
// callback server.
func followRedirect(t *testing.T, redirectURI string, params url.Values) error {
	t.Helper()

	resp, err := http.Get(redirectURI + "?" + params.Encode())
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

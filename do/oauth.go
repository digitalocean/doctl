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

package do

import (
	"context"
	"net/http"

	"github.com/digitalocean/godo"
)

const (
	oauthBaseURL  = "https://cloud.digitalocean.com"
	tokenInfoPath = "/v1/oauth/token/info"
)

// OAuthTokenInfo contains information about an OAuth token
type OAuthTokenInfo struct {
	ResourceOwnerID  int              `json:"resource_owner_id"`
	Scopes           []string         `json:"scopes"`
	ExpiresInSeconds int              `json:"expires_in_seconds"`
	Application      OAuthApplication `json:"application"`
	CreatedAt        int              `json:"created_at"`
}

// OAuthApplication contains info about an OAuth application
type OAuthApplication struct {
	UID string `json:"uid"`
}

// OAuthService is an interface for interacting with DigitalOcean's account api.
type OAuthService interface {
	TokenInfo(string) (*OAuthTokenInfo, error)
}

type oauthService struct {
	client *godo.Client
	server string
}

var _ OAuthService = &oauthService{}

// NewOAuthService builds an OAuthService instance.
func NewOAuthService(godoClient *godo.Client) OAuthService {
	return &oauthService{
		client: godoClient,
	}
}

func (oa *oauthService) TokenInfo(server string) (*OAuthTokenInfo, error) {
	tokenInfoURI := oauthBaseURL + tokenInfoPath
	if server != "" {
		tokenInfoURI = server + tokenInfoPath
	}

	ctx := context.TODO()
	req, err := oa.client.NewRequest(ctx, http.MethodGet, tokenInfoURI, nil)
	if err != nil {
		return nil, err
	}

	info := new(OAuthTokenInfo)
	_, err = oa.client.Do(ctx, req, info)
	if err != nil {
		return nil, err
	}

	return info, nil
}

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
	"github.com/digitalocean/godo"
)

// Account is a wrapper for godo.Account.
type Account struct {
	*godo.Account
}

// RateLimit is a wrapper for godo.Rate.
type RateLimit struct {
	*godo.Rate
}

// AccountService is an interface for interacting with DigitalOcean's account api.
type AccountService interface {
	Get() (*Account, error)
	RateLimit() (*RateLimit, error)
}

type accountService struct {
	client *godo.Client
}

var _ AccountService = &accountService{}

// NewAccountService builds an AccountService instance.
func NewAccountService(godoClient *godo.Client) AccountService {
	return &accountService{
		client: godoClient,
	}
}

func (as *accountService) Get() (*Account, error) {
	godoAccount, _, err := as.client.Account.Get(context.TODO())
	if err != nil {
		return nil, err
	}

	account := &Account{Account: godoAccount}
	return account, nil
}

func (as *accountService) RateLimit() (*RateLimit, error) {
	_, resp, err := as.client.Account.Get(context.TODO())
	if err != nil {
		return nil, err
	}

	rateLimit := &RateLimit{Rate: &resp.Rate}
	return rateLimit, nil
}

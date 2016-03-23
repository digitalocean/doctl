
/*
Copyright 2016 The Doctl Authors All rights reserved.
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
	"testing"

	"github.com/bryanl/godomock"
	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"
)

func TestAccountServiceGet(t *testing.T) {

	gAccountSvc := &godomock.MockAccountService{}

	gAccount := &godo.Account{UUID: "uuid"}
	gAccountSvc.On("Get").Return(gAccount, nil, nil)

	client := &godo.Client{
		Account: gAccountSvc,
	}
	as := NewAccountService(client)

	account, err := as.Get()
	assert.NoError(t, err)
	assert.Equal(t, "uuid", account.UUID)
}

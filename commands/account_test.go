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
	"testing"

	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"
)

var testAccount = &do.Account{
	Account: &godo.Account{
		DropletLimit:  10,
		Email:         "user@example.com",
		UUID:          "1234",
		EmailVerified: true,
	},
}

func TestAccountCommand(t *testing.T) {
	acctCmd := Account()
	assert.NotNil(t, acctCmd)
	assertCommandNames(t, acctCmd, "get", "ratelimit")
}

func TestAccountGet(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.account.EXPECT().Get().Return(testAccount, nil)

		err := RunAccountGet(config)
		assert.NoError(t, err)
	})
}

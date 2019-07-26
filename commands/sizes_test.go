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

var (
	testSize     = do.Size{Size: &godo.Size{Slug: "small"}}
	testSizeList = do.Sizes{testSize}
)

func TestSizeCommand(t *testing.T) {
	cmd := Size()
	assert.NotNil(t, cmd)
	assertCommandNames(t, cmd, "list")
}

func TestSizesList(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.sizes.EXPECT().List().Return(testSizeList, nil)

		err := RunSizeList(config)
		assert.NoError(t, err)
	})
}

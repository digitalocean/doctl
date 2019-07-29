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
	testRegion     = do.Region{Region: &godo.Region{Slug: "dev0"}}
	testRegionList = do.Regions{testRegion}
)

func TestRegionCommand(t *testing.T) {
	cmd := Region()
	assert.NotNil(t, cmd)
	assertCommandNames(t, cmd, "list")
}

func TestRegionsList(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.regions.EXPECT().List().Return(testRegionList, nil)

		err := RunRegionList(config)
		assert.NoError(t, err)
	})
}

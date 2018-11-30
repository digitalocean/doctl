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

package install

import (
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func TestValidate(t *testing.T) {
	appFS := &afero.MemMapFs{}

	appFS.Mkdir("test", 0755)
	afero.WriteFile(appFS, "test/dl", []byte("dl"), 0644)
	afero.WriteFile(appFS, "test/dl.sha256", []byte("2ca69efd4ea5af91a637f19ba0bab8b081d2c03773c4a72fcbf8817c856b33ef  /test/dl.sha256"), 0644)

	dl, err := appFS.Open("test/dl")
	assert.NoError(t, err)

	cs, err := appFS.Open("test/dl.sha256")
	assert.NoError(t, err)

	err = Validate(dl, cs)
	assert.NoError(t, err)
}

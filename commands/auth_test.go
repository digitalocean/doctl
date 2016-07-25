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

package commands

import (
	"io"
	"io/ioutil"
	"testing"

	"github.com/digitalocean/doctl/do"
	"github.com/stretchr/testify/assert"
)

func TestAuthCommand(t *testing.T) {
	cmd := Auth()
	assert.NotNil(t, cmd)
	assertCommandNames(t, cmd, "init")
}

func TestAuthInit(t *testing.T) {
	rtf := retrieveUserTokenFunc
	cfw := cfgFileWriter
	defer func() {
		retrieveUserTokenFunc = rtf
		cfgFileWriter = cfw
	}()

	retrieveUserTokenFunc = func() (string, error) {
		return "valid-token", nil
	}

	cfgFileWriter = func() (io.WriteCloser, error) { return &nopWriteCloser{Writer: ioutil.Discard}, nil }

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.account.On("Get").Return(&do.Account{}, nil)

		err := RunAuthInit(config)
		assert.NoError(t, err)
	})
}

type nopWriteCloser struct {
	io.Writer
}

var _ io.WriteCloser = (*nopWriteCloser)(nil)

func (d *nopWriteCloser) Close() error {
	return nil
}

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

package doctl

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v2"

	"github.com/stretchr/testify/assert"
)

func TestConfigSetCreatesFileIfNotFound(t *testing.T) {
	dir, err := ioutil.TempDir("", "doit")
	assert.NoError(t, err)

	thepath := filepath.Join(dir, configFile)
	assert.NoError(t, err)

	defer func() {
		os.RemoveAll(dir)
	}()

	cf := &ConfigFile{
		location: thepath,
	}

	err = cf.Set("foo", "bar")
	assert.NoError(t, err)

	_, err = os.Stat(thepath)
	assert.NoError(t, err)
}

func TestConfigSet(t *testing.T) {
	dir, err := ioutil.TempDir("", "doit")
	assert.NoError(t, err)

	thepath := filepath.Join(dir, configFile)
	assert.NoError(t, err)

	defer func() {
		os.RemoveAll(dir)
	}()

	cf := &ConfigFile{
		location: thepath,
	}

	err = cf.Set("foo", "bar")
	assert.NoError(t, err)

	r, err := cf.Open()
	assert.NoError(t, err)

	b, err := ioutil.ReadAll(r)
	assert.NoError(t, err)

	var config map[string]interface{}
	err = yaml.Unmarshal(b, &config)
	assert.NoError(t, err)

	assert.Equal(t, "bar", config["foo"])
}

func TestNewConfigFile(t *testing.T) {
	cf, err := NewConfigFile()
	assert.NoError(t, err)

	assert.NotEmpty(t, cf.location)
}

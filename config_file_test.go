package doit

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

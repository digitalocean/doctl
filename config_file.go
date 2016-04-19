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
	"io"
	"io/ioutil"
	"os"
	"os/user"
	"path/filepath"

	"gopkg.in/yaml.v2"
)

const (
	configFile = ".doctlcfg"
)

// ConfigFile is a doit config file.
type ConfigFile struct {
	location string
}

// NewConfigFile creates an instance of ConfigFile.
func NewConfigFile() (*ConfigFile, error) {
	usr, err := user.Current()
	if err != nil {
		return nil, err
	}

	location := filepath.Join(usr.HomeDir, configFile)

	return &ConfigFile{
		location: location,
	}, nil
}

// Set sets a ConfigFile key to a value. The value should be something
// that serializes to a valid YAML value.
func (cf *ConfigFile) Set(key string, val interface{}) error {
	c, err := cf.Open()
	if err != nil {
		switch err.(type) {
		case *os.PathError:
			err := cf.createConfigFile()
			if err != nil {
				return err
			}

			c, _ = cf.Open()
		default:
			return err
		}

	}

	b, err := ioutil.ReadAll(c)
	if err != nil {
		return err
	}

	var m map[string]interface{}
	err = yaml.Unmarshal(b, &m)
	if err != nil {
		return err
	}

	if m == nil {
		m = map[string]interface{}{}
	}

	m[key] = val

	out, err := yaml.Marshal(m)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(cf.location, out, 0600)
}

// Open opens a ConfigFile.
func (cf *ConfigFile) Open() (io.Reader, error) {
	_, err := os.Stat(cf.location)
	if err != nil {
		return nil, err
	}

	return os.Open(cf.location)
}

func (cf *ConfigFile) createConfigFile() error {
	f, err := os.Create(cf.location)
	if err != nil {
		return err
	}
	return f.Close()
}

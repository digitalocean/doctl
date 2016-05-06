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

package config

import (
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v2"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

const (
	pathSeparator = "."
)

// Load loads configuration from the configuration file.
func Load(cfgFile string) (Config, error) {
	if cfgFile == "" {
		cfgFile = filepath.Join(homeDir(), ".doctlcfg")
	}

	b, err := ioutil.ReadFile(cfgFile)
	if err != nil {
		return nil, err
	}

	buf := bytes.NewBuffer(b)
	return New(buf)
}

// Save saves the current configuration.
func Save(cfgFile string, c Config) error {
	f, err := os.Create(cfgFile)
	if err != nil {
		return err
	}

	err = c.Save(f)
	if err != nil {
		return err
	}

	return f.Close()
}

// Config is the configuration dictionary for doctl.
type Config interface {
	Get(string) interface{}
	Set(string, interface{})
	Delete(string) error
	List() map[string]interface{}
	Save(io.Writer) error

	BindFlag(key string, flag *pflag.Flag) error
	BindEnv(key, envVar string)
}

type config struct {
	v           *viper.Viper
	deletedKeys map[string]bool
}

var _ Config = (*config)(nil)

// New creates an instance of Config.
func New(r io.Reader) (Config, error) {
	v := viper.New()
	v.AutomaticEnv()
	v.SetEnvPrefix("DIGITALOCEAN")
	v.SetConfigType("yaml")
	v.SetDefault("output", "text")

	if err := v.ReadConfig(r); err != nil {
		return nil, err
	}

	return &config{
		v:           v,
		deletedKeys: map[string]bool{},
	}, nil
}

func (c *config) Get(key string) interface{} {
	return c.v.Get(key)
}

func (c *config) Set(key string, value interface{}) {
	c.v.Set(key, value)
}

func (c *config) Delete(key string) error {
	c.deletedKeys[key] = true
	return nil
}

func (c *config) List() map[string]interface{} {
	path := []string{}
	return c.dfs(path, c.v.AllKeys(), c.v.AllSettings())
}

func (c *config) dfs(path, keys []string, m map[string]interface{}) map[string]interface{} {
	out := map[string]interface{}{}
	for _, k := range keys {
		switch t := m[k].(type) {
		case map[interface{}]interface{}:
			path = append(path, k)
			pathStr := strings.Join(path, ".")
			if !c.deletedKeys[pathStr] {
				keys = []string{}

				m := map[string]interface{}{}
				for subk, subv := range t {
					keys = append(keys, subk.(string))
					m[subk.(string)] = subv
				}

				for k, v := range c.dfs(path, keys, m) {
					out[k] = v
				}
			}

		default:
			finalPath := append(path, k)
			pathStr := strings.Join(finalPath, ".")
			if !c.deletedKeys[pathStr] {
				out[pathStr] = m[k]
			}
		}
	}

	return out
}

func (c *config) Save(w io.Writer) error {
	b, err := yaml.Marshal(c.v.AllSettings())
	if err != nil {
		return err
	}

	_, err = w.Write(b)
	return err
}

func (c *config) BindFlag(key string, flag *pflag.Flag) error {
	return c.v.BindPFlag(key, flag)
}

func (c *config) BindEnv(key, envVar string) {
	c.v.BindEnv(key, envVar)
}

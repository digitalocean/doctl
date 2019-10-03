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

package internal

import (
	"fmt"

	"github.com/digitalocean/doctl"
	sshRunner "github.com/digitalocean/doctl/pkg/runner"
	"github.com/digitalocean/doctl/pkg/runner/mocks"
	"github.com/digitalocean/doctl/pkg/ssh"
	"github.com/digitalocean/godo"
	"github.com/spf13/viper"
)

// TestConfig is an implementation of Config for testing.
type TestConfig struct {
	SSHFn    func(user, host, keyPath string, port int, opts ssh.Options) sshRunner.Runner
	v        *viper.Viper
	IsSetMap map[string]bool
}

var _ doctl.Config = &TestConfig{}

// NewTestConfig creates a new, ready-to-use instance of a TestConfig.
func NewTestConfig() *TestConfig {
	return &TestConfig{
		SSHFn: func(u, h, kp string, p int, opts ssh.Options) sshRunner.Runner {
			return &runner.MockRunner{}
		},
		v:        viper.New(),
		IsSetMap: make(map[string]bool),
	}
}

// GetGodoClient mocks a GetGodoClient call. The returned godo client will
// be nil.
func (c *TestConfig) GetGodoClient(trace bool, accessToken string) (*godo.Client, error) {
	return &godo.Client{}, nil
}

// SSH returns a mock SSH runner.
func (c *TestConfig) SSH(user, host, keyPath string, port int, opts ssh.Options) sshRunner.Runner {
	return c.SSHFn(user, host, keyPath, port, opts)
}

// Set sets a config key.
func (c *TestConfig) Set(ns, key string, val interface{}) {
	nskey := fmt.Sprintf("%s-%s", ns, key)
	c.v.Set(nskey, val)
	c.IsSetMap[key] = true
}

// IsSet returns true if the given key is set on the config.
func (c *TestConfig) IsSet(key string) bool {
	return c.IsSetMap[key]
}

// GetString returns the string value for the key in the given namespace. Because
// this is a mock implementation, and error will never be returned.
func (c *TestConfig) GetString(ns, key string) (string, error) {
	nskey := fmt.Sprintf("%s-%s", ns, key)
	return c.v.GetString(nskey), nil
}

// GetInt returns the int value for the key in the given namespace. Because
// this is a mock implementation, and error will never be returned.
func (c *TestConfig) GetInt(ns, key string) (int, error) {
	nskey := fmt.Sprintf("%s-%s", ns, key)
	return c.v.GetInt(nskey), nil
}

// GetIntPtr returns the int value for the key in the given namespace. Because
// this is a mock implementation, and error will never be returned.
func (c *TestConfig) GetIntPtr(ns, key string) (*int, error) {
	nskey := fmt.Sprintf("%s-%s", ns, key)
	if !c.v.IsSet(nskey) {
		return nil, nil
	}
	val := c.v.GetInt(nskey)
	return &val, nil
}

// GetStringSlice returns the string slice value for the key in the given
// namespace. Because this is a mock implementation, and error will never be
// returned.
func (c *TestConfig) GetStringSlice(ns, key string) ([]string, error) {
	nskey := fmt.Sprintf("%s-%s", ns, key)
	return c.v.GetStringSlice(nskey), nil
}

// GetBool returns the bool value for the key in the given namespace. Because
// this is a mock implementation, and error will never be returned.
func (c *TestConfig) GetBool(ns, key string) (bool, error) {
	nskey := fmt.Sprintf("%s-%s", ns, key)
	return c.v.GetBool(nskey), nil
}

// GetBoolPtr returns the bool value for the key in the given namespace. Because
// this is a mock implementation, and error will never be returned.
func (c *TestConfig) GetBoolPtr(ns, key string) (*bool, error) {
	nskey := fmt.Sprintf("%s-%s", ns, key)
	if !c.v.IsSet(nskey) {
		return nil, nil
	}
	val := c.v.GetBool(nskey)
	return &val, nil
}


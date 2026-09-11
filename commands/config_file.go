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
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/digitalocean/doctl"
	"github.com/spf13/viper"
	yaml "gopkg.in/yaml.v2"
)

const authContextsKey = "auth-contexts"

// cfgFileReader reads the raw configuration file. It is a variable so that
// tests can supply a document rather than read the config of whoever is
// running them.
var cfgFileReader = defaultConfigFileReader

// configFile is the configuration as it exists on disk.
//
// doctl merges built-in defaults, this file, DIGITALOCEAN_* variables, and
// flags into one view, and viper deliberately forgets which layer each value
// came from. That view is the right thing to read and the wrong thing to write
// back: saving it persists whatever the environment happened to hold, which is
// how a token exported for one CI job became a durable plaintext file.
//
// Editing the file directly limits a write to what the command was asked to
// change and returns every other key, including ones this version knows
// nothing about, as it found them. Reading the YAML also keeps context names
// verbatim: viper lowercases keys and splits them on ".", which is what
// mangled dotted contexts in https://github.com/digitalocean/doctl/issues/996.
type configFile struct {
	doc map[string]any
}

// loadConfigFile reads the active configuration file. A file that does not
// exist yet is not an error; it loads as an empty document, which is what
// `auth init` on a new machine needs.
func loadConfigFile() (*configFile, error) {
	b, err := cfgFileReader()
	if err != nil {
		return nil, fmt.Errorf("Unable to read configuration: %w", err)
	}

	doc := map[string]any{}
	if err := yaml.Unmarshal(b, &doc); err != nil {
		return nil, fmt.Errorf("Unable to parse configuration: %w", err)
	}

	// An empty or explicitly null document unmarshals to a nil map.
	if doc == nil {
		doc = map[string]any{}
	}

	return &configFile{doc: doc}, nil
}

// setToken stores the token for a context, replacing any token already there.
func (c *configFile) setToken(context, token string) {
	if context == doctl.ArgDefaultContext {
		c.doc[doctl.ArgAccessToken] = token
		return
	}

	contexts := c.contexts()
	contexts[c.contextKey(context)] = token
	c.doc[authContextsKey] = contexts
}

// setCurrentContext records which context later commands should use.
func (c *configFile) setCurrentContext(context string) {
	c.doc[doctl.ArgContext] = context
}

// hasContext reports whether the file knows the named context. The default
// context always exists, since it is implied by the top-level access-token
// rather than listed alongside the others.
func (c *configFile) hasContext(context string) bool {
	if context == doctl.ArgDefaultContext {
		return true
	}

	_, ok := c.contexts()[c.contextKey(context)]

	return ok
}

// removeContext deletes a context's credentials.
func (c *configFile) removeContext(context string) error {
	if context == doctl.ArgDefaultContext {
		// The default context has no entry of its own to delete, so its token
		// is blanked instead. Leaving the key behind rather than removing it
		// preserves the shape a config file has always had.
		c.doc[doctl.ArgAccessToken] = ""
		return nil
	}

	contexts := c.contexts()

	key := c.contextKey(context)
	if _, ok := contexts[key]; !ok {
		return errors.New("Context not found")
	}

	delete(contexts, key)
	c.doc[authContextsKey] = contexts

	return nil
}

// write persists the document.
func (c *configFile) write() error {
	b, err := yaml.Marshal(c.doc)
	if err != nil {
		return errors.New("Unable to encode configuration to YAML format.")
	}

	f, err := cfgFileWriter()
	if err != nil {
		return err
	}

	defer f.Close()

	if _, err := f.Write(b); err != nil {
		return errors.New("Unable to write configuration.")
	}

	return nil
}

// contexts returns the named contexts, normalised to a map this code can write
// to. yaml decodes nested mappings with interface keys, and a file that has
// never held a named context has no such key at all.
func (c *configFile) contexts() map[any]any {
	switch v := c.doc[authContextsKey].(type) {
	case map[any]any:
		return v
	case map[string]any:
		contexts := make(map[any]any, len(v))
		for name, token := range v {
			contexts[name] = token
		}
		return contexts
	default:
		return map[any]any{}
	}
}

// contextKey finds the key a context is already stored under, so a name
// differing only in case updates that entry instead of adding a second one.
// doctl lowercases names before they reach here; files written by an older
// version, or by hand, may not have.
func (c *configFile) contextKey(context string) any {
	for key := range c.contexts() {
		if name, ok := key.(string); ok && strings.EqualFold(name, context) {
			return key
		}
	}

	return context
}

// defaultConfigFileReader reads the config file viper was pointed at.
func defaultConfigFileReader() ([]byte, error) {
	b, err := os.ReadFile(viper.GetString("config"))
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}

	return b, err
}

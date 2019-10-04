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

package doctl

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/blang/semver"
	"github.com/digitalocean/doctl/config"
	"github.com/digitalocean/doctl/pkg/runner"
	"github.com/digitalocean/doctl/pkg/ssh"
	"github.com/digitalocean/godo"
	"golang.org/x/oauth2"
)

const (
	// LatestReleaseURL is the latest release URL endpoint.
	LatestReleaseURL = "https://api.github.com/repos/digitalocean/doctl/releases/latest"
)

// Version is the version info for doit.
type Version struct {
	Major, Minor, Patch int
	Name, Build, Label  string
}

var (
	// Build, Major, Minor, Patch and Label are set at build time
	Build, Major, Minor, Patch, Label string

	// DoitVersion is doctl's version.
	DoitVersion Version

	// TraceHTTP traces http connections.
	TraceHTTP bool
)

func init() {
	if Build != "" {
		DoitVersion.Build = Build
	}
	if Major != "" {
		i, _ := strconv.Atoi(Major)
		DoitVersion.Major = i
	}
	if Minor != "" {
		i, _ := strconv.Atoi(Minor)
		DoitVersion.Minor = i
	}
	if Patch != "" {
		i, _ := strconv.Atoi(Patch)
		DoitVersion.Patch = i
	}
	if Label == "" {
		DoitVersion.Label = "dev"
	} else {
		DoitVersion.Label = Label
	}
}

func (v Version) String() string {
	var buffer bytes.Buffer
	buffer.WriteString(fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch))
	if v.Label != "" {
		buffer.WriteString("-" + v.Label)
	}

	return buffer.String()
}

// Complete is the complete version for doit.
func (v Version) Complete(lv LatestVersioner) string {
	var buffer bytes.Buffer
	buffer.WriteString(fmt.Sprintf("doctl version %s", v.String()))

	if v.Build != "" {
		buffer.WriteString(fmt.Sprintf("\nGit commit hash: %s", v.Build))
	}

	if tagName, err := lv.LatestVersion(); err == nil {
		v0, err1 := semver.Make(tagName)
		v1, err2 := semver.Make(v.String())

		if len(v0.Build) == 0 {
			v0, err1 = semver.Make(tagName + "-release")
		}

		if err1 == nil && err2 == nil && v0.GT(v1) {
			buffer.WriteString(fmt.Sprintf("\nrelease %s is available, check it out! ", tagName))
		}
	}

	return buffer.String()
}

// LatestVersioner an interface for detecting the latest version.
type LatestVersioner interface {
	LatestVersion() (string, error)
}

// GithubLatestVersioner retrieves the latest version from Github.
type GithubLatestVersioner struct{}

var _ LatestVersioner = &GithubLatestVersioner{}

// LatestVersion retrieves the latest version from Github or returns
// an error.
func (glv *GithubLatestVersioner) LatestVersion() (string, error) {
	u := LatestReleaseURL
	res, err := http.Get(u)
	if err != nil {
		return "", err
	}

	defer res.Body.Close()

	var m map[string]interface{}
	if err = json.NewDecoder(res.Body).Decode(&m); err != nil {
		return "", err
	}

	tn, ok := m["tag_name"]
	if !ok {
		return "", errors.New("could not find tag name in response")
	}

	tagName := tn.(string)
	return strings.TrimPrefix(tagName, "v"), nil
}

// Config is an interface that represent doit's config.
type Config interface {
	GetGodoClient(trace bool, accessToken string) (*godo.Client, error)
	SSH(user, host, keyPath string, port int, opts ssh.Options) runner.Runner
	Set(ns, key string, val interface{})
	IsSet(ns, key string) bool
	GetString(ns, key string) (string, error)
	GetBool(ns, key string) (bool, error)
	GetBoolPtr(ns, key string) (*bool, error)
	GetInt(ns, key string) (int, error)
	GetIntPtr(ns, key string) (*int, error)
	GetStringSlice(ns, key string) ([]string, error)
}

// LiveConfig is an implementation of Config for live values.
type LiveConfig struct {}

var _ Config = &LiveConfig{}

// GetGodoClient returns a GodoClient.
func (c *LiveConfig) GetGodoClient(trace bool, accessToken string) (*godo.Client, error) {
	if accessToken == "" {
		return nil, fmt.Errorf("access token is required. (hint: run 'doctl auth init')")
	}

	tokenSource := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: accessToken})
	oauthClient := oauth2.NewClient(context.Background(), tokenSource)

	if trace {
		r := newRecorder(oauthClient.Transport)

		go func() {
			for {
				select {
				case msg := <-r.req:
					log.Println("->", strconv.Quote(msg))
				case msg := <-r.resp:
					log.Println("<-", strconv.Quote(msg))
				}
			}
		}()

		oauthClient.Transport = r
	}

	args := []godo.ClientOpt{godo.SetUserAgent(userAgent())}

	apiURL := config.RootConfig.GetString("api-url")
	if apiURL != "" {
		args = append(args, godo.SetBaseURL(apiURL))
	}

	return godo.New(oauthClient, args...)
}

func userAgent() string {
	return "doctl/" + DoitVersion.String()
}

// SSH creates a ssh connection to a host.
func (c *LiveConfig) SSH(user, host, keyPath string, port int, opts ssh.Options) runner.Runner {
	return &ssh.Runner{
		User:            user,
		Host:            host,
		KeyPath:         keyPath,
		Port:            port,
		AgentForwarding: opts[ArgsSSHAgentForwarding].(bool),
		Command:         opts[ArgSSHCommand].(string),
	}
}

// Set sets a config key.
func (c *LiveConfig) Set(ns, key string, val interface{}) {
	config.RootConfig.Set(NsKey(ns, key), val)
}

// IsSet checks whether flag is set.
func (c *LiveConfig) IsSet(ns, key string) bool {
	return config.RootConfig.IsSet(NsKey(ns, key))
}

// GetString returns a config value as a string.
func (c *LiveConfig) GetString(ns, key string) (string, error) {
	nskey := NsKey(ns, key)
	str := config.RootConfig.GetString(nskey)

	if isRequired(nskey) && strings.TrimSpace(str) == "" {
		return "", NewMissingArgsErr(nskey)
	}
	return str, nil
}

// GetBool returns a config value as a bool.
func (c *LiveConfig) GetBool(ns, key string) (bool, error) {
	return config.RootConfig.GetBool(NsKey(ns, key)), nil
}

// GetBoolPtr returns a config value as a bool pointer.
func (c *LiveConfig) GetBoolPtr(ns, key string) (*bool, error) {
	if !c.IsSet(ns, key) {
		return nil, nil
	}
	val := config.RootConfig.GetBool(NsKey(ns, key))
	return &val, nil
}

// GetInt returns a config value as an int.
func (c *LiveConfig) GetInt(ns, key string) (int, error) {
	nskey := NsKey(ns, key)
	val := config.RootConfig.GetInt(nskey)

	if isRequired(nskey) && val == 0 {
		return 0, NewMissingArgsErr(nskey)
	}
	return val, nil
}

// GetIntPtr returns a config value as an int pointer.
func (c *LiveConfig) GetIntPtr(ns, key string) (*int, error) {
	nskey := NsKey(ns, key)

	if !c.IsSet(ns, key) {
		if isRequired(nskey) {
			return nil, NewMissingArgsErr(nskey)
		}
		return nil, nil
	}
	val := config.RootConfig.GetInt(nskey)
	return &val, nil
}

// GetStringSlice returns a config value as a string slice.
func (c *LiveConfig) GetStringSlice(ns, key string) ([]string, error) {
	nskey := NsKey(ns, key)
	val := config.RootConfig.GetStringSlice(nskey)

	if isRequired(nskey) && emptyStringSlice(val) {
		return nil, NewMissingArgsErr(nskey)
	}

	out := []string{}
	for _, item := range config.RootConfig.GetStringSlice(nskey) {
		item = strings.TrimPrefix(item, "[")
		item = strings.TrimSuffix(item, "]")

		list := strings.Split(item, ",")
		for _, str := range list {
			if str == "" {
				continue
			}
			out = append(out, str)
		}
	}
	return out, nil
}

func NsKey(ns, key string) string {
	return fmt.Sprintf("%s.%s", ns, key)
}

func isRequired(key string) bool {
	return config.RootConfig.GetBool(fmt.Sprintf("required.%s", key))
}

// This is needed because an empty StringSlice flag returns `["[]"]`
func emptyStringSlice(s []string) bool {
	return len(s) == 1 && s[0] == "[]"
}

// CommandName returns the name by which doctl was invoked
func CommandName() string {
	name, ok := os.LookupEnv("SNAP_NAME")
	if !ok {
		return os.Args[0]
	}
	return name
}

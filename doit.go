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
	"regexp"
	"strconv"
	"strings"

	"github.com/blang/semver"
	"github.com/digitalocean/doctl/pkg/runner"
	"github.com/digitalocean/doctl/pkg/ssh"
	"github.com/digitalocean/godo"
	jww "github.com/spf13/jwalterweatherman"
	"github.com/spf13/viper"
	"golang.org/x/oauth2"
)

const (
	// NSRoot is a configuration key that signifies this value is at the root.
	NSRoot = "doctl"

	// LatestReleaseURL is the latest release URL endpoint.
	LatestReleaseURL = "https://api.github.com/repos/digitalocean/doctl/releases/latest"
)

var (
	// DoitConfig holds the app's current configuration.
	DoitConfig Config = &LiveConfig{}

	// DoitVersion is doit's version.
	DoitVersion = Version{
		Major: 1,
		Minor: 19,
		Patch: 0,
		Label: "dev",
	}

	// Build is doit's build tag.
	Build string

	// Major is doctl's major version.
	Major string

	// Minor is doctl's minor version.
	Minor string

	// Patch is doctl's patch version.
	Patch string

	// Label is doctl's label.
	Label string

	// TraceHTTP traces http connections.
	TraceHTTP bool
)

func init() {
	jww.SetStdoutThreshold(jww.LevelError)
}

// Version is the version info for doit.
type Version struct {
	Major, Minor, Patch int
	Name, Build, Label  string
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
	IsSet(key string) bool
	GetString(ns, key string) (string, error)
	GetBool(ns, key string) (bool, error)
	GetInt(ns, key string) (int, error)
	GetStringSlice(ns, key string) ([]string, error)
}

// LiveConfig is an implementation of Config for live values.
type LiveConfig struct {
	godoClient *godo.Client
	cliArgs    map[string]bool
}

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

	apiURL := viper.GetString("api-url")
	if apiURL != "" {
		args = append(args, godo.SetBaseURL(apiURL))
	}

	godoClient, err := godo.New(oauthClient, args...)
	if err != nil {
		return nil, err
	}

	c.godoClient = godoClient
	return c.godoClient, nil
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
	nskey := fmt.Sprintf("%s-%s", ns, key)
	viper.Set(nskey, val)
}

func (c *LiveConfig) IsSet(key string) bool {
	r := regexp.MustCompile("\b*--([a-z-_]+)")
	matches := r.FindAllStringSubmatch(strings.Join(os.Args, " "), -1)
	if len(matches) == 0 {
		return false
	}

	if len(c.cliArgs) == 0 {
		args := make(map[string]bool)
		for _, match := range matches {
			args[match[1]] = true
		}
		c.cliArgs = args
	}

	return c.cliArgs[key]
}

// GetString returns a config value as a string.
func (c *LiveConfig) GetString(ns, key string) (string, error) {
	if ns == NSRoot {
		return viper.GetString(key), nil
	}

	nskey := fmt.Sprintf("%s.%s", ns, key)

	isRequired := viper.GetBool(fmt.Sprintf("required.%s", nskey))
	str := viper.GetString(nskey)

	if isRequired && strings.TrimSpace(str) == "" {
		return "", NewMissingArgsErr(nskey)
	}

	return str, nil
}

// GetBool returns a config value as a bool.
func (c *LiveConfig) GetBool(ns, key string) (bool, error) {
	if ns == NSRoot {
		return viper.GetBool(key), nil
	}

	nskey := fmt.Sprintf("%s.%s", ns, key)

	return viper.GetBool(nskey), nil
}

// GetInt returns a config value as an int.
func (c *LiveConfig) GetInt(ns, key string) (int, error) {
	if ns == NSRoot {
		return viper.GetInt(key), nil
	}

	nskey := fmt.Sprintf("%s.%s", ns, key)

	isRequired := viper.GetBool(fmt.Sprintf("required.%s", nskey))
	val := viper.GetInt(nskey)

	if isRequired && val == 0 {
		return 0, NewMissingArgsErr(nskey)
	}

	return val, nil
}

// GetStringSlice returns a config value as a string slice.
func (c *LiveConfig) GetStringSlice(ns, key string) ([]string, error) {
	if ns == NSRoot {
		return viper.GetStringSlice(key), nil
	}

	nskey := fmt.Sprintf("%s.%s", ns, key)

	isRequired := viper.GetBool(fmt.Sprintf("required.%s", nskey))
	val := viper.GetStringSlice(nskey)
	if isRequired && emptyStringSlice(val) {
		return nil, NewMissingArgsErr(nskey)
	}

	out := []string{}
	for _, item := range viper.GetStringSlice(nskey) {
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

// This is needed because an empty StringSlice flag returns `["[]"]`
func emptyStringSlice(s []string) bool {
	return len(s) == 1 && s[0] == "[]"
}

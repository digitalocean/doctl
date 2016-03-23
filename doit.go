package doit

import (
	"bytes"
	"fmt"
	"log"
	"strconv"

	"github.com/bryanl/doit/pkg/runner"
	"github.com/bryanl/doit/pkg/ssh"
	"github.com/digitalocean/godo"
	jww "github.com/spf13/jwalterweatherman"
	"github.com/spf13/viper"
	"golang.org/x/oauth2"
)

const (
	// NSRoot is a configuration key that signifies this value is at the root.
	NSRoot = "doit"
)

var (
	// DoitConfig holds the app's current configuration.
	DoitConfig Config = &LiveConfig{}

	// DoitVersion is doit's version.
	DoitVersion = Version{
		Major: 1,
		Minor: 0,
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

	if v.Build != "" {
		buffer.WriteString(" " + v.Build)
	}

	return buffer.String()
}

// Complete is the complete version for doit.
func (v Version) Complete() string {
	var buffer bytes.Buffer
	buffer.WriteString(fmt.Sprintf("doit version %s", v.String()))

	buffer.WriteString(fmt.Sprintf(" %q", v.Name))

	if v.Build != "" {
		buffer.WriteString(fmt.Sprintf("\nGit commit hash: %s", v.Build))
	}

	return buffer.String()
}

// Config is an interface that represent doit's config.
type Config interface {
	GetGodoClient(trace bool) *godo.Client
	SSH(user, host, keyPath string, port int) runner.Runner
	Set(ns, key string, val interface{})
	GetString(ns, key string) (string, error)
	GetBool(ns, key string) (bool, error)
	GetInt(ns, key string) (int, error)
	GetStringSlice(ns, key string) ([]string, error)
}

// LiveConfig is an implementation of Config for live values.
type LiveConfig struct {
	godoClient *godo.Client
}

var _ Config = &LiveConfig{}

// GetGodoClient returns a GodoClient.
func (c *LiveConfig) GetGodoClient(trace bool) *godo.Client {
	if c.godoClient != nil {
		return c.godoClient
	}

	token := viper.GetString("access-token")
	tokenSource := &TokenSource{AccessToken: token}
	oauthClient := oauth2.NewClient(oauth2.NoContext, tokenSource)

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

	c.godoClient = godo.NewClient(oauthClient)
	return c.godoClient
}

// SSH creates a ssh connection to a host.
func (c *LiveConfig) SSH(user, host, keyPath string, port int) runner.Runner {
	return &ssh.Runner{
		User:    user,
		Host:    host,
		KeyPath: keyPath,
		Port:    port,
	}

}

// Set sets a config key.
func (c *LiveConfig) Set(ns, key string, val interface{}) {
	nskey := fmt.Sprintf("%s-%s", ns, key)
	viper.Set(nskey, val)
}

// GetString returns a config value as a string.
func (c *LiveConfig) GetString(ns, key string) (string, error) {
	if ns == NSRoot {
		return viper.GetString(key), nil
	}

	nskey := fmt.Sprintf("%s.%s", ns, key)

	if _, ok := viper.AllSettings()[fmt.Sprintf("%s.required", nskey)]; ok {
		if viper.GetString(nskey) == "" {
			return "", NewMissingArgsErr(nskey)
		}
	}
	return viper.GetString(nskey), nil
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

	if _, ok := viper.AllSettings()[fmt.Sprintf("%s.required", nskey)]; ok {
		if viper.GetInt(nskey) < 0 {
			return 0, NewMissingArgsErr(nskey)
		}
	}

	return viper.GetInt(nskey), nil
}

// GetStringSlice returns a config value as a string slice.
func (c *LiveConfig) GetStringSlice(ns, key string) ([]string, error) {
	if ns == NSRoot {
		return viper.GetStringSlice(key), nil
	}

	nskey := fmt.Sprintf("%s.%s", ns, key)

	if _, ok := viper.AllSettings()[fmt.Sprintf("%s.required", nskey)]; ok {
		if viper.GetStringSlice(nskey) == nil {
			return nil, NewMissingArgsErr(nskey)
		}
	}

	return viper.GetStringSlice(nskey), nil
}

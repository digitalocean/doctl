package config

import (
	"time"

	"github.com/digitalocean/doctl"
)

// doctlConfigSource wraps doctl.Config to implement the ConfigSource interface.
type doctlConfigSource struct {
	config doctl.Config
}

func (c *doctlConfigSource) IsSet(key string) bool {
	return c.config.IsSet(key)
}

func (c *doctlConfigSource) GetString(key string) string {
	v, _ := c.config.GetString("", key)
	return v
}

func (c *doctlConfigSource) GetBool(key string) bool {
	v, _ := c.config.GetBool("", key)
	return v
}

func (c *doctlConfigSource) GetDuration(key string) time.Duration {
	v, _ := c.config.GetDuration("", key)
	return v
}

// DoctlConfigSource converts a doctl.Config into a ConfigSource with an optional default namespace.
func DoctlConfigSource(config doctl.Config, ns string) ConfigSource {
	var mutateKey func(string) string
	if ns != "" {
		mutateKey = func(key string) string {
			return nsKey(ns, key)
		}
	}
	return &mutatingConfigSource{
		cs:        &doctlConfigSource{config},
		mutateKey: mutateKey,
	}
}

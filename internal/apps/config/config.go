package config

import (
	"strings"
	"time"
)

// ConfigSource is a config source.
type ConfigSource interface {
	IsSet(key string) bool
	GetString(key string) string
	GetBool(key string) bool
	GetDuration(key string) time.Duration
}

type mutatingConfigSource struct {
	cs          ConfigSource
	mutateKey   func(key string) string
	mutateIsSet bool // some config sources except IsSet to _not_ include the namespace
}

func (s *mutatingConfigSource) key(key string) string {
	if s.mutateKey != nil {
		key = s.mutateKey(key)
	}
	return key
}

func (s *mutatingConfigSource) IsSet(key string) bool {
	if s.mutateIsSet {
		key = s.key(key)
	}
	return s.cs.IsSet(key)
}

func (s *mutatingConfigSource) GetString(key string) string {
	return s.cs.GetString(s.key(key))
}

func (s *mutatingConfigSource) GetBool(key string) bool {
	return s.cs.GetBool(s.key(key))
}

func (s *mutatingConfigSource) GetDuration(key string) time.Duration {
	return s.cs.GetDuration(s.key(key))
}

func MutatingConfigSource(cs ConfigSource, mutateKey func(key string) string) ConfigSource {
	return &mutatingConfigSource{
		cs:          cs,
		mutateKey:   mutateKey,
		mutateIsSet: true,
	}
}

// NamespacedConfigSource accepts a ConfigSource and configures a default namespace that is prefixed to the key on all
// Get* calls.
func NamespacedConfigSource(cs ConfigSource, ns string) ConfigSource {
	var mutateKey func(string) string
	if ns != "" {
		mutateKey = func(key string) string {
			return nsKey(ns, key)
		}
	}
	return MutatingConfigSource(cs, mutateKey)
}

// Multi returns a config source that wraps multiple config sources.
// Each source is evaluated in order and the first match is returned.
func Multi(sources ...ConfigSource) ConfigSource {
	return &multiConfigSource{sources}
}

type multiConfigSource struct {
	sources []ConfigSource
}

func (s *multiConfigSource) IsSet(key string) bool {
	for _, s := range s.sources {
		if s != nil && s.IsSet(key) {
			return true
		}
	}
	return false
}

func (s *multiConfigSource) GetString(key string) string {
	for _, s := range s.sources {
		if s != nil && s.IsSet(key) {
			return s.GetString(key)
		}
	}
	return ""
}

func (s *multiConfigSource) GetBool(key string) bool {
	for _, s := range s.sources {
		if s != nil && s.IsSet(key) {
			return s.GetBool(key)
		}
	}
	return false
}

func (s *multiConfigSource) GetDuration(key string) time.Duration {
	for _, s := range s.sources {
		if s != nil && s.IsSet(key) {
			return s.GetDuration(key)
		}
	}
	return 0
}

func nsKey(parts ...string) string {
	return strings.Join(parts, ".")
}

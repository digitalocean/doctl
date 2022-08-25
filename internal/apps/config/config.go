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
	cs        ConfigSource
	mutateKey func(key string) string
	// excludeMethods is a list of methods that should receive the original non-mutated input. example: "IsSet".
	excludeMethods map[string]bool
}

func (s *mutatingConfigSource) key(method string, key string) string {
	if !s.excludeMethods[method] && s.mutateKey != nil {
		key = s.mutateKey(key)
	}
	return key
}

func (s *mutatingConfigSource) IsSet(key string) bool {
	return s.cs.IsSet(s.key("IsSet", key))
}

func (s *mutatingConfigSource) GetString(key string) string {
	return s.cs.GetString(s.key("GetString", key))
}

func (s *mutatingConfigSource) GetBool(key string) bool {
	return s.cs.GetBool(s.key("GetBool", key))
}

func (s *mutatingConfigSource) GetDuration(key string) time.Duration {
	return s.cs.GetDuration(s.key("GetDuration", key))
}

func MutatingConfigSource(cs ConfigSource, mutateKey func(key string) string, excludeMethods []string) ConfigSource {
	excludeMethodsMap := make(map[string]bool)
	for _, m := range excludeMethods {
		excludeMethodsMap[m] = true
	}
	return &mutatingConfigSource{
		cs:             cs,
		mutateKey:      mutateKey,
		excludeMethods: excludeMethodsMap,
	}
}

// KeyNamespaceMutator returns a mutator that prefixes a namespace to the key with a `.` delimiter.
func KeyNamespaceMutator(ns string) func(key string) string {
	return func(key string) string {
		if ns == "" {
			return key
		}
		return nsKey(ns, key)
	}
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

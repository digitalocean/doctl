package doit

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v2"
)

const (
	pathSplitter = "."
)

type configMap map[interface{}]interface{}

// Config2 is the configuration for Doit.
type Config2 struct {
	raw []byte
}

// NewConfig2 creates a Config2 from a reader.
func NewConfig2(raw []byte) (*Config2, error) {
	return &Config2{
		raw: raw,
	}, nil
}

// Int returns an integer parameter.
func (c *Config2) Int(path string) (int, error) {
	raw, err := c.extractKey(path)
	if err != nil {
		return 0, err
	}

	return raw.(int), nil
}

// String returns a string parameter.
func (c *Config2) String(path string) (string, error) {
	raw, err := c.extractKey(path)
	if err != nil {
		return "", err
	}

	return raw.(string), nil
}

// Bool returns a boolean parameter.
func (c *Config2) Bool(path string) (bool, error) {
	raw, err := c.extractKey(path)
	if err != nil {
		return false, err
	}

	return raw.(bool), nil
}

func (c *Config2) extractKey(path string) (interface{}, error) {
	var cm configMap
	err := yaml.Unmarshal(c.raw, &cm)
	if err != nil {
		return nil, err
	}

	keys := strings.Split(path, pathSplitter)
	for _, k := range keys {
		v, ok := cm[k]
		if !ok {
			return nil, fmt.Errorf("unable to find key %q in path %q", k, path)
		}

		switch v.(type) {
		case configMap:
			cm = v.(configMap)
		default:
			break
		}
	}

	keyI := len(keys) - 1
	raw, ok := cm[keys[keyI]]
	if !ok {
		return "", fmt.Errorf("unable to find value for key %q in path %q", keyI, path)
	}

	return raw, nil
}

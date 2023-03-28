package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestMutatingConfigSource(t *testing.T) {
	// cs - keyed by word
	cs := NewTestConfigSource(map[string]any{
		"one":   true,
		"two":   time.Minute,
		"three": "hello",
	})
	for _, k := range []string{"one", "two", "three"} {
		assert.True(t, cs.IsSet(k), k)
	}
	for _, k := range []string{"1", "2", "3"} {
		assert.False(t, cs.IsSet(k), k)
	}

	// mcs - keyed by number
	mcs := MutatingConfigSource(cs, func(key string) string {
		// translate number to word
		return map[string]string{
			"1": "one",
			"2": "two",
			"3": "three",
		}[key]
	}, nil)
	for _, k := range []string{"one", "two", "three"} {
		assert.False(t, mcs.IsSet(k), k)
	}
	for _, k := range []string{"1", "2", "3"} {
		assert.True(t, mcs.IsSet(k), k)
	}

	assert.Equal(t, true, mcs.GetBool("1"))
	assert.Equal(t, time.Minute, mcs.GetDuration("2"))
	assert.Equal(t, "hello", mcs.GetString("3"))
}

func TestKeyNamespaceMutator(t *testing.T) {
	empty := KeyNamespaceMutator("")
	assert.Equal(t, "", empty(""))
	assert.Equal(t, "key", empty("key"))

	ns := KeyNamespaceMutator("namespace")
	assert.Equal(t, "namespace.", ns(""))
	assert.Equal(t, "namespace.key", ns("key"))
}

func TestMulti(t *testing.T) {
	items1 := map[string]any{
		"name":   "Bufo",
		"ttl":    3 * time.Hour,
		"verify": true,
	}
	c1 := NewTestConfigSource(items1)
	items2 := map[string]any{
		"slug":   "bufo",
		"verify": false,
	}
	c2 := NewTestConfigSource(items2)

	// c2 should be evaluated before c1
	multi := Multi(c2, c1)
	for k := range items1 {
		assert.True(t, c1.IsSet(k), k)
		assert.True(t, multi.IsSet(k), k)
	}
	for k := range items2 {
		assert.True(t, c2.IsSet(k), k)
		assert.True(t, multi.IsSet(k), k)
	}

	assert.Equal(t, "Bufo", multi.GetString("name"))       // c1
	assert.Equal(t, 3*time.Hour, multi.GetDuration("ttl")) // c1
	assert.Equal(t, false, multi.GetBool("verify"))        // c2 overrides c1
	assert.Equal(t, "bufo", multi.GetString("slug"))       // c2
}

func NewTestConfigSource(vals map[string]any) *TestConfigSource {
	m := make(map[string]any)
	for k, v := range vals {
		m[k] = v
	}
	return &TestConfigSource{m}
}

type TestConfigSource struct {
	vals map[string]any
}

func (s *TestConfigSource) Set(key string, value any) {
	s.vals[key] = value
}

func (s *TestConfigSource) IsSet(key string) bool {
	_, ok := s.vals[key]
	return ok
}

func (s *TestConfigSource) GetString(key string) string {
	v, ok := s.vals[key]
	if !ok {
		return ""
	}
	vv, ok := v.(string)
	if !ok {
		return ""
	}
	return vv
}

func (s *TestConfigSource) GetBool(key string) bool {
	v, ok := s.vals[key]
	if !ok {
		return false
	}
	vv, ok := v.(bool)
	if !ok {
		return false
	}
	return vv
}

func (s *TestConfigSource) GetDuration(key string) time.Duration {
	v, ok := s.vals[key]
	if !ok {
		return 0
	}
	vv, ok := v.(time.Duration)
	if !ok {
		return 0
	}
	return vv
}

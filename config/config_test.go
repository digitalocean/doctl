/*
Copyright 2016 The Doctl Authors All rights reserved.
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

package config

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"
)

func TestSet(t *testing.T) {
	var b bytes.Buffer
	c, err := New(&b)
	assert.NoError(t, err)

	c.Set("foo", "bar")
	err = c.Save(&b)
	assert.NoError(t, err)

	var m map[string]interface{}
	err = yaml.Unmarshal(b.Bytes(), &m)
	assert.NoError(t, err)

	assert.Equal(t, "bar", m["foo"])
}

func TestSetChildKey(t *testing.T) {
	var b bytes.Buffer
	c, err := New(&b)
	assert.NoError(t, err)

	c.Set("foo.bar", "baz")
	err = c.Save(&b)
	assert.NoError(t, err)

	v := c.Get("foo.bar")
	assert.Equal(t, "baz", v.(string))
}

func TestGet(t *testing.T) {
	in := []byte("foo:\n  bar: baz")
	buf := bytes.NewBuffer(in)

	c, err := New(buf)
	assert.NoError(t, err)

	v := c.Get("foo.bar")
	assert.Equal(t, "baz", v)
}

func TestList(t *testing.T) {
	in := []byte("foo:\n  bar: good\n  one: good")
	buf := bytes.NewBuffer(in)

	c, err := New(buf)
	assert.NoError(t, err)

	m := c.List()

	keys := []string{"foo.bar", "foo.one"}
	for _, k := range keys {
		assert.Equal(t, "good", m[k])
	}
}

func TestDeleteKey(t *testing.T) {
	in := []byte("foo:\n  bar: good\n  one: good")
	buf := bytes.NewBuffer(in)

	c, err := New(buf)
	assert.NoError(t, err)

	err = c.Delete("foo.bar")
	assert.NoError(t, err)

	_, ok := c.List()["foo.bar"]
	assert.False(t, ok)
}

func TestDeleteTree(t *testing.T) {
	in := []byte("foo:\n  bar: good\n  one: good")
	buf := bytes.NewBuffer(in)

	c, err := New(buf)
	assert.NoError(t, err)

	err = c.Delete("foo")
	assert.NoError(t, err)

	assert.Len(t, c.List(), 1)
}

package doit

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateGlobalArgs(t *testing.T) {
	argDir := ConfigArgDir{
		"": {
			"t1": "test1",
			"t2": "test2",
		},
	}

	cfg := `
t1: ex1
t2: ex2`

	expected := []string{
		"--test1", "ex1",
		"--test2", "ex2",
	}

	cf := NewConfigFile(argDir, []byte(cfg))
	got, err := cf.Args("")
	assert.NoError(t, err)
	assert.Equal(t, expected, got)
}

func TestGenerateCommandArgs(t *testing.T) {
	argDir := ConfigArgDir{
		"foo/bar": {
			"baz": "test1",
		},
	}

	cfg := `
commands:
  foo:
    bar:
      baz: qux`

	expected := []string{
		"--test1", "qux",
	}

	cf := NewConfigFile(argDir, []byte(cfg))
	got, err := cf.Args("foo/bar")
	assert.NoError(t, err)
	assert.Equal(t, expected, got)
}

func TestPrependGlobalArgs(t *testing.T) {
	newArgs := []string{
		"--foo", "bar",
	}

	osargs := []string{
		"myapp", "action",
	}

	got := GlobalArgs(osargs, newArgs)
	expected := []string{
		"myapp", "--foo", "bar", "action",
	}

	assert.Equal(t, expected, got)
}

func TestAppendCommandArgs(t *testing.T) {
	newArgs := []string{
		"--bar", "baz",
	}

	osargs := []string{
		"myapp", "action",
	}

	got := CommandArgs(osargs, newArgs)
	expected := []string{
		"myapp", "action", "--bar", "baz",
	}

	assert.Equal(t, expected, got)
}

func Test_mapPath(t *testing.T) {
	m := yamlMap{
		"foo": yamlMap{
			"bar": yamlMap{
				"baz": "qux",
			},
		},
	}

	cases := []struct {
		e     interface{}
		p     string
		isErr bool
	}{
		{e: yamlMap{"baz": "qux"}, p: "foo/bar"},
		{e: m, p: ""},
		{e: nil, p: "missing", isErr: true},
	}

	for _, c := range cases {
		got, err := mapPath(m, c.p)

		switch c.isErr {
		case true:
			assert.Error(t, err)
		default:
			assert.NoError(t, err)
		}
		assert.Equal(t, c.e, got, "for path:"+c.p)
	}
}

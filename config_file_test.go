package doit

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateArgs(t *testing.T) {
	argMap := ConfigArgMap{
		"t1": "test1",
		"t2": "test2",
	}

	cfg := `
t1: ex1
t2: ex2`

	expected := []string{
		"--test1", "ex1",
		"--test2", "ex2",
	}

	cf := NewConfigFile(argMap, []byte(cfg))
	got, err := cf.Args()
	assert.NoError(t, err)
	assert.Equal(t, expected, got)
}

func TestInsertArgs(t *testing.T) {
	newArgs := []string{
		"--foo", "bar",
	}

	osargs := []string{
		"myapp", "action",
	}

	got := InsertArgs(osargs, newArgs)
	expected := []string{
		"myapp", "--foo", "bar", "action",
	}

	assert.Equal(t, expected, got)
}

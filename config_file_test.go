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

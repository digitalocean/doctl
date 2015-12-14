package commands

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVersionCommand(t *testing.T) {
	cmd := Version()
	assert.NotNil(t, cmd)
	assertCommandNames(t, cmd)
}

package doit

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMissingArgsErr(t *testing.T) {
	err := NewMissingArgsErr("test-cmd")
	assert.Equal(t, "(test-cmd) command is missing required arguments", err.Error())
}

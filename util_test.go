package doit

import (
	"fmt"
	"testing"

	"github.com/bryanl/doit/Godeps/_workspace/src/github.com/stretchr/testify/assert"
)

func TestMockRunner(t *testing.T) {
	e := fmt.Errorf("an error")
	mr := MockRunner{e}

	assert.Equal(t, e, mr.Run())
}

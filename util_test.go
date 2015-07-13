package doit

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMockRunner(t *testing.T) {
	e := fmt.Errorf("an error")
	mr := mockRunner{e}

	assert.Equal(t, e, mr.Run())
}

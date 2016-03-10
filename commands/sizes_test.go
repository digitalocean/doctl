package commands

import (
	"testing"

	"github.com/bryanl/doit/do"
	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"
)

var (
	testSize     = do.Size{Size: &godo.Size{Slug: "small"}}
	testSizeList = do.Sizes{testSize}
)

func TestSizeCommand(t *testing.T) {
	cmd := Size()
	assert.NotNil(t, cmd)
	assertCommandNames(t, cmd, "list")
}

func TestSizesList(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.sizes.On("List").Return(testSizeList, nil)

		err := RunSizeList(config)
		assert.NoError(t, err)
	})
}

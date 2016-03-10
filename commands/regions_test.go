package commands

import (
	"testing"

	"github.com/bryanl/doit/do"
	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"
)

var (
	testRegion     = do.Region{Region: &godo.Region{Slug: "dev0"}}
	testRegionList = do.Regions{testRegion}
)

func TestRegionCommand(t *testing.T) {
	cmd := Region()
	assert.NotNil(t, cmd)
	assertCommandNames(t, cmd, "list")
}

func TestRegionsList(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.regions.On("List").Return(testRegionList, nil)

		err := RunRegionList(config)
		assert.NoError(t, err)
	})
}

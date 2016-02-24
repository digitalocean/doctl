package commands

import (
	"testing"

	"github.com/bryanl/doit/do"
	domocks "github.com/bryanl/doit/do/mocks"
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
	withTestClient(func(config *cmdConfig) {
		rs := &domocks.RegionsService{}
		config.rs = rs

		rs.On("List").Return(testRegionList, nil)

		err := RunRegionList(config)
		assert.NoError(t, err)
	})
}

package commands

import (
	"testing"

	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"

	"github.com/stretchr/testify/assert"
)

var (
	testOneClick = do.OneClick{
		OneClick: &godo.OneClick{
			Slug: "test-slug",
			Type: "droplet",
		},
	}

	testOneClickList = do.OneClicks{
		testOneClick,
	}
)

func TestOneClickCommand(t *testing.T) {
	cmd := OneClicks()
	assert.NotNil(t, cmd)
	assertCommandNames(t, cmd, "list")
}

func TestOneClickListNoType(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.oneClick.EXPECT().List("").Return(testOneClickList, nil)
		err := RunOneClickList(config)
		assert.NoError(t, err)
	})
}

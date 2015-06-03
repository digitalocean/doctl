package docli

import (
	"flag"
	"testing"

	"github.com/codegangsta/cli"
	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"
)

var (
	testRegion     = godo.Region{Slug: "dev0"}
	testRegionList = []godo.Region{testRegion}
)

func TestRegionsList(t *testing.T) {
	didList := false

	client := &godo.Client{
		Regions: &RegionsServiceMock{
			ListFn: func(opt *godo.ListOptions) ([]godo.Region, *godo.Response, error) {
				didList = true

				resp := &godo.Response{
					Links: &godo.Links{
						Pages: &godo.Pages{},
					},
				}
				return testRegionList, resp, nil
			},
		},
	}

	cs := &TestClientSource{client}
	fs := flag.NewFlagSet("flag set", 0)

	WithinTest(cs, fs, func(c *cli.Context) {
		RegionList(c)
		assert.True(t, didList)
	})
}

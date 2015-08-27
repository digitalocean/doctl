package commands

import (
	"io/ioutil"
	"testing"

	"github.com/bryanl/doit"
	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"
)

var (
	testSize     = godo.Size{Slug: "small"}
	testSizeList = []godo.Size{testSize}
)

func TestSizesList(t *testing.T) {
	didList := false

	client := &godo.Client{
		Sizes: &doit.SizesServiceMock{
			ListFn: func(opt *godo.ListOptions) ([]godo.Size, *godo.Response, error) {
				didList = true

				resp := &godo.Response{
					Links: &godo.Links{
						Pages: &godo.Pages{},
					},
				}
				return testSizeList, resp, nil
			},
		},
	}

	withTestClient(client, func(c *TestConfig) {
		ns := "test"
		RunSizeList(ns, ioutil.Discard)
		assert.True(t, didList)
	})
}

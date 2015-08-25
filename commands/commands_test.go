package commands

import (
	"github.com/bryanl/doit"
	"github.com/digitalocean/godo"
)

var (
	testDroplet = godo.Droplet{
		ID: 1,
		Image: &godo.Image{
			ID:           1,
			Name:         "an-image",
			Distribution: "DOOS",
		},
		Name: "a-droplet",
		Networks: &godo.Networks{
			V4: []godo.NetworkV4{
				{IPAddress: "8.8.8.8", Type: "public"},
				{IPAddress: "172.16.1.2", Type: "private"},
			},
		},
		Region: &godo.Region{
			Slug: "test0",
			Name: "test 0",
		},
	}
	testDropletList = []godo.Droplet{testDroplet}
	testKernel      = godo.Kernel{ID: 1}
	testKernelList  = []godo.Kernel{testKernel}
)

type testFn func(c doit.ViperConfig)

func withTestClient(client *godo.Client, tFn testFn) {
	ogConfig := doit.VConfig
	defer func() {
		doit.VConfig = ogConfig
	}()

	doit.VConfig = doit.NewTestViperConfig(client)

	tFn(doit.VConfig)
}

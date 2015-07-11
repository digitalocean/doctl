package doit

import (
	"os"
	"testing"

	"github.com/digitalocean/godo"
)

var (
	testDroplet = godo.Droplet{
		ID:   1,
		Name: "a-droplet",
		Networks: &godo.Networks{
			V4: []godo.NetworkV4{
				{IPAddress: "8.8.8.8", Type: "public"},
				{IPAddress: "172.16.1.2", Type: "private"},
			},
		},
	}
	testDropletList = []godo.Droplet{testDroplet}
	testKernel      = godo.Kernel{ID: 1}
	testKernelList  = []godo.Kernel{testKernel}

	lastBailOut bailOut
)

type bailOut struct {
	err error
	msg string
}

func TestMain(m *testing.M) {
	Bail = func(err error, msg string) {
		lastBailOut = bailOut{
			err: err,
			msg: msg,
		}
	}

	os.Exit(m.Run())
}

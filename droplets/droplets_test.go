package droplets

import (
	"flag"
	"testing"

	"github.com/bryanl/docli/docli"
	"github.com/codegangsta/cli"
	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"
)

var (
	testAction      = godo.Action{ID: 1}
	testActionList  = []godo.Action{testAction}
	testDroplet     = godo.Droplet{ID: 1}
	testDropletList = []godo.Droplet{testDroplet}
	testKernel      = godo.Kernel{ID: 1}
	testKernelList  = []godo.Kernel{testKernel}
	testImage       = godo.Image{ID: 1}
	testImageList   = []godo.Image{testImage}
)

func TestDropletActionList(t *testing.T) {
	client := &godo.Client{
		Droplets: &docli.DropletsServiceMock{
			ActionsFn: func(id int, opts *godo.ListOptions) ([]godo.Action, *godo.Response, error) {
				if got, expected := id, 1; got != expected {
					t.Errorf("ActionsFn() id = %d; expected %d", got, expected)
				}

				resp := &godo.Response{
					Links: &godo.Links{
						Pages: &godo.Pages{},
					},
				}
				return testActionList, resp, nil
			},
		},
	}

	cs := &docli.TestClientSource{client}
	fs := flag.NewFlagSet("flag set", 0)
	fs.Int(argDropletID, 1, argDropletID)

	docli.WithinTest(cs, fs, func(c *cli.Context) {
		Actions(c)
	})
}

func TestDropletBackupList(t *testing.T) {
	client := &godo.Client{
		Droplets: &docli.DropletsServiceMock{
			BackupsFn: func(id int, opts *godo.ListOptions) ([]godo.Image, *godo.Response, error) {
				if got, expected := id, 1; got != expected {
					t.Errorf("BackupsFn() id = %d; expected %d", got, expected)
				}

				resp := &godo.Response{
					Links: &godo.Links{
						Pages: &godo.Pages{},
					},
				}
				return testImageList, resp, nil
			},
		},
	}

	cs := &docli.TestClientSource{client}
	fs := flag.NewFlagSet("flag set", 0)
	fs.Int(argDropletID, 1, argDropletID)

	docli.WithinTest(cs, fs, func(c *cli.Context) {
		Backups(c)
	})
}

func TestDropletCreate(t *testing.T) {
	client := &godo.Client{
		Droplets: &docli.DropletsServiceMock{
			CreateFn: func(cr *godo.DropletCreateRequest) (*godo.Droplet, *godo.Response, error) {
				expected := &godo.DropletCreateRequest{
					Name:    "droplet",
					Image:   godo.DropletCreateImage{Slug: "image"},
					Region:  "dev0",
					Size:    "1gb",
					SSHKeys: []godo.DropletCreateSSHKey{},
				}

				assert.Equal(t, cr, expected, "create requests did not match")

				return &testDroplet, nil, nil
			},
		},
	}

	cs := &docli.TestClientSource{client}
	fs := flag.NewFlagSet("flag set", 0)
	fs.String(argDropletName, "droplet", argDropletName)
	fs.String(argRegionSlug, "dev0", argRegionSlug)
	fs.String(argSizeSlug, "1gb", argSizeSlug)
	fs.String(argImage, "image", argImage)

	docli.WithinTest(cs, fs, func(c *cli.Context) {
		Create(c)
	})
}

func TestDropletDelete(t *testing.T) {
	client := &godo.Client{
		Droplets: &docli.DropletsServiceMock{
			DeleteFn: func(id int) (*godo.Response, error) {
				assert.Equal(t, id, testDroplet.ID, "droplet ids did not match")
				return nil, nil
			},
		},
	}

	cs := &docli.TestClientSource{client}
	fs := flag.NewFlagSet("flag set", 0)
	fs.Int(argDropletID, testDroplet.ID, argDropletID)

	docli.WithinTest(cs, fs, func(c *cli.Context) {
		Delete(c)
	})
}

func TestDropletGet(t *testing.T) {
	client := &godo.Client{
		Droplets: &docli.DropletsServiceMock{
			GetFn: func(id int) (*godo.Droplet, *godo.Response, error) {
				assert.Equal(t, id, testDroplet.ID, "droplet ids did not match")
				return &testDroplet, nil, nil
			},
		},
	}

	cs := &docli.TestClientSource{client}
	fs := flag.NewFlagSet("flag set", 0)
	fs.Int(argDropletID, testDroplet.ID, argDropletID)

	docli.WithinTest(cs, fs, func(c *cli.Context) {
		Get(c)
	})
}

func TestDropletKernelList(t *testing.T) {
	client := &godo.Client{
		Droplets: &docli.DropletsServiceMock{
			KernelsFn: func(id int, opts *godo.ListOptions) ([]godo.Kernel, *godo.Response, error) {
				if got, expected := id, 1; got != expected {
					t.Errorf("KernelsFn() id = %d; expected %d", got, expected)
				}

				resp := &godo.Response{
					Links: &godo.Links{
						Pages: &godo.Pages{},
					},
				}
				return testKernelList, resp, nil
			},
		},
	}

	cs := &docli.TestClientSource{client}
	fs := flag.NewFlagSet("flag set", 0)
	fs.Int(argDropletID, 1, argDropletID)

	docli.WithinTest(cs, fs, func(c *cli.Context) {
		Kernels(c)
	})
}

func TestDropletNeighbors(t *testing.T) {
	didRun := false
	client := &godo.Client{
		Droplets: &docli.DropletsServiceMock{
			NeighborsFn: func(id int) ([]godo.Droplet, *godo.Response, error) {
				didRun = true
				assert.Equal(t, id, 1)

				resp := &godo.Response{
					Links: &godo.Links{
						Pages: &godo.Pages{},
					},
				}
				return testDropletList, resp, nil
			},
		},
	}

	cs := &docli.TestClientSource{client}
	fs := flag.NewFlagSet("flag set", 0)
	fs.Int(argDropletID, 1, argDropletID)

	docli.WithinTest(cs, fs, func(c *cli.Context) {
		Neighbors(c)
		assert.True(t, didRun)
	})
}

func TestDropletSnapshotList(t *testing.T) {
	client := &godo.Client{
		Droplets: &docli.DropletsServiceMock{
			SnapshotsFn: func(id int, opts *godo.ListOptions) ([]godo.Image, *godo.Response, error) {
				assert.Equal(t, id, 1)

				resp := &godo.Response{
					Links: &godo.Links{
						Pages: &godo.Pages{},
					},
				}
				return testImageList, resp, nil
			},
		},
	}

	cs := &docli.TestClientSource{client}
	fs := flag.NewFlagSet("flag set", 0)
	fs.Int(argDropletID, 1, argDropletID)

	docli.WithinTest(cs, fs, func(c *cli.Context) {
		Snapshots(c)
	})
}

func TestDropletsList(t *testing.T) {
	didRun := false
	client := &godo.Client{
		Droplets: &docli.DropletsServiceMock{
			ListFn: func(opts *godo.ListOptions) ([]godo.Droplet, *godo.Response, error) {
				didRun = true
				resp := &godo.Response{
					Links: &godo.Links{
						Pages: &godo.Pages{},
					},
				}
				return testDropletList, resp, nil
			},
		},
	}

	cs := &docli.TestClientSource{client}

	docli.WithinTest(cs, nil, func(c *cli.Context) {
		List(c)
		assert.True(t, didRun)
	})
}

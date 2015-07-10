package doit

import (
	"flag"
	"testing"

	"github.com/codegangsta/cli"
	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"
)

func TestDropletActionList(t *testing.T) {
	client := &godo.Client{
		Droplets: &DropletsServiceMock{
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

	cs := NewTestConfig(client)
	fs := flag.NewFlagSet("flag set", 0)
	fs.Int(ArgDropletID, 1, ArgDropletID)

	WithinTest(cs, fs, func(c *cli.Context) {
		DropletActions(c)
	})
}

func TestDropletBackupList(t *testing.T) {
	client := &godo.Client{
		Droplets: &DropletsServiceMock{
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

	cs := NewTestConfig(client)
	fs := flag.NewFlagSet("flag set", 0)
	fs.Int(ArgDropletID, 1, ArgDropletID)

	WithinTest(cs, fs, func(c *cli.Context) {
		DropletBackups(c)
	})
}

func TestDropletCreate(t *testing.T) {
	client := &godo.Client{
		Droplets: &DropletsServiceMock{
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

	cs := NewTestConfig(client)
	fs := flag.NewFlagSet("flag set", 0)
	fs.String(ArgDropletName, "droplet", ArgDropletName)
	fs.String(ArgRegionSlug, "dev0", ArgRegionSlug)
	fs.String(ArgSizeSlug, "1gb", ArgSizeSlug)
	fs.String(ArgImage, "image", ArgImage)

	WithinTest(cs, fs, func(c *cli.Context) {
		DropletCreate(c)
	})
}

func TestDropletDelete(t *testing.T) {
	client := &godo.Client{
		Droplets: &DropletsServiceMock{
			DeleteFn: func(id int) (*godo.Response, error) {
				assert.Equal(t, id, testDroplet.ID, "droplet ids did not match")
				return nil, nil
			},
		},
	}

	cs := NewTestConfig(client)
	fs := flag.NewFlagSet("flag set", 0)
	fs.Int(ArgDropletID, testDroplet.ID, ArgDropletID)

	WithinTest(cs, fs, func(c *cli.Context) {
		DropletDelete(c)
	})
}

func TestDropletGet(t *testing.T) {
	client := &godo.Client{
		Droplets: &DropletsServiceMock{
			GetFn: func(id int) (*godo.Droplet, *godo.Response, error) {
				assert.Equal(t, id, testDroplet.ID, "droplet ids did not match")
				return &testDroplet, nil, nil
			},
		},
	}

	cs := NewTestConfig(client)
	fs := flag.NewFlagSet("flag set", 0)
	fs.Int(ArgDropletID, testDroplet.ID, ArgDropletID)

	WithinTest(cs, fs, func(c *cli.Context) {
		DropletGet(c)
	})
}

func TestDropletKernelList(t *testing.T) {
	client := &godo.Client{
		Droplets: &DropletsServiceMock{
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

	cs := NewTestConfig(client)
	fs := flag.NewFlagSet("flag set", 0)
	fs.Int(ArgDropletID, 1, ArgDropletID)

	WithinTest(cs, fs, func(c *cli.Context) {
		DropletKernels(c)
	})
}

func TestDropletNeighbors(t *testing.T) {
	didRun := false
	client := &godo.Client{
		Droplets: &DropletsServiceMock{
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

	cs := NewTestConfig(client)
	fs := flag.NewFlagSet("flag set", 0)
	fs.Int(ArgDropletID, 1, ArgDropletID)

	WithinTest(cs, fs, func(c *cli.Context) {
		DropletNeighbors(c)
		assert.True(t, didRun)
	})
}

func TestDropletSnapshotList(t *testing.T) {
	client := &godo.Client{
		Droplets: &DropletsServiceMock{
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

	cs := NewTestConfig(client)
	fs := flag.NewFlagSet("flag set", 0)
	fs.Int(ArgDropletID, 1, ArgDropletID)

	WithinTest(cs, fs, func(c *cli.Context) {
		DropletSnapshots(c)
	})
}

func TestDropletsList(t *testing.T) {
	didRun := false
	client := &godo.Client{
		Droplets: &DropletsServiceMock{
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

	cs := NewTestConfig(client)

	WithinTest(cs, nil, func(c *cli.Context) {
		DropletList(c)
		assert.True(t, didRun)
	})
}

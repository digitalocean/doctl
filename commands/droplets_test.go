package commands

import (
	"io/ioutil"
	"strconv"
	"testing"

	"github.com/bryanl/doit"
	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"
)

var (
	testImage = godo.Image{
		ID:      1,
		Slug:    "slug",
		Regions: []string{"test0"},
	}
	testImageList = []godo.Image{testImage}
)

func TestDropletCommand(t *testing.T) {
	cmd := Droplet()
	assert.NotNil(t, cmd)
	assertCommandNames(t, cmd, "actions", "backups", "create", "delete", "get", "kernels", "list", "neighbors", "snapshots")
}

func TestDropletActionList(t *testing.T) {
	client := &godo.Client{
		Droplets: &doit.DropletsServiceMock{
			ActionsFn: func(id int, opts *godo.ListOptions) ([]godo.Action, *godo.Response, error) {
				assert.Equal(t, 1, id)

				resp := &godo.Response{
					Links: &godo.Links{
						Pages: &godo.Pages{},
					},
				}
				return testActionList, resp, nil
			},
		},
	}

	withTestClient(client, func(c *TestConfig) {
		ns := "test"
		err := RunDropletActions(ns, c, ioutil.Discard, []string{"1"})
		assert.NoError(t, err)
	})
}

func TestDropletBackupList(t *testing.T) {
	client := &godo.Client{
		Droplets: &doit.DropletsServiceMock{
			BackupsFn: func(id int, opts *godo.ListOptions) ([]godo.Image, *godo.Response, error) {
				assert.Equal(t, 1, id)

				resp := &godo.Response{
					Links: &godo.Links{
						Pages: &godo.Pages{},
					},
				}
				return testImageList, resp, nil
			},
		},
	}

	withTestClient(client, func(c *TestConfig) {
		ns := "test"
		err := RunDropletBackups(ns, c, ioutil.Discard, []string{"1"})
		assert.NoError(t, err)
	})
}

func TestDropletCreate(t *testing.T) {
	client := &godo.Client{
		Droplets: &doit.DropletsServiceMock{
			CreateFn: func(cr *godo.DropletCreateRequest) (*godo.Droplet, *godo.Response, error) {
				expected := &godo.DropletCreateRequest{
					Name:     "droplet",
					Image:    godo.DropletCreateImage{Slug: "image"},
					Region:   "dev0",
					Size:     "1gb",
					UserData: "#cloud-config",
					SSHKeys:  []godo.DropletCreateSSHKey{},
				}

				assert.Equal(t, cr, expected, "create requests did not match")

				return &testDroplet, nil, nil
			},
		},
	}

	withTestClient(client, func(c *TestConfig) {
		ns := "test"
		c.Set(ns, doit.ArgRegionSlug, "dev0")
		c.Set(ns, doit.ArgSizeSlug, "1gb")
		c.Set(ns, doit.ArgImage, "image")
		c.Set(ns, doit.ArgUserData, "#cloud-config")

		err := RunDropletCreate(ns, c, ioutil.Discard, []string{"droplet"})
		assert.NoError(t, err)
	})
}

func TestDropletCreateUserDataFile(t *testing.T) {
	userData, err := ioutil.ReadFile("../testdata/cloud-config.yml")
	if err != nil {
		t.Fatal(err)
	}

	client := &godo.Client{
		Droplets: &doit.DropletsServiceMock{
			CreateFn: func(cr *godo.DropletCreateRequest) (*godo.Droplet, *godo.Response, error) {
				expected := &godo.DropletCreateRequest{
					Name:     "droplet",
					Image:    godo.DropletCreateImage{Slug: "image"},
					Region:   "dev0",
					Size:     "1gb",
					UserData: string(userData),
					SSHKeys:  []godo.DropletCreateSSHKey{},
				}

				assert.Equal(t, cr, expected, "create requests did not match")

				return &testDroplet, nil, nil
			},
		},
	}

	withTestClient(client, func(c *TestConfig) {
		ns := "test"

		c.Set(ns, doit.ArgRegionSlug, "dev0")
		c.Set(ns, doit.ArgSizeSlug, "1gb")
		c.Set(ns, doit.ArgImage, "image")
		c.Set(ns, doit.ArgUserDataFile, "../testdata/cloud-config.yml")

		err := RunDropletCreate(ns, c, ioutil.Discard, []string{"droplet"})
		assert.NoError(t, err)
	})
}

func TestDropletDelete(t *testing.T) {
	client := &godo.Client{
		Droplets: &doit.DropletsServiceMock{
			DeleteFn: func(id int) (*godo.Response, error) {
				assert.Equal(t, id, testDroplet.ID, "droplet ids did not match")
				return nil, nil
			},
		},
	}

	withTestClient(client, func(c *TestConfig) {
		ns := "test"

		err := RunDropletDelete(ns, c, ioutil.Discard, []string{strconv.Itoa(testDroplet.ID)})
		assert.NoError(t, err)
	})
}

func TestDropletGet(t *testing.T) {
	client := &godo.Client{
		Droplets: &doit.DropletsServiceMock{
			GetFn: func(id int) (*godo.Droplet, *godo.Response, error) {
				assert.Equal(t, id, testDroplet.ID, "droplet ids did not match")
				return &testDroplet, nil, nil
			},
		},
	}

	withTestClient(client, func(c *TestConfig) {
		ns := "test"

		err := RunDropletGet(ns, c, ioutil.Discard, []string{strconv.Itoa(testDroplet.ID)})
		assert.NoError(t, err)
	})
}

func TestDropletKernelList(t *testing.T) {
	client := &godo.Client{
		Droplets: &doit.DropletsServiceMock{
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

	withTestClient(client, func(c *TestConfig) {
		ns := "test"

		err := RunDropletKernels(ns, c, ioutil.Discard, []string{"1"})
		assert.NoError(t, err)
	})
}

func TestDropletNeighbors(t *testing.T) {
	didRun := false
	client := &godo.Client{
		Droplets: &doit.DropletsServiceMock{
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

	withTestClient(client, func(c *TestConfig) {
		ns := "test"

		err := RunDropletNeighbors(ns, c, ioutil.Discard, []string{"1"})
		assert.NoError(t, err)
		assert.True(t, didRun)
	})
}

func TestDropletSnapshotList(t *testing.T) {
	client := &godo.Client{
		Droplets: &doit.DropletsServiceMock{
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

	withTestClient(client, func(c *TestConfig) {
		ns := "test"

		err := RunDropletSnapshots(ns, c, ioutil.Discard, []string{"1"})
		assert.NoError(t, err)
	})
}

func TestDropletsList(t *testing.T) {
	didRun := false
	client := &godo.Client{
		Droplets: &doit.DropletsServiceMock{
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

	withTestClient(client, func(c *TestConfig) {
		ns := "test"
		err := RunDropletList(ns, c, ioutil.Discard, []string{})
		assert.NoError(t, err)
		assert.True(t, didRun)
	})
}

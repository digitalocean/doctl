package doit

import (
	"flag"
	"testing"

	"github.com/codegangsta/cli"
	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"
)

var (
	testImage     = godo.Image{ID: 1, Slug: "slug"}
	testImageList = []godo.Image{testImage}
)

func TestImagesList(t *testing.T) {
	didRun := false

	client := &godo.Client{
		Images: &ImagesServiceMock{
			ListFn: func(opts *godo.ListOptions) ([]godo.Image, *godo.Response, error) {
				didRun = true

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

	withinTest(cs, fs, func(c *cli.Context) {
		ImagesList(c)
		assert.True(t, didRun)
	})
}

func TestImagesListDistribution(t *testing.T) {
	didRun := false

	client := &godo.Client{
		Images: &ImagesServiceMock{
			ListDistributionFn: func(opts *godo.ListOptions) ([]godo.Image, *godo.Response, error) {
				didRun = true

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

	withinTest(cs, fs, func(c *cli.Context) {
		ImagesListDistribution(c)
		assert.True(t, didRun)
	})
}

func TestImagesListApplication(t *testing.T) {
	didRun := false

	client := &godo.Client{
		Images: &ImagesServiceMock{
			ListApplicationFn: func(opts *godo.ListOptions) ([]godo.Image, *godo.Response, error) {
				didRun = true

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

	withinTest(cs, fs, func(c *cli.Context) {
		ImagesListApplication(c)
		assert.True(t, didRun)
	})
}

func TestImagesListUser(t *testing.T) {
	didRun := false

	client := &godo.Client{
		Images: &ImagesServiceMock{
			ListUserFn: func(opts *godo.ListOptions) ([]godo.Image, *godo.Response, error) {
				didRun = true

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

	withinTest(cs, fs, func(c *cli.Context) {
		ImagesListUser(c)
		assert.True(t, didRun)
	})
}

func TestImagesGetByID(t *testing.T) {
	client := &godo.Client{
		Images: &ImagesServiceMock{
			GetByIDFn: func(id int) (*godo.Image, *godo.Response, error) {
				assert.Equal(t, id, testImage.ID, "image id not equal")
				return &testImage, nil, nil
			},
			GetBySlugFn: func(slug string) (*godo.Image, *godo.Response, error) {
				t.Error("should not try to load slug")
				return nil, nil, nil
			},
		},
	}

	cs := NewTestConfig(client)
	fs := flag.NewFlagSet("flag set", 0)
	fs.Int(ArgImage, testImage.ID, ArgImage)

	withinTest(cs, fs, func(c *cli.Context) {
		ImagesGet(c)
	})
}

func TestImagesGetBySlug(t *testing.T) {
	client := &godo.Client{
		Images: &ImagesServiceMock{
			GetByIDFn: func(id int) (*godo.Image, *godo.Response, error) {
				t.Error("should not try to load id")
				return nil, nil, nil
			},
			GetBySlugFn: func(slug string) (*godo.Image, *godo.Response, error) {
				assert.Equal(t, slug, testImage.Slug, "image id not equal")
				return &testImage, nil, nil
			},
		},
	}

	cs := NewTestConfig(client)
	fs := flag.NewFlagSet("flag set", 0)
	fs.String(ArgImage, testImage.Slug, ArgImage)

	withinTest(cs, fs, func(c *cli.Context) {
		ImagesGet(c)
	})
}

func TestImagesNoID(t *testing.T) {
	client := &godo.Client{
		Images: &ImagesServiceMock{
			GetByIDFn: func(id int) (*godo.Image, *godo.Response, error) {
				t.Error("should not try to load id")
				return nil, nil, nil
			},
			GetBySlugFn: func(slug string) (*godo.Image, *godo.Response, error) {
				t.Error("should not try to load slug")
				return &testImage, nil, nil
			},
		},
	}

	cs := NewTestConfig(client)
	fs := flag.NewFlagSet("flag set", 0)

	withinTest(cs, fs, func(c *cli.Context) {
		ImagesGet(c)
	})
}

func TestImagesUpdate(t *testing.T) {
	client := &godo.Client{
		Images: &ImagesServiceMock{
			UpdateFn: func(id int, req *godo.ImageUpdateRequest) (*godo.Image, *godo.Response, error) {
				expected := &godo.ImageUpdateRequest{Name: "new-name"}
				assert.Equal(t, req, expected)
				assert.Equal(t, id, testImage.ID)

				return &testImage, nil, nil
			},
		},
	}

	cs := NewTestConfig(client)
	fs := flag.NewFlagSet("flag set", 0)
	fs.Int(ArgImageID, testImage.ID, ArgImageID)
	fs.String(ArgImageName, "new-name", ArgImageName)

	withinTest(cs, fs, func(c *cli.Context) {
		ImagesUpdate(c)
	})
}

func TestImagesDelete(t *testing.T) {
	client := &godo.Client{
		Images: &ImagesServiceMock{
			DeleteFn: func(id int) (*godo.Response, error) {
				assert.Equal(t, id, testImage.ID)
				return nil, nil
			},
		},
	}

	cs := NewTestConfig(client)
	fs := flag.NewFlagSet("flag set", 0)
	fs.Int(ArgImageID, testImage.ID, ArgImageID)

	withinTest(cs, fs, func(c *cli.Context) {
		ImagesDelete(c)
	})
}

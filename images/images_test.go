package images

import (
	"flag"
	"testing"

	"github.com/bryanl/docli"
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
		Images: &docli.ImagesServiceMock{
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

	cs := &docli.TestClientSource{client}
	fs := flag.NewFlagSet("flag set", 0)

	docli.WithinTest(cs, fs, func(c *cli.Context) {
		List(c)
		assert.True(t, didRun)
	})
}

func TestImagesListDistribution(t *testing.T) {
	didRun := false

	client := &godo.Client{
		Images: &docli.ImagesServiceMock{
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

	cs := &docli.TestClientSource{client}
	fs := flag.NewFlagSet("flag set", 0)

	docli.WithinTest(cs, fs, func(c *cli.Context) {
		ListDistribution(c)
		assert.True(t, didRun)
	})
}

func TestImagesListApplication(t *testing.T) {
	didRun := false

	client := &godo.Client{
		Images: &docli.ImagesServiceMock{
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

	cs := &docli.TestClientSource{client}
	fs := flag.NewFlagSet("flag set", 0)

	docli.WithinTest(cs, fs, func(c *cli.Context) {
		ListApplication(c)
		assert.True(t, didRun)
	})
}

func TestImagesListUser(t *testing.T) {
	didRun := false

	client := &godo.Client{
		Images: &docli.ImagesServiceMock{
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

	cs := &docli.TestClientSource{client}
	fs := flag.NewFlagSet("flag set", 0)

	docli.WithinTest(cs, fs, func(c *cli.Context) {
		ListUser(c)
		assert.True(t, didRun)
	})
}

func TestImagesGetByID(t *testing.T) {
	client := &godo.Client{
		Images: &docli.ImagesServiceMock{
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

	cs := &docli.TestClientSource{client}
	fs := flag.NewFlagSet("flag set", 0)
	fs.Int(argImage, testImage.ID, argImage)

	docli.WithinTest(cs, fs, func(c *cli.Context) {
		Get(c)
	})
}

func TestImagesGetBySlug(t *testing.T) {
	client := &godo.Client{
		Images: &docli.ImagesServiceMock{
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

	cs := &docli.TestClientSource{client}
	fs := flag.NewFlagSet("flag set", 0)
	fs.String(argImage, testImage.Slug, argImage)

	docli.WithinTest(cs, fs, func(c *cli.Context) {
		Get(c)
	})
}

func TestImagesUpdate(t *testing.T) {
	client := &godo.Client{
		Images: &docli.ImagesServiceMock{
			UpdateFn: func(id int, req *godo.ImageUpdateRequest) (*godo.Image, *godo.Response, error) {
				expected := &godo.ImageUpdateRequest{Name: "new-name"}
				assert.Equal(t, req, expected)
				assert.Equal(t, id, testImage.ID)

				return &testImage, nil, nil
			},
		},
	}

	cs := &docli.TestClientSource{client}
	fs := flag.NewFlagSet("flag set", 0)
	fs.Int(argImageID, testImage.ID, argImageID)
	fs.String(argImageName, "new-name", argImageName)

	docli.WithinTest(cs, fs, func(c *cli.Context) {
		Update(c)
	})
}

func TestImagesDelete(t *testing.T) {
	client := &godo.Client{
		Images: &docli.ImagesServiceMock{
			DeleteFn: func(id int) (*godo.Response, error) {
				assert.Equal(t, id, testImage.ID)
				return nil, nil
			},
		},
	}

	cs := &docli.TestClientSource{client}
	fs := flag.NewFlagSet("flag set", 0)
	fs.Int(argImageID, testImage.ID, argImageID)

	docli.WithinTest(cs, fs, func(c *cli.Context) {
		Delete(c)
	})
}

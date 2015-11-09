package commands

import (
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/bryanl/doit"
	"github.com/bryanl/doit/Godeps/_workspace/src/github.com/digitalocean/godo"
	"github.com/bryanl/doit/Godeps/_workspace/src/github.com/stretchr/testify/assert"
)

func TestImagesList(t *testing.T) {
	didRun := false

	client := &godo.Client{
		Images: &doit.ImagesServiceMock{
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

	withTestClient(client, func(c *TestConfig) {
		ns := "test"
		RunImagesList(ns, ioutil.Discard)
	})
}

func TestImagesListDistribution(t *testing.T) {
	didRun := false

	client := &godo.Client{
		Images: &doit.ImagesServiceMock{
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

	withTestClient(client, func(c *TestConfig) {
		ns := "test"
		RunImagesListDistribution(ns, ioutil.Discard)
		assert.True(t, didRun)
	})
}

func TestImagesListApplication(t *testing.T) {
	didRun := false

	client := &godo.Client{
		Images: &doit.ImagesServiceMock{
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

	withTestClient(client, func(c *TestConfig) {
		ns := "test"
		RunImagesListApplication(ns, ioutil.Discard)
		assert.True(t, didRun)
	})
}

func TestImagesListUser(t *testing.T) {
	didRun := false

	client := &godo.Client{
		Images: &doit.ImagesServiceMock{
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

	withTestClient(client, func(c *TestConfig) {
		ns := "test"
		RunImagesListUser(ns, ioutil.Discard)
		assert.True(t, didRun)
	})
}

func TestImagesGetByID(t *testing.T) {
	client := &godo.Client{
		Images: &doit.ImagesServiceMock{
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

	withTestClient(client, func(c *TestConfig) {
		ns := "test"
		c.Set(ns, doit.ArgImage, testImage.ID)

		RunImagesGet(ns, ioutil.Discard)
	})
}

func TestImagesGetBySlug(t *testing.T) {
	client := &godo.Client{
		Images: &doit.ImagesServiceMock{
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

	withTestClient(client, func(c *TestConfig) {
		ns := "test"
		c.Set(ns, doit.ArgImage, testImage.Slug)

		RunImagesGet(ns, ioutil.Discard)
	})
}

func TestImagesNoID(t *testing.T) {
	client := &godo.Client{
		Images: &doit.ImagesServiceMock{
			GetByIDFn: func(id int) (*godo.Image, *godo.Response, error) {
				t.Error("should not try to load id")
				return nil, nil, fmt.Errorf("not here")
			},
			GetBySlugFn: func(slug string) (*godo.Image, *godo.Response, error) {
				t.Error("should not try to load slug")
				return nil, nil, fmt.Errorf("not here")
			},
		},
	}

	withTestClient(client, func(c *TestConfig) {
		ns := "test"
		RunImagesGet(ns, ioutil.Discard)
	})
}

func TestImagesUpdate(t *testing.T) {
	client := &godo.Client{
		Images: &doit.ImagesServiceMock{
			UpdateFn: func(id int, req *godo.ImageUpdateRequest) (*godo.Image, *godo.Response, error) {
				expected := &godo.ImageUpdateRequest{Name: "new-name"}
				assert.Equal(t, req, expected)
				assert.Equal(t, id, testImage.ID)

				return &testImage, nil, nil
			},
		},
	}

	withTestClient(client, func(c *TestConfig) {
		ns := "test"
		c.Set(ns, doit.ArgImageID, testImage.ID)
		c.Set(ns, doit.ArgImageName, "new-name")

		RunImagesUpdate(ns, ioutil.Discard)
	})
}

func TestImagesDelete(t *testing.T) {
	client := &godo.Client{
		Images: &doit.ImagesServiceMock{
			DeleteFn: func(id int) (*godo.Response, error) {
				assert.Equal(t, id, testImage.ID)
				return nil, nil
			},
		},
	}

	withTestClient(client, func(c *TestConfig) {
		ns := "test"
		c.Set(ns, doit.ArgImageID, testImage.ID)

		RunImagesDelete(ns, ioutil.Discard)
	})

}

package commands

import (
	"fmt"
	"io/ioutil"
	"strconv"
	"testing"

	"github.com/bryanl/doit"
	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"
)

func TestImageCommand(t *testing.T) {
	cmd := Images()
	assert.NotNil(t, cmd)
	assertCommandNames(t, cmd, "delete", "get", "list", "list-application", "list-distribution", "list-user", "update")
}

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
		config := &cmdConfig{
			ns:         "test",
			doitConfig: c,
			out:        ioutil.Discard,
		}

		RunImagesList(config)
		assert.True(t, didRun)
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
		config := &cmdConfig{
			ns:         "test",
			doitConfig: c,
			out:        ioutil.Discard,
		}

		RunImagesListDistribution(config)
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
		config := &cmdConfig{
			ns:         "test",
			doitConfig: c,
			out:        ioutil.Discard,
		}

		RunImagesListApplication(config)
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
		config := &cmdConfig{
			ns:         "test",
			doitConfig: c,
			out:        ioutil.Discard,
		}

		RunImagesListUser(config)
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
		config := &cmdConfig{
			ns:         "test",
			doitConfig: c,
			out:        ioutil.Discard,
		}

		RunImagesGet(config)
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
		config := &cmdConfig{
			ns:         "test",
			doitConfig: c,
			out:        ioutil.Discard,
		}

		c.Set(config.ns, doit.ArgImage, testImage.Slug)

		RunImagesGet(config)
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
		config := &cmdConfig{
			ns:         "test",
			doitConfig: c,
			out:        ioutil.Discard,
		}

		RunImagesGet(config)
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
		config := &cmdConfig{
			ns:         "test",
			doitConfig: c,
			out:        ioutil.Discard,
		}

		config.args = append(config.args, strconv.Itoa(testImage.ID))

		c.Set(config.ns, doit.ArgImageName, "new-name")

		RunImagesUpdate(config)
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
		config := &cmdConfig{
			ns:         "test",
			doitConfig: c,
			out:        ioutil.Discard,
		}

		config.args = append(config.args, strconv.Itoa(testImage.ID))

		RunImagesDelete(config)
	})

}

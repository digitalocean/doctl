package commands

import (
	"fmt"
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

	withTestClient(client, func(config *cmdConfig) {
		err := RunImagesList(config)
		assert.True(t, didRun)
		assert.NoError(t, err)
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

	withTestClient(client, func(config *cmdConfig) {
		err := RunImagesListDistribution(config)
		assert.True(t, didRun)
		assert.NoError(t, err)
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

	withTestClient(client, func(config *cmdConfig) {
		err := RunImagesListApplication(config)
		assert.True(t, didRun)
		assert.NoError(t, err)
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

	withTestClient(client, func(config *cmdConfig) {
		err := RunImagesListUser(config)
		assert.True(t, didRun)
		assert.NoError(t, err)
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

	withTestClient(client, func(config *cmdConfig) {
		config.args = append(config.args, strconv.Itoa(testImage.ID))
		err := RunImagesGet(config)
		assert.NoError(t, err)
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

	withTestClient(client, func(config *cmdConfig) {
		config.args = append(config.args, testImage.Slug)
		err := RunImagesGet(config)
		assert.NoError(t, err)
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

	withTestClient(client, func(config *cmdConfig) {
		err := RunImagesGet(config)
		assert.Error(t, err)
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

	withTestClient(client, func(config *cmdConfig) {
		config.args = append(config.args, strconv.Itoa(testImage.ID))
		config.doitConfig.Set(config.ns, doit.ArgImageName, "new-name")
		err := RunImagesUpdate(config)
		assert.NoError(t, err)
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

	withTestClient(client, func(config *cmdConfig) {
		config.args = append(config.args, strconv.Itoa(testImage.ID))

		err := RunImagesDelete(config)
		assert.NoError(t, err)
	})

}

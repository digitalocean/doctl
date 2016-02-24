package commands

import (
	"strconv"
	"testing"

	"github.com/bryanl/doit"
	domocks "github.com/bryanl/doit/do/mocks"
	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"
)

func TestImageCommand(t *testing.T) {
	cmd := Images()
	assert.NotNil(t, cmd)
	assertCommandNames(t, cmd, "delete", "get", "list", "list-application", "list-distribution", "list-user", "update")
}

func TestImagesList(t *testing.T) {
	withTestClient(func(config *cmdConfig) {
		is := &domocks.ImagesService{}
		config.is = is

		is.On("List", false).Return(testImageList, nil)

		err := RunImagesList(config)
		assert.NoError(t, err)
	})
}

func TestImagesListDistribution(t *testing.T) {
	withTestClient(func(config *cmdConfig) {
		is := &domocks.ImagesService{}
		config.is = is

		is.On("ListDistribution", false).Return(testImageList, nil)

		err := RunImagesListDistribution(config)
		assert.NoError(t, err)
	})
}

func TestImagesListApplication(t *testing.T) {
	withTestClient(func(config *cmdConfig) {
		is := &domocks.ImagesService{}
		config.is = is

		is.On("ListApplication", false).Return(testImageList, nil)

		err := RunImagesListApplication(config)
		assert.NoError(t, err)
	})
}

func TestImagesListUser(t *testing.T) {
	withTestClient(func(config *cmdConfig) {
		is := &domocks.ImagesService{}
		config.is = is

		is.On("ListUser", false).Return(testImageList, nil)

		err := RunImagesListUser(config)
		assert.NoError(t, err)
	})
}

func TestImagesGetByID(t *testing.T) {
	withTestClient(func(config *cmdConfig) {
		is := &domocks.ImagesService{}
		config.is = is

		is.On("GetByID", testImage.ID).Return(&testImage, nil)

		config.args = append(config.args, strconv.Itoa(testImage.ID))
		err := RunImagesGet(config)
		assert.NoError(t, err)
	})
}

func TestImagesGetBySlug(t *testing.T) {
	withTestClient(func(config *cmdConfig) {
		is := &domocks.ImagesService{}
		config.is = is

		is.On("GetBySlug", testImage.Slug).Return(&testImage, nil)

		config.args = append(config.args, testImage.Slug)
		err := RunImagesGet(config)
		assert.NoError(t, err)
	})
}

func TestImagesNoID(t *testing.T) {
	withTestClient(func(config *cmdConfig) {
		is := &domocks.ImagesService{}
		config.is = is

		err := RunImagesGet(config)
		assert.Error(t, err)
	})
}

func TestImagesUpdate(t *testing.T) {
	withTestClient(func(config *cmdConfig) {
		is := &domocks.ImagesService{}
		config.is = is

		iur := &godo.ImageUpdateRequest{Name: "new-name"}
		is.On("Update", testImage.ID, iur).Return(&testImage, nil)

		config.args = append(config.args, strconv.Itoa(testImage.ID))
		config.doitConfig.Set(config.ns, doit.ArgImageName, "new-name")
		err := RunImagesUpdate(config)
		assert.NoError(t, err)
	})
}

func TestImagesDelete(t *testing.T) {
	withTestClient(func(config *cmdConfig) {
		is := &domocks.ImagesService{}
		config.is = is

		is.On("Delete", testImage.ID).Return(nil)

		config.args = append(config.args, strconv.Itoa(testImage.ID))

		err := RunImagesDelete(config)
		assert.NoError(t, err)
	})

}

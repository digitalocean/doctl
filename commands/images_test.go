package commands

import (
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
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.images.On("List", false).Return(testImageList, nil)

		err := RunImagesList(config)
		assert.NoError(t, err)
	})
}

func TestImagesListDistribution(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.images.On("ListDistribution", false).Return(testImageList, nil)

		err := RunImagesListDistribution(config)
		assert.NoError(t, err)
	})
}

func TestImagesListApplication(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.images.On("ListApplication", false).Return(testImageList, nil)

		err := RunImagesListApplication(config)
		assert.NoError(t, err)
	})
}

func TestImagesListUser(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.images.On("ListUser", false).Return(testImageList, nil)

		err := RunImagesListUser(config)
		assert.NoError(t, err)
	})
}

func TestImagesGetByID(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.images.On("GetByID", testImage.ID).Return(&testImage, nil)

		config.Args = append(config.Args, strconv.Itoa(testImage.ID))
		err := RunImagesGet(config)
		assert.NoError(t, err)
	})
}

func TestImagesGetBySlug(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.images.On("GetBySlug", testImage.Slug).Return(&testImage, nil)

		config.Args = append(config.Args, testImage.Slug)
		err := RunImagesGet(config)
		assert.NoError(t, err)
	})
}

func TestImagesNoID(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		err := RunImagesGet(config)
		assert.Error(t, err)
	})
}

func TestImagesUpdate(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		iur := &godo.ImageUpdateRequest{Name: "new-name"}
		tm.images.On("Update", testImage.ID, iur).Return(&testImage, nil)

		config.Args = append(config.Args, strconv.Itoa(testImage.ID))
		config.Doit.Set(config.NS, doit.ArgImageName, "new-name")
		err := RunImagesUpdate(config)
		assert.NoError(t, err)
	})
}

func TestImagesDelete(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.images.On("Delete", testImage.ID).Return(nil)

		config.Args = append(config.Args, strconv.Itoa(testImage.ID))

		err := RunImagesDelete(config)
		assert.NoError(t, err)
	})

}

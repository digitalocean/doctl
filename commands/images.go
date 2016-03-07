package commands

import (
	"fmt"
	"os"
	"strconv"

	"github.com/bryanl/doit"
	"github.com/bryanl/doit/do"
	"github.com/digitalocean/godo"
	"github.com/spf13/cobra"
)

// Images creates an image command.
func Images() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "image",
		Short: "image commands",
		Long:  "image commands",
	}

	out := os.Stdout

	cmdImagesList := cmdBuilder(cmd, RunImagesList, "list", "list images", out, displayerType(&image{}))
	addBoolFlag(cmdImagesList, doit.ArgImagePublic, false, "List public images")

	cmdImagesListDistribution := cmdBuilder(cmd, RunImagesListDistribution,
		"list-distribution", "list distribution images", out, displayerType(&image{}))
	addBoolFlag(cmdImagesListDistribution, doit.ArgImagePublic, false, "List public images")

	cmdImagesListApplication := cmdBuilder(cmd, RunImagesListApplication,
		"list-application", "list application images", out, displayerType(&image{}))
	addBoolFlag(cmdImagesListApplication, doit.ArgImagePublic, false, "List public images")

	cmdImagesListUser := cmdBuilder(cmd, RunImagesListDistribution,
		"list-user", "list user images", out, displayerType(&image{}))
	addBoolFlag(cmdImagesListUser, doit.ArgImagePublic, false, "List public images")

	cmdBuilder(cmd, RunImagesGet, "get <image-id|image-slug>", "Get image", out, displayerType(&image{}))

	cmdImagesUpdate := cmdBuilder(cmd, RunImagesUpdate, "update <image-id>", "Update image", out, displayerType(&image{}))
	addStringFlag(cmdImagesUpdate, doit.ArgImageName, "", "Image name", requiredOpt())

	cmdBuilder(cmd, RunImagesDelete, "delete <image-id>", "Delete image", out)

	return cmd
}

// RunImagesList images.
func RunImagesList(c *cmdConfig) error {
	is := c.images()

	public, err := c.doitConfig.GetBool(c.ns, doit.ArgImagePublic)
	if err != nil {
		return err
	}

	list, err := is.List(public)
	if err != nil {
		return err
	}

	item := &image{images: list}
	return c.display(item)
}

// RunImagesListDistribution lists distributions that are available.
func RunImagesListDistribution(c *cmdConfig) error {
	is := c.images()

	public, err := c.doitConfig.GetBool(c.ns, doit.ArgImagePublic)
	if err != nil {
		return err
	}

	list, err := is.ListDistribution(public)
	if err != nil {
		return err
	}

	item := &image{images: list}
	return c.display(item)

}

// RunImagesListApplication lists application iamges.
func RunImagesListApplication(c *cmdConfig) error {
	is := c.images()

	public, err := c.doitConfig.GetBool(c.ns, doit.ArgImagePublic)
	if err != nil {
		return err
	}

	list, err := is.ListApplication(public)
	if err != nil {
		return err
	}

	item := &image{images: list}
	return c.display(item)
}

// RunImagesListUser lists user images.
func RunImagesListUser(c *cmdConfig) error {
	is := c.images()

	public, err := c.doitConfig.GetBool(c.ns, doit.ArgImagePublic)
	if err != nil {
		return err
	}

	list, err := is.ListUser(public)
	if err != nil {
		return err
	}

	item := &image{images: list}
	return c.display(item)
}

// RunImagesGet retrieves an image by id or slug.
func RunImagesGet(c *cmdConfig) error {
	is := c.images()

	if len(c.args) != 1 {
		return doit.NewMissingArgsErr(c.ns)
	}

	rawID := c.args[0]

	var i *do.Image
	var err error

	if id, cerr := strconv.Atoi(rawID); cerr == nil {
		i, err = is.GetByID(id)
	} else {
		if len(rawID) > 0 {
			i, err = is.GetBySlug(rawID)
		} else {
			err = fmt.Errorf("image identifier is required")
		}
	}

	if err != nil {
		return err
	}

	item := &image{images: do.Images{*i}}
	return c.display(item)
}

// RunImagesUpdate updates an image.
func RunImagesUpdate(c *cmdConfig) error {
	is := c.images()

	if len(c.args) != 1 {
		return doit.NewMissingArgsErr(c.ns)
	}

	id, err := strconv.Atoi(c.args[0])
	if err != nil {
		return err
	}

	name, err := c.doitConfig.GetString(c.ns, doit.ArgImageName)

	req := &godo.ImageUpdateRequest{
		Name: name,
	}

	i, err := is.Update(id, req)
	if err != nil {
		return err
	}

	item := &image{images: do.Images{*i}}
	return c.display(item)
}

// RunImagesDelete deletes an image.
func RunImagesDelete(c *cmdConfig) error {
	is := c.images()

	if len(c.args) != 1 {
		return doit.NewMissingArgsErr(c.ns)
	}

	id, err := strconv.Atoi(c.args[0])
	if err != nil {
		return err
	}

	return is.Delete(id)
}

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

	cmdImagesList := CmdBuilder(cmd, RunImagesList, "list", "list images", out, displayerType(&image{}))
	AddBoolFlag(cmdImagesList, doit.ArgImagePublic, false, "List public images")

	cmdImagesListDistribution := CmdBuilder(cmd, RunImagesListDistribution,
		"list-distribution", "list distribution images", out, displayerType(&image{}))
	AddBoolFlag(cmdImagesListDistribution, doit.ArgImagePublic, false, "List public images")

	cmdImagesListApplication := CmdBuilder(cmd, RunImagesListApplication,
		"list-application", "list application images", out, displayerType(&image{}))
	AddBoolFlag(cmdImagesListApplication, doit.ArgImagePublic, false, "List public images")

	cmdImagesListUser := CmdBuilder(cmd, RunImagesListDistribution,
		"list-user", "list user images", out, displayerType(&image{}))
	AddBoolFlag(cmdImagesListUser, doit.ArgImagePublic, false, "List public images")

	CmdBuilder(cmd, RunImagesGet, "get <image-id|image-slug>", "Get image", out, displayerType(&image{}))

	cmdImagesUpdate := CmdBuilder(cmd, RunImagesUpdate, "update <image-id>", "Update image", out, displayerType(&image{}))
	AddStringFlag(cmdImagesUpdate, doit.ArgImageName, "", "Image name", requiredOpt())

	CmdBuilder(cmd, RunImagesDelete, "delete <image-id>", "Delete image", out)

	return cmd
}

// RunImagesList images.
func RunImagesList(c *CmdConfig) error {
	is := c.Images()

	public, err := c.Doit.GetBool(c.NS, doit.ArgImagePublic)
	if err != nil {
		return err
	}

	list, err := is.List(public)
	if err != nil {
		return err
	}

	item := &image{images: list}
	return c.Display(item)
}

// RunImagesListDistribution lists distributions that are available.
func RunImagesListDistribution(c *CmdConfig) error {
	is := c.Images()

	public, err := c.Doit.GetBool(c.NS, doit.ArgImagePublic)
	if err != nil {
		return err
	}

	list, err := is.ListDistribution(public)
	if err != nil {
		return err
	}

	item := &image{images: list}
	return c.Display(item)

}

// RunImagesListApplication lists application iamges.
func RunImagesListApplication(c *CmdConfig) error {
	is := c.Images()

	public, err := c.Doit.GetBool(c.NS, doit.ArgImagePublic)
	if err != nil {
		return err
	}

	list, err := is.ListApplication(public)
	if err != nil {
		return err
	}

	item := &image{images: list}
	return c.Display(item)
}

// RunImagesListUser lists user images.
func RunImagesListUser(c *CmdConfig) error {
	is := c.Images()

	public, err := c.Doit.GetBool(c.NS, doit.ArgImagePublic)
	if err != nil {
		return err
	}

	list, err := is.ListUser(public)
	if err != nil {
		return err
	}

	item := &image{images: list}
	return c.Display(item)
}

// RunImagesGet retrieves an image by id or slug.
func RunImagesGet(c *CmdConfig) error {
	is := c.Images()

	if len(c.Args) != 1 {
		return doit.NewMissingArgsErr(c.NS)
	}

	rawID := c.Args[0]

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
	return c.Display(item)
}

// RunImagesUpdate updates an image.
func RunImagesUpdate(c *CmdConfig) error {
	is := c.Images()

	if len(c.Args) != 1 {
		return doit.NewMissingArgsErr(c.NS)
	}

	id, err := strconv.Atoi(c.Args[0])
	if err != nil {
		return err
	}

	name, err := c.Doit.GetString(c.NS, doit.ArgImageName)

	req := &godo.ImageUpdateRequest{
		Name: name,
	}

	i, err := is.Update(id, req)
	if err != nil {
		return err
	}

	item := &image{images: do.Images{*i}}
	return c.Display(item)
}

// RunImagesDelete deletes an image.
func RunImagesDelete(c *CmdConfig) error {
	is := c.Images()

	if len(c.Args) != 1 {
		return doit.NewMissingArgsErr(c.NS)
	}

	id, err := strconv.Atoi(c.Args[0])
	if err != nil {
		return err
	}

	return is.Delete(id)
}

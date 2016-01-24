package commands

import (
	"fmt"
	"io"
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
func RunImagesList(ns string, config doit.Config, out io.Writer, args []string) error {
	client := config.GetGodoClient()
	is := do.NewImagesService(client)

	public, err := config.GetBool(ns, doit.ArgImagePublic)
	if err != nil {
		return err
	}

	list, err := is.List(public)
	if err != nil {
		return err
	}

	dc := &outputConfig{
		ns:     ns,
		config: config,
		item:   &image{images: list},
		out:    out,
	}
	return displayOutput(dc)
}

// RunImagesListDistribution lists distributions that are available.
func RunImagesListDistribution(ns string, config doit.Config, out io.Writer, args []string) error {
	client := config.GetGodoClient()
	is := do.NewImagesService(client)

	public, err := config.GetBool(ns, doit.ArgImagePublic)
	if err != nil {
		return err
	}

	list, err := is.ListDistribution(public)
	if err != nil {
		return err
	}

	dc := &outputConfig{
		ns:     ns,
		config: config,
		item:   &image{images: list},
		out:    out,
	}
	return displayOutput(dc)
}

// RunImagesListApplication lists application iamges.
func RunImagesListApplication(ns string, config doit.Config, out io.Writer, args []string) error {
	client := config.GetGodoClient()
	is := do.NewImagesService(client)

	public, err := config.GetBool(ns, doit.ArgImagePublic)
	if err != nil {
		return err
	}

	list, err := is.ListApplication(public)
	if err != nil {
		return err
	}

	dc := &outputConfig{
		ns:     ns,
		config: config,
		item:   &image{images: list},
		out:    out,
	}
	return displayOutput(dc)
}

// RunImagesListUser lists user images.
func RunImagesListUser(ns string, config doit.Config, out io.Writer, args []string) error {
	client := config.GetGodoClient()
	is := do.NewImagesService(client)

	public, err := config.GetBool(ns, doit.ArgImagePublic)
	if err != nil {
		return err
	}

	list, err := is.ListUser(public)
	if err != nil {
		return err
	}

	dc := &outputConfig{
		ns:     ns,
		config: config,
		item:   &image{images: list},
		out:    out,
	}
	return displayOutput(dc)
}

// RunImagesGet retrieves an image by id or slug.
func RunImagesGet(ns string, config doit.Config, out io.Writer, args []string) error {
	client := config.GetGodoClient()
	is := do.NewImagesService(client)

	if len(args) != 1 {
		return doit.NewMissingArgsErr(ns)
	}

	rawID := args[0]

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

	dc := &outputConfig{
		ns:     ns,
		config: config,
		item:   &image{images: do.Images{*i}},
		out:    out,
	}

	return displayOutput(dc)
}

// RunImagesUpdate updates an image.
func RunImagesUpdate(ns string, config doit.Config, out io.Writer, args []string) error {
	client := config.GetGodoClient()
	is := do.NewImagesService(client)

	if len(args) != 1 {
		return doit.NewMissingArgsErr(ns)
	}

	id, err := strconv.Atoi(args[0])
	if err != nil {
		return err
	}

	name, err := config.GetString(ns, doit.ArgImageName)

	req := &godo.ImageUpdateRequest{
		Name: name,
	}

	i, err := is.Update(id, req)
	if err != nil {
		return err
	}

	dc := &outputConfig{
		ns:     ns,
		config: config,
		item:   &image{images: do.Images{*i}},
		out:    out,
	}

	return displayOutput(dc)
}

// RunImagesDelete deletes an image.
func RunImagesDelete(ns string, config doit.Config, out io.Writer, args []string) error {
	client := config.GetGodoClient()
	is := do.NewImagesService(client)

	if len(args) != 1 {
		return doit.NewMissingArgsErr(ns)
	}

	id, err := strconv.Atoi(args[0])
	if err != nil {
		return err
	}

	return is.Delete(id)
}

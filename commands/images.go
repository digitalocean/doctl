package commands

import (
	"fmt"
	"io"
	"os"
	"strconv"

	"github.com/bryanl/doit"
	"github.com/bryanl/doit/Godeps/_workspace/src/github.com/digitalocean/godo"
	"github.com/bryanl/doit/Godeps/_workspace/src/github.com/spf13/cobra"
)

// Images creates an image command.
func Images() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "image",
		Short: "image commands",
		Long:  "image commands",
	}

	out := os.Stdout

	cmdImagesList := cmdBuilder(RunImagesList, "list", "list images", out)
	cmd.AddCommand(cmdImagesList)
	addBoolFlag(cmdImagesList, doit.ArgImagePublic, false, "List public images")

	cmdImagesListDistribution := cmdBuilder(RunImagesListDistribution,
		"list-distribution", "list distribution images", out)
	cmd.AddCommand(cmdImagesListDistribution)
	addBoolFlag(cmdImagesListDistribution, doit.ArgImagePublic, false, "List public images")

	cmdImagesListApplication := cmdBuilder(RunImagesListApplication,
		"list-application", "list application images", out)
	cmd.AddCommand(cmdImagesListApplication)
	addBoolFlag(cmdImagesListApplication, doit.ArgImagePublic, false, "List public images")

	cmdImagesListUser := cmdBuilder(RunImagesListDistribution,
		"list-user", "list user images", out)
	cmd.AddCommand(cmdImagesListUser)
	addBoolFlag(cmdImagesListUser, doit.ArgImagePublic, false, "List public images")

	cmdImagesGet := cmdBuilder(RunImagesGet, "get <image-id|image-slug>", "Get image", out)
	cmd.AddCommand(cmdImagesGet)

	cmdImagesUpdate := cmdBuilder(RunImagesUpdate, "update <image-id>", "Update image", out)
	cmd.AddCommand(cmdImagesUpdate)
	addStringFlag(cmdImagesUpdate, doit.ArgImageName, "", "Image name", requiredOpt())

	cmdImagesDelete := cmdBuilder(RunImagesDelete, "delete <image-id>", "Delete image", out)
	cmd.AddCommand(cmdImagesDelete)

	return cmd
}

// RunImagesList images.
func RunImagesList(ns string, config doit.Config, out io.Writer, args []string) error {
	client := config.GetGodoClient()
	return listImages(ns, config, out, client.Images.List)
}

// RunImagesListDistribution lists distributions that are available.
func RunImagesListDistribution(ns string, config doit.Config, out io.Writer, args []string) error {
	client := config.GetGodoClient()
	return listImages(ns, config, out, client.Images.ListDistribution)
}

// RunImagesListApplication lists application iamges.
func RunImagesListApplication(ns string, config doit.Config, out io.Writer, args []string) error {
	client := config.GetGodoClient()
	return listImages(ns, config, out, client.Images.ListApplication)
}

// RunImagesListUser lists user images.
func RunImagesListUser(ns string, config doit.Config, out io.Writer, args []string) error {
	client := config.GetGodoClient()
	return listImages(ns, config, out, client.Images.ListUser)
}

// RunImagesGet retrieves an image by id or slug.
func RunImagesGet(ns string, config doit.Config, out io.Writer, args []string) error {
	client := config.GetGodoClient()

	if len(args) != 1 {
		return doit.NewMissingArgsErr(ns)
	}

	rawID := args[0]

	var image *godo.Image
	var err error

	if id, cerr := strconv.Atoi(rawID); cerr == nil {
		image, _, err = client.Images.GetByID(id)
	} else {
		if len(rawID) > 0 {
			image, _, err = client.Images.GetBySlug(rawID)
		} else {
			err = fmt.Errorf("image identifier is required")
		}
	}

	if err != nil {
		return err
	}

	return doit.DisplayOutput(image, out)
}

// RunImagesUpdate updates an image.
func RunImagesUpdate(ns string, config doit.Config, out io.Writer, args []string) error {
	client := config.GetGodoClient()

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

	image, _, err := client.Images.Update(id, req)
	if err != nil {
		return err
	}

	return doit.DisplayOutput(image, out)
}

// RunImagesDelete deletes an image.
func RunImagesDelete(ns string, config doit.Config, out io.Writer, args []string) error {
	client := config.GetGodoClient()

	if len(args) != 1 {
		return doit.NewMissingArgsErr(ns)
	}

	id, err := strconv.Atoi(args[0])
	if err != nil {
		return err
	}

	_, err = client.Images.Delete(id)
	return err
}

type listFn func(*godo.ListOptions) ([]godo.Image, *godo.Response, error)

func listImages(ns string, config doit.Config, out io.Writer, lFn listFn) error {
	public, err := config.GetBool(ns, doit.ArgImagePublic)
	if err != nil {
		return err
	}

	fn := func(opt *godo.ListOptions) ([]interface{}, *godo.Response, error) {
		list, resp, err := lFn(opt)
		if err != nil {
			return nil, nil, err
		}

		si := []interface{}{}
		for _, i := range list {
			if (public && i.Public) || !public {
				si = append(si, i)
			}
		}

		return si, resp, err
	}

	si, err := doit.PaginateResp(fn)
	if err != nil {
		return err
	}

	list := make([]godo.Image, len(si))
	for i := range si {
		list[i] = si[i].(godo.Image)
	}

	return doit.DisplayOutput(list, out)
}

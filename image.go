package main

import (
	"errors"
	"log"

	"github.com/digitalocean/doctl/Godeps/_workspace/src/github.com/codegangsta/cli"
	"github.com/digitalocean/doctl/Godeps/_workspace/src/github.com/digitalocean/godo"
)

var ImageCommand = cli.Command{
	Name:    "image",
	Aliases: []string{"i"},
	Usage:   "Image commands.",
	Subcommands: []cli.Command{
		{
			Name:    "delete",
			Aliases: []string{"d"},
			Usage:   "[--id | <name>] Delete an image.",
			Flags: []cli.Flag{
				cli.IntFlag{
					Name:  "id",
					Usage: "ID for Image. (e.g. 1234567)",
				},
			},
			Action: imageDelete,
			Before: imageDeleteBefore,
		},
		{
			Name:    "list",
			Aliases: []string{"l"},
			Usage:   "List images.",
			Flags: []cli.Flag{
				cli.BoolFlag{
					Name:  "applications, apps",
					Usage: "Only list public One-Click application images",
				},
				cli.BoolFlag{
					Name:  "distributions, distros",
					Usage: "Only list public distribution images",
				},
				cli.BoolFlag{
					Name:  "private",
					Usage: "Only private images",
				},
				cli.IntFlag{
					Name:  "page",
					Value: 0,
					Usage: "What number page of images to fetch.",
				},
				cli.IntFlag{
					Name:  "page-size",
					Value: 20,
					Usage: "Number of actions to fetch per page.",
				},
			},
			Action: imageList,
			Before: imageListBefore,
		},
		{
			Name:    "rename",
			Aliases: []string{"r"},
			Usage:   "[--id | <name>] <new name> Rename an image.",
			Flags: []cli.Flag{
				cli.IntFlag{
					Name:  "id",
					Usage: "ID for Image. (e.g. 1234567)",
				},
			},
			Action: imageRename,
			Before: imageRenameBefore,
		},
		{
			Name:    "show",
			Aliases: []string{"s"},
			Usage:   "[--id | <name>] Show information about an image.",
			Flags: []cli.Flag{
				cli.IntFlag{
					Name:  "id",
					Usage: "ID for Image. (e.g. 1234567)",
				},
			},
			Action: imageShow,
			Before: imageShowBefore,
		},
	},
}

func imageDeleteBefore(ctx *cli.Context) error {
	if ctx.Int("id") == 0 && len(ctx.Args()) != 1 {
		return errors.New("Error: Must provide ID or name for an image.")
	}
	return nil
}

func imageDelete(ctx *cli.Context) {
	id := ctx.Int("id")
	if id == 0 {
		image, err := FindImageByName(client, ctx.Args()[0])
		if err != nil {
			log.Fatalf("%s", err)
		} else {
			id = image.ID
		}
	}

	_, err := client.Images.Delete(id)
	if err != nil {
		log.Fatalf("Unable to delete image: %s", err)
	} else {
		log.Print("Image successfully deleted.")
	}
}

func imageListBefore(ctx *cli.Context) error {
	errMsg := "You can only use one of '--applications', '--distributions', or '--private'."
	switch {
	default:
		return nil
	case ctx.BoolT("apps") == true && ctx.BoolT("distros") == true:
		return errors.New(errMsg)
	case ctx.BoolT("apps") == true && ctx.BoolT("private") == true:
		return errors.New(errMsg)
	case ctx.BoolT("distros") == true && ctx.BoolT("private") == true:
		return errors.New(errMsg)
	}
}

func imageList(ctx *cli.Context) {
	opt := &godo.ListOptions{
		Page:    ctx.Int("page"),
		PerPage: ctx.Int("page-size"),
	}

	var imageList []godo.Image
	var err error
	switch {
	default:
		imageList, _, err = client.Images.List(opt)
	case ctx.BoolT("distros") == true:
		imageList, _, err = client.Images.ListDistribution(opt)
	case ctx.BoolT("apps") == true:
		imageList, _, err = client.Images.ListApplication(opt)
	case ctx.BoolT("private") == true:
		imageList, _, err = client.Images.ListUser(opt)
	}

	if err != nil {
		log.Fatalf("Unable to list images: %s", err)
	}

	cliOut := NewCLIOutput()
	defer cliOut.Flush()
	cliOut.Header("OS", "Name", "ID", "Slug", "Regions")
	for _, image := range imageList {
		cliOut.Writeln("%s\t%s\t%d\t%s\t%s\n", image.Distribution, image.Name,
			image.ID, image.Slug, image.Regions)
	}
}

func imageRenameBefore(ctx *cli.Context) error {
	if ctx.Int("id") == 0 && len(ctx.Args()) != 2 {
		return errors.New("Error: Must provide ID or name for an image and its new name.")
	} else if ctx.Int("id") != 0 && len(ctx.Args()) != 1 {
		return errors.New("Error: Must provide a new name for the image.")
	}
	return nil
}

func imageRename(ctx *cli.Context) {
	var newName string
	if ctx.Int("id") != 0 {
		newName = ctx.Args()[0]
	} else {
		newName = ctx.Args()[1]
	}

	id := ctx.Int("id")
	if id == 0 {
		image, err := FindImageByName(client, ctx.Args()[0])
		if err != nil {
			log.Fatalf("%s", err)
		} else {
			id = image.ID
		}
	}

	updateRequest := &godo.ImageUpdateRequest{
		Name: newName,
	}
	image, _, err := client.Images.Update(id, updateRequest)
	if err != nil {
		log.Fatalf("%s", err)
	} else {
		WriteOutput(image)
	}
}

func imageShowBefore(ctx *cli.Context) error {
	if ctx.Int("id") == 0 && len(ctx.Args()) != 1 {
		return errors.New("Error: Must provide ID or name for an image.")
	}
	return nil
}

func imageShow(ctx *cli.Context) {
	id := ctx.Int("id")
	if id == 0 {
		image, err := FindImageByName(client, ctx.Args()[0])
		if err != nil {
			log.Fatalf("%s")
		} else {
			id = image.ID
		}
	}

	image, _, err := client.Images.GetByID(id)
	if err != nil {
		log.Fatalf("%s", err)
	} else {
		WriteOutput(image)
	}
}

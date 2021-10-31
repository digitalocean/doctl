/*
Copyright 2018 The Doctl Authors All rights reserved.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package commands

import (
	"fmt"
	"strconv"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/commands/displayers"
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
	"github.com/spf13/cobra"
)

// Images creates an image command.
func Images() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:   "image",
			Short: "Display commands to manage images",
			Long: `The sub-commands of ` + "`" + `doctl compute image` + "`" + ` manage images. A DigitalOcean image can be used to create a Droplet.

Currently, there are five types of images: snapshots, backups, custom images, distributions, and One-Click Apps.

- Snapshots provide a full copy of an existing Droplet instance taken on demand.
- Backups are similar to snapshots but are created automatically at regular intervals when enabled for a Droplet.
- Custom images are Linux-based virtual machine images that you may upload for use on DigitalOcean. These can be in one of the following formats: raw, qcow2, vhdx, vdi, or vmdk.
- Distributions are the public Linux distributions that are available to be used as a base to create Droplets.
- Applications, or One-Click Apps, are distributions pre-configured with additional software.`,
		},
	}
	imageDetail := `

- The image's ID
- The image's name
- The type of image. This is either ` + "`" + `snapshot` + "`" + `, ` + "`" + `backup` + "`" + `, or ` + "`" + `custom` + "`" + `.
- The distribution of the image. For custom images, this is user defined.
- The image's slug. This is a uniquely identifying string that is associated with each of the DigitalOcean-provided public images. These can be used to reference a public image as an alternative to the numeric id.
- Whether the image is public or not. An image that is public is available to all accounts. A non-public image is only accessible from your account. This is boolean, true or false.
- The region the image is available in. The regions are represented by their identifying slug values.
- The image's creation date, in ISO8601 combined date and time format.
- The minimum Droplet disk size in GB required for a Droplet to use this image.
- The size of the image in GB.
- The description of the image. (optional)
- A status string indicating the state of a custom image. This may be ` + "`" + `NEW` + "`" + `, ` + "`" + `available` + "`" + `, ` + "`" + `pending` + "`" + `, or ` + "`" + `deleted` + "`" + `.
- A string containing information about errors that may occur when importing a custom image.
`
	cmdImagesList := CmdBuilder(cmd, RunImagesList, "list", "List images on your account", `Use this command to list all private images on your account. To list public images, use the `+"`"+`--public`+"`"+` flag. This command returns the following information about each image:`+imageDetail, Writer,
		aliasOpt("ls"), displayerType(&displayers.Image{}))
	AddBoolFlag(cmdImagesList, doctl.ArgImagePublic, "", false, "List public images")

	cmdImagesListDistribution := CmdBuilder(cmd, RunImagesListDistribution,
		"list-distribution", "List available distribution images", `Use this command to list the distribution images available from DigitalOcean. This command returns the following information about each image:`+imageDetail, Writer,
		displayerType(&displayers.Image{}))
	AddBoolFlag(cmdImagesListDistribution, doctl.ArgImagePublic, "", true, "List public images")

	cmdImagesListApplication := CmdBuilder(cmd, RunImagesListApplication,
		"list-application", "List available One-Click Apps", `Use this command to list all public One-Click Apps that are currently available on the DigitalOcean Marketplace. This command returns the following information about each image:`+imageDetail, Writer,
		displayerType(&displayers.Image{}))
	AddBoolFlag(cmdImagesListApplication, doctl.ArgImagePublic, "", true, "List public images")

	cmdImagesListUser := CmdBuilder(cmd, RunImagesListUser,
		"list-user", "List user-created images", `Use this command to list user-created images, such as snapshots or custom images that you have uploaded to your account. This command returns the following information about each image:`+imageDetail, Writer,
		displayerType(&displayers.Image{}))
	AddBoolFlag(cmdImagesListUser, doctl.ArgImagePublic, "", false, "List public images")

	CmdBuilder(cmd, RunImagesGet, "get <image-id|image-slug>", "Retrieve information about an image", `Use this command to get the following information about the specified image:`+imageDetail, Writer,
		displayerType(&displayers.Image{}))

	cmdImagesUpdate := CmdBuilder(cmd, RunImagesUpdate, "update <image-id>", "Update an image's metadata", `Use this command to change an image's metadata, including its name, description, and distribution.`, Writer,
		displayerType(&displayers.Image{}))
	AddStringFlag(cmdImagesUpdate, doctl.ArgImageName, "", "", "Image name", requiredOpt())

	cmdRunImagesDelete := CmdBuilder(cmd, RunImagesDelete, "delete <image-id>", "Permanently delete an image from your account", `This command deletes the specified image from your account. This is irreversible.`, Writer)
	AddBoolFlag(cmdRunImagesDelete, doctl.ArgForce, doctl.ArgShortForce, false, "Force image delete")

	cmdRunImagesCreate := CmdBuilder(cmd, RunImagesCreate, "create <image-name>", "Create custom image", `This command creates an image in your DigitalOcean account. You can specify a URL for the image contents, the region at which to store the image, and image metadata.`, Writer)
	AddStringFlag(cmdRunImagesCreate, doctl.ArgImageExternalURL, "", "", "Custom image retrieval URL", requiredOpt())
	AddStringFlag(cmdRunImagesCreate, doctl.ArgRegionSlug, "", "", "Region slug identifier", requiredOpt())
	AddStringFlag(cmdRunImagesCreate, doctl.ArgImageDistro, "", "Unknown", "Custom image distribution")
	AddStringFlag(cmdRunImagesCreate, doctl.ArgImageDescription, "", "", "Description of image")
	AddStringSliceFlag(cmdRunImagesCreate, doctl.ArgTagNames, "", []string{}, "List of tags applied to image")

	return cmd
}

// RunImagesList images.
func RunImagesList(c *CmdConfig) error {
	is := c.Images()

	public, err := c.Doit.GetBool(c.NS, doctl.ArgImagePublic)
	if err != nil {
		return err
	}

	list, err := is.List(public)
	if err != nil {
		return err
	}

	if !public && len(list) < 1 {
		notice("Listing private images. Use '--public' to include all images.")
	}

	item := &displayers.Image{Images: list}
	return c.Display(item)
}

// RunImagesListDistribution lists distributions that are available.
func RunImagesListDistribution(c *CmdConfig) error {
	is := c.Images()

	public, err := c.Doit.GetBool(c.NS, doctl.ArgImagePublic)
	if err != nil {
		return err
	}

	list, err := is.ListDistribution(public)
	if err != nil {
		return err
	}

	item := &displayers.Image{Images: list}
	return c.Display(item)

}

// RunImagesListApplication lists application images.
func RunImagesListApplication(c *CmdConfig) error {
	is := c.Images()

	public, err := c.Doit.GetBool(c.NS, doctl.ArgImagePublic)
	if err != nil {
		return err
	}

	list, err := is.ListApplication(public)
	if err != nil {
		return err
	}

	item := &displayers.Image{Images: list}
	return c.Display(item)
}

// RunImagesListUser lists user images.
func RunImagesListUser(c *CmdConfig) error {
	is := c.Images()

	public, err := c.Doit.GetBool(c.NS, doctl.ArgImagePublic)
	if err != nil {
		return err
	}

	list, err := is.ListUser(public)
	if err != nil {
		return err
	}

	item := &displayers.Image{Images: list}
	return c.Display(item)
}

// RunImagesGet retrieves an image by id or slug.
func RunImagesGet(c *CmdConfig) error {
	is := c.Images()

	err := ensureOneArg(c)
	if err != nil {
		return err
	}

	rawID := c.Args[0]

	var i *do.Image

	if id, cerr := strconv.Atoi(rawID); cerr == nil {
		i, err = is.GetByID(id)
	} else {
		if len(rawID) > 0 {
			i, err = is.GetBySlug(rawID)
		} else {
			err = fmt.Errorf("An image ID is required.")
		}
	}

	if err != nil {
		return err
	}

	item := &displayers.Image{Images: do.Images{*i}}
	return c.Display(item)
}

// RunImagesUpdate updates an image.
func RunImagesUpdate(c *CmdConfig) error {
	is := c.Images()

	err := ensureOneArg(c)
	if err != nil {
		return err
	}

	id, err := strconv.Atoi(c.Args[0])
	if err != nil {
		return err
	}

	name, err := c.Doit.GetString(c.NS, doctl.ArgImageName)
	if err != nil {
		return err
	}

	req := &godo.ImageUpdateRequest{
		Name: name,
	}

	i, err := is.Update(id, req)
	if err != nil {
		return err
	}

	item := &displayers.Image{Images: do.Images{*i}}
	return c.Display(item)
}

// RunImagesDelete deletes an image.
func RunImagesDelete(c *CmdConfig) error {
	is := c.Images()

	if len(c.Args) < 1 {
		return doctl.NewMissingArgsErr(c.NS)
	}

	force, err := c.Doit.GetBool(c.NS, doctl.ArgForce)
	if err != nil {
		return err
	}

	if force || AskForConfirmDelete("image", len(c.Args)) == nil {

		for _, el := range c.Args {
			id, err := strconv.Atoi(el)
			if err != nil {
				return err
			}
			if err := is.Delete(id); err != nil {
				return err
			}
		}

	} else {
		return errOperationAborted
	}

	return nil
}

// RunImagesCreate creates a new custom image.
func RunImagesCreate(c *CmdConfig) error {
	r := new(godo.CustomImageCreateRequest)

	if err := buildCustomImageRequestFromArgs(c, r); err != nil {
		return err
	}

	is := c.Images()
	i, err := is.Create(r)
	if err != nil {
		return err
	}

	item := &displayers.Image{Images: do.Images{*i}}
	return c.Display(item)
}

func buildCustomImageRequestFromArgs(c *CmdConfig, r *godo.CustomImageCreateRequest) error {
	if len(c.Args) != 1 {
		return doctl.NewMissingArgsErr(fmt.Sprintf("%s.%s", c.NS, doctl.ArgImageName))
	}
	name := c.Args[0]

	addr, err := c.Doit.GetString(c.NS, doctl.ArgImageExternalURL)
	if err != nil {
		return err
	}
	region, err := c.Doit.GetString(c.NS, doctl.ArgRegionSlug)
	if err != nil {
		return err
	}
	distro, err := c.Doit.GetString(c.NS, doctl.ArgImageDistro)
	if err != nil {
		return err
	}
	desc, err := c.Doit.GetString(c.NS, doctl.ArgImageDescription)
	if err != nil {
		return err
	}
	tags, err := c.Doit.GetStringSlice(c.NS, doctl.ArgTagNames)
	if err != nil {
		return err
	}

	r.Name = name
	r.Url = addr
	r.Region = region
	r.Distribution = distro
	r.Description = desc
	r.Tags = tags

	return nil
}

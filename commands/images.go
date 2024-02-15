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

Currently, there are five types of images: snapshots, backups, custom images, distributions, and one-click application.

- Snapshots provide a full copy of an existing Droplet instance taken on demand.
- Backups are similar to snapshots but are created automatically at regular intervals when enabled for a Droplet.
- Custom images are Linux-based virtual machine images that you may upload for use on DigitalOcean. We support the following formats: raw, qcow2, vhdx, vdi, or vmdk.
- Distributions are the public Linux distributions that are available to be used as a base to create Droplets.
- Applications, or one-click apps, are distributions pre-configured with additional software, such as WordPress, Django, or Flask.`,
		},
	}
	imageDetail := `

- The image's ID
- The image's name
- The type of image. Possible values: ` + "`" + `snapshot` + "`" + `, ` + "`" + `backup` + "`" + `, ` + "`" + `custom` + "`" + `.
- The distribution of the image. For custom images, this is user defined.
- The image's slug. This is a unique string that identifies each DigitalOcean-provided public image. These can be used to reference a public image as an alternative to the numeric ID.
- Whether the image is public or not. An public image is available to all accounts. A private image is only accessible from your account. This is boolean value, true or false.
- The minimum Droplet disk size required for a Droplet to use this image, in GB.
`
	cmdImagesList := CmdBuilder(cmd, RunImagesList, "list", "List images on your account", `Lists all private images on your account. To list public images, use the `+"`"+`--public`+"`"+` flag. This command returns the following information about each image:`+imageDetail, Writer,
		aliasOpt("ls"), displayerType(&displayers.Image{}))
	AddBoolFlag(cmdImagesList, doctl.ArgImagePublic, "", false, "Lists public images")
	cmdImagesList.Example = `The following example lists all private images on your account and uses the ` + "`" + `--format` + "`" + ` flag to return only the ID, distribution, and slug for each image: doctl compute image list --format ID,Distribution,Slug`

	cmdImagesListDistribution := CmdBuilder(cmd, RunImagesListDistribution,
		"list-distribution", "List available distribution images", `Lists the distribution images available from DigitalOcean. This command returns the following information about each image:`+imageDetail, Writer,
		displayerType(&displayers.Image{}))
	AddBoolFlag(cmdImagesListDistribution, doctl.ArgImagePublic, "", true, "Lists public images")
	cmdImagesListDistribution.Example = `The following example lists all public distribution images available from DigitalOcean and uses the ` + "`" + `--format` + "`" + ` flag to return only the ID, distribution, and slug for each image: doctl compute image list-distribution --format ID,Distribution,Slug`

	cmdImagesListApplication := CmdBuilder(cmd, RunImagesListApplication,
		"list-application", "List available One-Click Apps", `Lists all public one-click apps that are currently available on the DigitalOcean Marketplace. This command returns the following information about each image:`+imageDetail, Writer,
		displayerType(&displayers.Image{}))
	AddBoolFlag(cmdImagesListApplication, doctl.ArgImagePublic, "", true, "Lists public images")
	cmdImagesListApplication.Example = `The following example lists all public One-Click Apps available from DigitalOcean and uses the ` + "`" + `--format` + "`" + ` flag to return only the ID, name, distribution, and slug for each image: doctl compute image list-application --format ID,Name,Distribution,Slug`

	cmdImagesListUser := CmdBuilder(cmd, RunImagesListUser,
		"list-user", "List user-created images", `Use this command to list user-created images, such as snapshots or custom images that you have uploaded to your account. This command returns the following information about each image:`+imageDetail, Writer,
		displayerType(&displayers.Image{}))
	AddBoolFlag(cmdImagesListUser, doctl.ArgImagePublic, "", false, "Lists public images")
	cmdImagesListUser.Example = `The following example lists all user-created images on your account and uses the ` + "`" + `--format` + "`" + ` flag to return only the ID, name, distribution, and slug for each image: doctl compute image list-user --format ID,Name,Distribution,Slug`

	cmdImagesGet := CmdBuilder(cmd, RunImagesGet, "get <image-id|image-slug>", "Retrieve information about an image", `Returns the following information about the specified image:`+imageDetail, Writer,
		displayerType(&displayers.Image{}))
	cmdImagesGet.Example = `The following example retrieves information about an image with the ID ` + "`" + `386734086` + "`" + `: doctl compute image get 386734086`

	cmdImagesUpdate := CmdBuilder(cmd, RunImagesUpdate, "update <image-id>", "Update an image's metadata", `Updates an image's metadata, including its name, description, and distribution.`, Writer,
		displayerType(&displayers.Image{}))
	AddStringFlag(cmdImagesUpdate, doctl.ArgImageName, "", "", "The name of the image to update", requiredOpt())
	cmdImagesUpdate.Example = `The following example updates the name of an image with the ID ` + "`" + `386734086` + "`" + ` to ` + "`" + `New Image Name` + "`" + `: doctl compute image update 386734086 --name "Example Image Name"`

	cmdRunImagesDelete := CmdBuilder(cmd, RunImagesDelete, "delete <image-id>", "Permanently delete an image from your account", `Permanently deletes an image from your account. This is irreversible.`, Writer,
		aliasOpt("rm"))
	AddBoolFlag(cmdRunImagesDelete, doctl.ArgForce, doctl.ArgShortForce, false, "Force image delete")
	cmdRunImagesDelete.Example = `The following example deletes an image with the ID ` + "`" + `386734086` + "`" + `: doctl compute image delete 386734086`

	cmdRunImagesCreate := CmdBuilder(cmd, RunImagesCreate, "create <image-name>", "Create custom image", `Creates an image in your DigitalOcean account. Specify a URL to download the image from and the region to store the image in. You can add additional metadata to the image using the optional flags.`, Writer)
	AddStringFlag(cmdRunImagesCreate, doctl.ArgImageExternalURL, "", "", "The URL to retrieve the image from", requiredOpt())
	AddStringFlag(cmdRunImagesCreate, doctl.ArgRegionSlug, "", "", "The slug of the region you want to store the image in. For a list of region slugs, use the `doctl compute region list` command.", requiredOpt())
	AddStringFlag(cmdRunImagesCreate, doctl.ArgImageDistro, "", "Unknown", "A custom image distribution slug to apply to the image")
	AddStringFlag(cmdRunImagesCreate, doctl.ArgImageDescription, "", "", "An optional description of the image")
	AddStringSliceFlag(cmdRunImagesCreate, doctl.ArgTagNames, "", []string{}, "A list of tag names to apply to the image")
	cmdRunImagesCreate.Example = `The following example creates a custom image named ` + "`" + `Example Image` + "`" + ` from a URL and stores it in the ` + "`" + `nyc1` + "`" + ` region: doctl compute image create "Example Image" --image-url "https://example.com/image.iso" --region nyc1`

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

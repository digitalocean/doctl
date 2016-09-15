/*
Copyright 2016 The Doctl Authors All rights reserved.
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
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
	"github.com/spf13/cobra"
)

// Images creates an image command.
func Images() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:   "image",
			Short: "image commands",
			Long:  "image commands",
		},
		DocCategories: []string{"image"},
		IsIndex:       true,
	}

	cmdImagesList := CmdBuilder(cmd, RunImagesList, "list", "list images", Writer,
		aliasOpt("ls"), displayerType(&image{}), docCategories("image"))
	AddBoolFlag(cmdImagesList, doctl.ArgImagePublic, false, "List public images")

	cmdImagesListDistribution := CmdBuilder(cmd, RunImagesListDistribution,
		"list-distribution", "list distribution images", Writer,
		displayerType(&image{}), docCategories("image"))
	AddBoolFlag(cmdImagesListDistribution, doctl.ArgImagePublic, false, "List public images")

	cmdImagesListApplication := CmdBuilder(cmd, RunImagesListApplication,
		"list-application", "list application images", Writer,
		displayerType(&image{}), docCategories("image"))
	AddBoolFlag(cmdImagesListApplication, doctl.ArgImagePublic, false, "List public images")

	cmdImagesListUser := CmdBuilder(cmd, RunImagesListUser,
		"list-user", "list user images", Writer,
		displayerType(&image{}), docCategories("image"))
	AddBoolFlag(cmdImagesListUser, doctl.ArgImagePublic, false, "List public images")

	CmdBuilder(cmd, RunImagesGet, "get <image-id|image-slug>", "Get image", Writer,
		displayerType(&image{}), docCategories("image"))

	cmdImagesUpdate := CmdBuilder(cmd, RunImagesUpdate, "update <image-id>", "Update image", Writer,
		displayerType(&image{}), docCategories("image"))
	AddStringFlag(cmdImagesUpdate, doctl.ArgImageName, "", "Image name", requiredOpt())

	CmdBuilder(cmd, RunImagesDelete, "delete <image-id>", "Delete image", Writer,
		docCategories("image"))

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

	item := &image{images: list}
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

	item := &image{images: list}
	return c.Display(item)

}

// RunImagesListApplication lists application iamges.
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

	item := &image{images: list}
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

	item := &image{images: list}
	return c.Display(item)
}

// RunImagesGet retrieves an image by id or slug.
func RunImagesGet(c *CmdConfig) error {
	is := c.Images()

	if len(c.Args) != 1 {
		return doctl.NewMissingArgsErr(c.NS)
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
		return doctl.NewMissingArgsErr(c.NS)
	}

	id, err := strconv.Atoi(c.Args[0])
	if err != nil {
		return err
	}

	name, err := c.Doit.GetString(c.NS, doctl.ArgImageName)

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

	if len(c.Args) < 1 {
		return doctl.NewMissingArgsErr(c.NS)
	}

	for _, el := range c.Args {
		id, err := strconv.Atoi(el)
		if err != nil {
			return err
		}
		if err := is.Delete(id); err != nil {
			return err
		}
	}

	return nil
}

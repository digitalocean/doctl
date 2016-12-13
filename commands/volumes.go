package commands

import (
	"fmt"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
	"github.com/dustin/go-humanize"
	"github.com/gobwas/glob"
	"github.com/spf13/cobra"
)

// Volume creates the Volume command
func Volume() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:   "volume",
			Short: "volume commands",
			Long:  "volume is used to access volume commands",
		},
	}

	cmdRunVolumeList := CmdBuilder(cmd, RunVolumeList, "list", "list volume", Writer,
		aliasOpt("ls"), displayerType(&volume{}))
	AddStringFlag(cmdRunVolumeList, doctl.ArgRegionSlug, "", "", "Volume region")

	cmdVolumeCreate := CmdBuilder(cmd, RunVolumeCreate, "create [name]", "create a volume", Writer,
		aliasOpt("c"), displayerType(&volume{}))

	AddStringFlag(cmdVolumeCreate, doctl.ArgVolumeSize, "", "4TiB", "Volume size",
		requiredOpt())
	AddStringFlag(cmdVolumeCreate, doctl.ArgVolumeDesc, "", "", "Volume description")
	AddStringFlag(cmdVolumeCreate, doctl.ArgVolumeRegion, "", "", "Volume region",
		requiredOpt())

	CmdBuilder(cmd, RunVolumeDelete, "delete [ID]", "delete a volume", Writer,
		aliasOpt("rm"))

		CmdBuilder(cmd, RunVolumeGet, "get [ID]", "get a volume", Writer, aliasOpt("g"),
		displayerType(&volume{}))

	return cmd

}

// RunVolumeList returns a list of volumes.
func RunVolumeList(c *CmdConfig) error {

	al := c.Volumes()

	region, err := c.Doit.GetString(c.NS, doctl.ArgRegionSlug)
	if err != nil {
		return nil
	}

	matches := []glob.Glob{}
	for _, globStr := range c.Args {
		g, err := glob.Compile(globStr)
		if err != nil {
			return fmt.Errorf("unknown glob %q", globStr)
		}

		matches = append(matches, g)
	}

	list, err := al.List()
	if err != nil {
		return err
	}
	var matchedList []do.Volume

	for _, volume := range list {
		var skip = true
		if len(matches) == 0 {
			skip = false
		} else {
			for _, m := range matches {
				if m.Match(volume.ID) {
					skip = false
				}
				if m.Match(volume.Name) {
					skip = false
				}
			}
		}

		if !skip && region != "" {
			if region != volume.Region.Slug {
				skip = true
			}
		}

		if !skip {
			matchedList = append(matchedList, volume)
		}
	}
	item := &volume{volumes: matchedList}
	return c.Display(item)
}

// RunVolumeCreate creates a volume.
func RunVolumeCreate(c *CmdConfig) error {
	if len(c.Args) == 0 {
		return doctl.NewMissingArgsErr(c.NS)
	}

	name := c.Args[0]

	sizeStr, err := c.Doit.GetString(c.NS, doctl.ArgVolumeSize)
	if err != nil {
		return err
	}
	size, err := humanize.ParseBytes(sizeStr)
	if err != nil {
		return err
	}

	desc, err := c.Doit.GetString(c.NS, doctl.ArgVolumeDesc)
	if err != nil {
		return err
	}

	region, err := c.Doit.GetString(c.NS, doctl.ArgVolumeRegion)
	if err != nil {
		return err

	}

	var createVolume godo.VolumeCreateRequest

	createVolume.Name = name
	createVolume.SizeGigaBytes = int64(size / (1 << 30))
	createVolume.Description = desc
	createVolume.Region = region

	al := c.Volumes()

	d, err := al.CreateVolume(&createVolume)
	if err != nil {
		return err
	}
	item := &volume{volumes: []do.Volume{*d}}
	return c.Display(item)

}

// RunVolumeDelete deletes a volume.
func RunVolumeDelete(c *CmdConfig) error {
	if len(c.Args) == 0 {
		return doctl.NewMissingArgsErr(c.NS)

	}
	id := c.Args[0]
	al := c.Volumes()
	if err := al.DeleteVolume(id); err != nil {
		return err

	}
	return nil
}

// RunVolumeGet gets a volume.
func RunVolumeGet(c *CmdConfig) error {
	if len(c.Args) == 0 {
		return doctl.NewMissingArgsErr(c.NS)

	}
	id := c.Args[0]
	al := c.Volumes()
	d, err := al.Get(id)
	if err != nil {
		return err
	}
	item := &volume{volumes: []do.Volume{*d}}
	return c.Display(item)
}

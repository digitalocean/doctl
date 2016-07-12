package commands

import (
	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
	"github.com/dustin/go-humanize"
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

	CmdBuilder(cmd, RunVolumeList, "list", "list volume", Writer,
		aliasOpt("ls"), displayerType(&volume{}))

	cmdVolumeCreate := CmdBuilder(cmd, RunVolumeCreate, "create [name]", "create a volume", Writer,
		aliasOpt("c"), displayerType(&volume{}))

	AddStringFlag(cmdVolumeCreate, doctl.ArgVolumeSize, "4TiB", "Volume size",
		requiredOpt())
	AddStringFlag(cmdVolumeCreate, doctl.ArgVolumeDesc, "", "Volume description")
	AddStringFlag(cmdVolumeCreate, doctl.ArgVolumeRegion, "", "Volume region",
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
	d, err := al.List()
	if err != nil {
		return err
	}
	item := &volume{volumes: d}
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

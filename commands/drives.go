package commands

import (
	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
	"github.com/dustin/go-humanize"
	"github.com/spf13/cobra"
)

// Drive creates the Drive command
// NOTE: This command will currently only work for those in the
// block storage private beta on DigitalOcean.
func Drive() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:   "drive",
			Short: "drive commands",
			Long:  "drive is used to access drive commands",
		},
	}

	CmdBuilder(cmd, RunDriveList, "list", "list drive", Writer,
		aliasOpt("ls"), displayerType(&drive{}))

	cmdDriveCreate := CmdBuilder(cmd, RunDriveCreate, "create [name]", "create a drive", Writer,
		aliasOpt("c"), displayerType(&drive{}))

	AddStringFlag(cmdDriveCreate, doctl.ArgDriveSize, "4TiB", "Drive size",
		requiredOpt())
	AddStringFlag(cmdDriveCreate, doctl.ArgDriveDesc, "", "Drive description")
	AddStringFlag(cmdDriveCreate, doctl.ArgDriveRegion, "", "Drive region",
		requiredOpt())

	CmdBuilder(cmd, RunDriveDelete, "delete [ID]", "delete a drive", Writer,
		aliasOpt("rm"))

	CmdBuilder(cmd, RunDriveGet, "get [ID]", "get a drive", Writer, aliasOpt("g"),
		displayerType(&drive{}))

	return cmd

}

// RunDriveList returns a list of drives.
func RunDriveList(c *CmdConfig) error {
	al := c.Drives()
	d, err := al.List()
	if err != nil {
		return err
	}
	item := &drive{drives: d}
	return c.Display(item)
}

// RunDriveCreate creates a drive.
func RunDriveCreate(c *CmdConfig) error {
	if len(c.Args) == 0 {
		return doctl.NewMissingArgsErr(c.NS)
	}

	name := c.Args[0]

	sizeStr, err := c.Doit.GetString(c.NS, doctl.ArgDriveSize)
	if err != nil {
		return err
	}
	size, err := humanize.ParseBytes(sizeStr)
	if err != nil {
		return err
	}

	desc, err := c.Doit.GetString(c.NS, doctl.ArgDriveDesc)
	if err != nil {
		return err
	}

	region, err := c.Doit.GetString(c.NS, doctl.ArgDriveRegion)
	if err != nil {
		return err

	}

	var createDrive godo.DriveCreateRequest

	createDrive.Name = name
	createDrive.SizeGibiBytes = int64(size / (1 << 30))
	createDrive.Description = desc
	createDrive.Region = region

	al := c.Drives()

	d, err := al.CreateDrive(&createDrive)
	if err != nil {
		return err
	}
	item := &drive{drives: []do.Drive{*d}}
	return c.Display(item)

}

// RunDriveDelete deletes a drive.
func RunDriveDelete(c *CmdConfig) error {
	if len(c.Args) == 0 {
		return doctl.NewMissingArgsErr(c.NS)

	}
	id := c.Args[0]
	al := c.Drives()
	if err := al.DeleteDrive(id); err != nil {
		return err

	}
	return nil
}

// RunDriveGet gets a drive.
func RunDriveGet(c *CmdConfig) error {
	if len(c.Args) == 0 {
		return doctl.NewMissingArgsErr(c.NS)

	}
	id := c.Args[0]
	al := c.Drives()
	d, err := al.Get(id)
	if err != nil {
		return err
	}
	item := &drive{drives: []do.Drive{*d}}
	return c.Display(item)
}

package commands

import (
	"github.com/digitalocean/doctl/do"
)

type nsListItem struct {
	ns do.OutputNamespace
}

func (i nsListItem) Title() string {
	return i.ns.Label + " (" + i.ns.Region + ")"
}

func (i nsListItem) Description() string {
	return i.ns.Namespace
}

func (i nsListItem) FilterValue() string {
	return i.ns.Label
}

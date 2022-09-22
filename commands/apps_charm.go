package commands

import (
	"strings"

	"github.com/digitalocean/doctl/commands/charm"
	"github.com/digitalocean/doctl/commands/charm/template"
	"github.com/digitalocean/godo"
)

type componentListItem struct {
	spec godo.AppComponentSpec
}

func (i componentListItem) Title() string {
	return i.spec.GetName()
}

func (i componentListItem) Description() string {
	desc := []string{
		strings.ToLower(charm.SnakeToTitle(i.spec.GetType())) + " component",
	}

	if buildable, ok := i.spec.(godo.AppBuildableComponentSpec); ok {
		if sourceDir := buildable.GetSourceDir(); sourceDir != "" {
			desc = append(desc, template.String(`located in ./{{highlight .}}`, sourceDir))
		}
	}

	return strings.Join(desc, "\n")
}

func (i componentListItem) FilterValue() string {
	return i.spec.GetName()
}

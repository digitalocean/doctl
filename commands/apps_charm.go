package commands

import (
	"strings"

	"github.com/digitalocean/doctl/commands/charm"
	"github.com/digitalocean/doctl/commands/charm/template"
	"github.com/digitalocean/doctl/internal/apps/builder"
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
		if builder.IsDockerBuild(buildable) {
			desc[0] += " [dockerfile]"
		} else if builder.IsCNBBuild(buildable) {
			desc[0] += " [buildpacks]"
		}

		if sourceDir := buildable.GetSourceDir(); sourceDir != "" {
			desc = append(desc, template.String(`located in ./{{highlight .}}`, sourceDir))
		}
	}

	return strings.Join(desc, "\n")
}

func (i componentListItem) FilterValue() string {
	return i.spec.GetName()
}

type appListItem struct {
	*godo.App
}

func (i appListItem) Title() string {
	return i.GetSpec().GetName()
}

func (i appListItem) Description() string {
	desc := []string{}

	if i.LiveDomain != "" {
		desc = append(desc, i.LiveDomain)
	}
	if !i.LastDeploymentActiveAt.IsZero() {
		desc = append(desc, template.String(`last deployed {{timeAgo .}}`, i.LastDeploymentActiveAt))
	} else {
		desc = append(desc, template.String(`created {{timeAgo .}}`, i.CreatedAt))
	}

	return strings.Join(desc, "\n")
}

func (i appListItem) FilterValue() string {
	return i.GetSpec().GetName()
}

package commands

import (
	"fmt"
	"strings"

	"github.com/digitalocean/doctl/commands/charm"
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
		snakeToTitle(string(i.spec.GetType())) + " component",
	}

	if buildable, ok := i.spec.(godo.AppBuildableComponentSpec); ok {
		if sourceDir := buildable.GetSourceDir(); sourceDir != "" {
			desc = append(desc, "located in ./"+charm.TextHighlight.S(sourceDir))
		}
	}

	return strings.Join(desc, "\n")
}
func (i componentListItem) FilterValue() string {
	return i.spec.GetName()
}

func snakeToTitle(s string) string {
	return strings.Title(strings.ReplaceAll(strings.ToLower(fmt.Sprint(s)), "_", " "))
}

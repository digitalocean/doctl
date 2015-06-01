package actions

import (
	"testing"

	"github.com/bryanl/docli/docli"
	"github.com/codegangsta/cli"
	"github.com/digitalocean/godo"
)

var testActionList = []godo.Action{
	{ID: 1},
}

func TestActionList(t *testing.T) {
	actionDidList := false

	client := &godo.Client{
		Actions: &docli.ActionsServiceMock{
			ListFn: func(opts *godo.ListOptions) ([]godo.Action, *godo.Response, error) {
				actionDidList = true
				resp := &godo.Response{
					Links: &godo.Links{
						Pages: &godo.Pages{},
					},
				}
				return testActionList, resp, nil
			},
		},
	}

	cs := &docli.TestClientSource{client}

	docli.WithinTest(cs, func(c *cli.Context) {
		Action(c)
		if !actionDidList {
			t.Errorf("Action() did not run")
		}
	})
}

package commands

import (
	"io/ioutil"
	"testing"

	"github.com/bryanl/doit"
	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"
)

var (
	testAction     = godo.Action{ID: 1, Region: &godo.Region{Slug: "dev0"}}
	testActionList = []godo.Action{
		testAction,
	}
)

func TestActionsCommand(t *testing.T) {
	cmd := Actions()
	assert.NotNil(t, cmd)
	assertCommandNames(t, cmd, "get", "list", "wait")
}

func TestActionList(t *testing.T) {
	actionDidList := false

	client := &godo.Client{
		Actions: &doit.ActionsServiceMock{
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

	withTestClient(client, func(c *TestConfig) {
		config := &cmdConfig{
			ns:         "test",
			doitConfig: c,
			out:        ioutil.Discard,
		}

		err := RunCmdActionList(config)
		assert.NoError(t, err)

		if !actionDidList {
			t.Errorf("Action() did not run")
		}
	})
}

func TestActionGet(t *testing.T) {
	client := &godo.Client{
		Actions: &doit.ActionsServiceMock{
			GetFn: func(id int) (*godo.Action, *godo.Response, error) {
				if got, expected := id, testAction.ID; got != expected {
					t.Errorf("GetFn() called with %d; expected %d", got, expected)
				}
				return &testAction, nil, nil
			},
		},
	}

	withTestClient(client, func(c *TestConfig) {
		config := &cmdConfig{
			ns:         "test",
			doitConfig: c,
			out:        ioutil.Discard,
		}

		config.args = append(config.args, "1")

		err := RunCmdActionGet(config)
		assert.NoError(t, err)
	})
}

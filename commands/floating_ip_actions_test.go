package commands

import (
	"io/ioutil"
	"testing"

	"github.com/bryanl/doit"
	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"
)

func TestFloatingIPActionCommand(t *testing.T) {
	cmd := FloatingIPAction()
	assert.NotNil(t, cmd)
	assertCommandNames(t, cmd, "assign", "get", "unassign")
}

func TestFloatingIPActionsGet(t *testing.T) {
	client := &godo.Client{
		FloatingIPActions: &doit.FloatingIPActionsServiceMock{
			GetFn: func(ip string, actionID int) (*godo.Action, *godo.Response, error) {
				assert.Equal(t, "127.0.0.1", ip)
				assert.Equal(t, 2, actionID)
				return &testAction, nil, nil
			},
		},
	}

	withTestClient(client, func(c *TestConfig) {
		ns := "test"
		RunFloatingIPActionsGet(ns, c, ioutil.Discard, []string{"127.0.0.1", "2"})
	})

}

func TestFloatingIPActionsAssign(t *testing.T) {
	client := &godo.Client{
		FloatingIPActions: &doit.FloatingIPActionsServiceMock{
			AssignFn: func(ip string, dropletID int) (*godo.Action, *godo.Response, error) {

				assert.Equal(t, ip, "127.0.0.1")
				assert.Equal(t, dropletID, 2)

				return &testAction, nil, nil
			},
		},
	}

	withTestClient(client, func(c *TestConfig) {
		ns := "test"
		RunFloatingIPActionsAssign(ns, c, ioutil.Discard, []string{"127.0.0.1", "2"})
	})
}

func TestFloatingIPActionsUnassign(t *testing.T) {
	client := &godo.Client{
		FloatingIPActions: &doit.FloatingIPActionsServiceMock{
			UnassignFn: func(ip string) (*godo.Action, *godo.Response, error) {

				assert.Equal(t, ip, "127.0.0.1")

				return &testAction, nil, nil
			},
		},
	}

	withTestClient(client, func(c *TestConfig) {
		ns := "test"

		RunFloatingIPActionsUnassign(ns, c, ioutil.Discard, []string{"127.0.0.1"})
	})
}

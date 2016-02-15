package commands

import (
	"testing"

	"github.com/bryanl/doit"
	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"
)

func TestFloatingIPCommands(t *testing.T) {
	cmd := FloatingIP()
	assert.NotNil(t, cmd)
	assertCommandNames(t, cmd, "create", "delete", "get", "list")
}

func TestFloatingIPsList(t *testing.T) {
	didRun := false

	client := &godo.Client{
		FloatingIPs: &doit.FloatingIPsServiceMock{
			ListFn: func(opts *godo.ListOptions) ([]godo.FloatingIP, *godo.Response, error) {
				didRun = true

				resp := &godo.Response{
					Links: &godo.Links{
						Pages: &godo.Pages{},
					},
				}
				return testFloatingIPList, resp, nil
			},
		},
	}

	withTestClient(client, func(config *cmdConfig) {
		RunFloatingIPList(config)
		if !didRun {
			t.Errorf("List() did not run")
		}
	})
}

func TestFloatingIPsGet(t *testing.T) {
	client := &godo.Client{
		FloatingIPs: &doit.FloatingIPsServiceMock{
			GetFn: func(ip string) (*godo.FloatingIP, *godo.Response, error) {
				assert.Equal(t, "127.0.0.1", ip)
				return &testFloatingIP, nil, nil
			},
		},
	}

	withTestClient(client, func(config *cmdConfig) {
		config.args = append(config.args, "127.0.0.1")

		RunFloatingIPGet(config)
	})
}

func TestFloatingIPsCreate(t *testing.T) {
	client := &godo.Client{
		FloatingIPs: &doit.FloatingIPsServiceMock{
			CreateFn: func(req *godo.FloatingIPCreateRequest) (*godo.FloatingIP, *godo.Response, error) {
				assert.Equal(t, "dev0", req.Region)
				assert.Equal(t, 1, req.DropletID)
				return &testFloatingIP, nil, nil
			},
		},
	}

	withTestClient(client, func(config *cmdConfig) {
		config.doitConfig.Set(config.ns, doit.ArgRegionSlug, "dev0")
		config.doitConfig.Set(config.ns, doit.ArgDropletID, 1)

		RunFloatingIPCreate(config)
	})
}

func TestFloatingIPsDelete(t *testing.T) {
	client := &godo.Client{
		FloatingIPs: &doit.FloatingIPsServiceMock{
			DeleteFn: func(ip string) (*godo.Response, error) {
				assert.Equal(t, "127.0.0.1", ip)
				return nil, nil
			},
		},
	}

	withTestClient(client, func(config *cmdConfig) {
		config.args = append(config.args, "127.0.0.1")

		RunFloatingIPDelete(config)
	})
}

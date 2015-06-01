package domains

import (
	"testing"

	"github.com/bryanl/docli/docli"
	"github.com/codegangsta/cli"
	"github.com/digitalocean/godo"
)

var (
	testDomain     = godo.Domain{Name: "example.com"}
	testDomainList = []godo.Domain{
		testDomain,
	}
)

func TestDomainsList(t *testing.T) {
	domainsDisList := false

	client := &godo.Client{
		Domains: &docli.DomainsServiceMock{
			ListFn: func(opts *godo.ListOptions) ([]godo.Domain, *godo.Response, error) {
				domainsDisList = true
				resp := &godo.Response{
					Links: &godo.Links{},
				}
				return testDomainList, resp, nil
			},
		},
	}

	cs := &docli.TestClientSource{client}

	docli.WithinTest(cs, nil, func(c *cli.Context) {
		List(c)
		if !domainsDisList {
			t.Errorf("List() did not run")
		}
	})
}

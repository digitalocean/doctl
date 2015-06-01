package domains

import (
	"flag"
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

func TestDomainsGet(t *testing.T) {
	client := &godo.Client{
		Domains: &docli.DomainsServiceMock{
			GetFn: func(name string) (*godo.Domain, *godo.Response, error) {
				if got, expected := name, testDomain.Name; got != expected {
					t.Errorf("GetFn() called with %q; expected %q", got, expected)
				}
				return &testDomain, nil, nil
			},
		},
	}

	cs := &docli.TestClientSource{client}
	fs := flag.NewFlagSet("flag set", 0)
	fs.String("domain-name", testDomain.Name, "domain-id")

	docli.WithinTest(cs, fs, func(c *cli.Context) {
		Get(c)
	})
}

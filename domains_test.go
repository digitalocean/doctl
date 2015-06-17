package docli

import (
	"flag"
	"reflect"
	"testing"

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
		Domains: &DomainsServiceMock{
			ListFn: func(opts *godo.ListOptions) ([]godo.Domain, *godo.Response, error) {
				domainsDisList = true
				resp := &godo.Response{
					Links: &godo.Links{},
				}
				return testDomainList, resp, nil
			},
		},
	}

	cs := NewTestConfig(client)

	WithinTest(cs, nil, func(c *cli.Context) {
		DomainList(c)
		if !domainsDisList {
			t.Errorf("List() did not run")
		}
	})
}

func TestDomainsGet(t *testing.T) {
	client := &godo.Client{
		Domains: &DomainsServiceMock{
			GetFn: func(name string) (*godo.Domain, *godo.Response, error) {
				if got, expected := name, testDomain.Name; got != expected {
					t.Errorf("GetFn() called with %q; expected %q", got, expected)
				}
				return &testDomain, nil, nil
			},
		},
	}

	cs := NewTestConfig(client)
	fs := flag.NewFlagSet("flag set", 0)
	fs.String("domain-name", testDomain.Name, "domain-id")

	WithinTest(cs, fs, func(c *cli.Context) {
		DomainGet(c)
	})
}

func TestDomainsCreate(t *testing.T) {
	client := &godo.Client{
		Domains: &DomainsServiceMock{
			CreateFn: func(req *godo.DomainCreateRequest) (*godo.Domain, *godo.Response, error) {
				expected := &godo.DomainCreateRequest{
					Name:      testDomain.Name,
					IPAddress: "127.0.0.1",
				}
				if got := req; !reflect.DeepEqual(got, expected) {
					t.Errorf("CreateFn() called with %#v; expected %#v", got, expected)
				}
				return &testDomain, nil, nil
			},
		},
	}

	cs := NewTestConfig(client)
	fs := flag.NewFlagSet("flag set", 0)
	fs.String("domain-name", testDomain.Name, "domain-name")
	fs.String("ip-address", "127.0.0.1", "ip- address")

	WithinTest(cs, fs, func(c *cli.Context) {
		DomainCreate(c)
	})
}

func TestDomainsDelete(t *testing.T) {
	client := &godo.Client{
		Domains: &DomainsServiceMock{
			DeleteFn: func(name string) (*godo.Response, error) {
				if got, expected := name, testDomain.Name; got != expected {
					t.Errorf("DeleteFn() received %q; expected %q", got, expected)
				}
				return nil, nil
			},
		},
	}

	cs := NewTestConfig(client)
	fs := flag.NewFlagSet("flag set", 0)
	fs.String("domain-name", testDomain.Name, "domain-name")

	WithinTest(cs, fs, func(c *cli.Context) {
		DomainDelete(c)
	})
}

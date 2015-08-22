package commands

import (
	"io/ioutil"
	"reflect"
	"testing"

	"github.com/bryanl/doit"
	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"
)

var (
	testDomain     = godo.Domain{Name: "example.com"}
	testDomainList = []godo.Domain{
		testDomain,
	}
	testRecord     = godo.DomainRecord{ID: 1}
	testRecordList = []godo.DomainRecord{testRecord}
)

func TestDomainsCreate(t *testing.T) {
	client := &godo.Client{
		Domains: &doit.DomainsServiceMock{
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

	withTestClient(client, func(c doit.ViperConfig) {
		c.Set(doit.ArgDomainName, testDomain.Name)
		c.Set(doit.ArgIPAddress, "127.0.0.1")
		err := RunDomainCreate(ioutil.Discard)
		assert.NoError(t, err)
	})
}

func TestDomainsList(t *testing.T) {
	domainsDisList := false

	client := &godo.Client{
		Domains: &doit.DomainsServiceMock{
			ListFn: func(opts *godo.ListOptions) ([]godo.Domain, *godo.Response, error) {
				domainsDisList = true
				resp := &godo.Response{
					Links: &godo.Links{},
				}
				return testDomainList, resp, nil
			},
		},
	}

	withTestClient(client, func(c doit.ViperConfig) {
		err := RunDomainList(ioutil.Discard)
		assert.NoError(t, err)
		if !domainsDisList {
			t.Errorf("List() did not run")
		}
	})
}

func TestDomainsGet(t *testing.T) {
	client := &godo.Client{
		Domains: &doit.DomainsServiceMock{
			GetFn: func(name string) (*godo.Domain, *godo.Response, error) {
				if got, expected := name, testDomain.Name; got != expected {
					t.Errorf("GetFn() called with %q; expected %q", got, expected)
				}
				return &testDomain, nil, nil
			},
		},
	}

	withTestClient(client, func(c doit.ViperConfig) {
		c.Set(doit.ArgDomainName, testDomain.Name)
		err := RunDomainGet(ioutil.Discard)
		assert.NoError(t, err)
	})
}

func TestDomainsGet_DomainRequred(t *testing.T) {
	client := &godo.Client{}

	withTestClient(client, func(c doit.ViperConfig) {
		err := RunDomainGet(ioutil.Discard)
		assert.Error(t, err)
	})
}

func TestDomainsDelete(t *testing.T) {
	client := &godo.Client{
		Domains: &doit.DomainsServiceMock{
			DeleteFn: func(name string) (*godo.Response, error) {
				if got, expected := name, testDomain.Name; got != expected {
					t.Errorf("DeleteFn() received %q; expected %q", got, expected)
				}
				return nil, nil
			},
		},
	}

	withTestClient(client, func(c doit.ViperConfig) {
		c.Set(doit.ArgDomainName, testDomain.Name)
		err := RunDomainDelete(ioutil.Discard)
		assert.NoError(t, err)
	})
}

func TestDomainsGet_RequiredArguments(t *testing.T) {
	client := &godo.Client{}

	withTestClient(client, func(c doit.ViperConfig) {
		err := RunDomainDelete(ioutil.Discard)
		assert.Error(t, err)
	})
}

func TestRecordsList(t *testing.T) {
	recordsDidList := false

	client := &godo.Client{
		Domains: &doit.DomainsServiceMock{
			RecordsFn: func(name string, opts *godo.ListOptions) ([]godo.DomainRecord, *godo.Response, error) {
				recordsDidList = true
				return testRecordList, nil, nil
			},
		},
	}

	withTestClient(client, func(c doit.ViperConfig) {
		c.Set(doit.ArgDomainName, "example.com")

		err := RunRecordList(ioutil.Discard)
		assert.NoError(t, err)
		assert.True(t, recordsDidList)
	})
}

func TestRecordList_RequiredArguments(t *testing.T) {
	client := &godo.Client{}

	withTestClient(client, func(c doit.ViperConfig) {
		err := RunRecordList(ioutil.Discard)
		assert.Error(t, err)
	})
}

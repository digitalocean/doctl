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

	withTestClient(client, func(c *TestConfig) {
		ns := "test"
		c.Set(ns, doit.ArgDomainName, testDomain.Name)
		c.Set(ns, doit.ArgDomainName, testDomain.Name)
		c.Set(ns, doit.ArgIPAddress, "127.0.0.1")
		err := RunDomainCreate(ns, ioutil.Discard)
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

	withTestClient(client, func(c *TestConfig) {
		ns := "test"
		err := RunDomainList(ns, ioutil.Discard)
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

	withTestClient(client, func(c *TestConfig) {
		ns := "test"
		c.Set(ns, doit.ArgDomainName, testDomain.Name)
		err := RunDomainGet(ns, ioutil.Discard)
		assert.NoError(t, err)
	})
}

func TestDomainsGet_DomainRequred(t *testing.T) {
	client := &godo.Client{}

	withTestClient(client, func(c *TestConfig) {
		ns := "test"
		err := RunDomainGet(ns, ioutil.Discard)
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

	withTestClient(client, func(c *TestConfig) {
		ns := "test"
		c.Set(ns, doit.ArgDomainName, testDomain.Name)
		err := RunDomainDelete(ns, ioutil.Discard)
		assert.NoError(t, err)
	})
}

func TestDomainsGet_RequiredArguments(t *testing.T) {
	client := &godo.Client{}

	withTestClient(client, func(c *TestConfig) {
		ns := "test"
		err := RunDomainDelete(ns, ioutil.Discard)
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

	withTestClient(client, func(c *TestConfig) {
		ns := "test"
		c.Set(ns, doit.ArgDomainName, "example.com")

		err := RunRecordList(ns, ioutil.Discard)
		assert.NoError(t, err)
		assert.True(t, recordsDidList)
	})
}

func TestRecordList_RequiredArguments(t *testing.T) {
	client := &godo.Client{}

	withTestClient(client, func(c *TestConfig) {
		ns := "test"
		err := RunRecordList(ns, ioutil.Discard)
		assert.Error(t, err)
	})
}

func TestRecordsCreate(t *testing.T) {
	client := &godo.Client{
		Domains: &doit.DomainsServiceMock{
			CreateRecordFn: func(name string, req *godo.DomainRecordEditRequest) (*godo.DomainRecord, *godo.Response, error) {
				expected := &godo.DomainRecordEditRequest{
					Type: "A",
					Name: "foo.example.com.",
					Data: "192.168.1.1",
				}

				assert.Equal(t, "example.com", name)
				assert.Equal(t, expected, req)

				return &testRecord, nil, nil
			},
		},
	}

	withTestClient(client, func(c *TestConfig) {
		ns := "test"
		c.Set(ns, doit.ArgDomainName, "example.com")
		c.Set(ns, doit.ArgRecordType, "A")
		c.Set(ns, doit.ArgRecordName, "foo.example.com.")
		c.Set(ns, doit.ArgRecordData, "192.168.1.1")

		err := RunRecordCreate(ns, ioutil.Discard)
		assert.NoError(t, err)
	})
}

func TestRecordCreate_RequiredArguments(t *testing.T) {
	client := &godo.Client{}

	withTestClient(client, func(c *TestConfig) {
		ns := "test"
		err := RunRecordCreate(ns, ioutil.Discard)
		assert.Error(t, err)
	})
}

func TestRecordsDelete(t *testing.T) {
	client := &godo.Client{
		Domains: &doit.DomainsServiceMock{
			DeleteRecordFn: func(name string, id int) (*godo.Response, error) {
				if got, expected := name, "example.com"; got != expected {
					t.Errorf("CreateFn domain name = %q; expected %q", got, expected)
				}
				if got, expected := id, 1; got != expected {
					t.Errorf("CreateFn id = %d; expected %d", got, expected)
				}
				return nil, nil
			},
		},
	}

	withTestClient(client, func(c *TestConfig) {
		ns := "test"
		c.Set(ns, doit.ArgDomainName, "example.com")
		c.Set(ns, doit.ArgRecordID, 1)
		err := RunRecordDelete(ns, ioutil.Discard)
		assert.NoError(t, err)
	})
}

func TestRecordsUpdate(t *testing.T) {
	client := &godo.Client{
		Domains: &doit.DomainsServiceMock{
			EditRecordFn: func(name string, id int, req *godo.DomainRecordEditRequest) (*godo.DomainRecord, *godo.Response, error) {
				expected := &godo.DomainRecordEditRequest{
					Type: "A",
					Name: "foo.example.com.",
					Data: "192.168.1.1",
				}

				assert.Equal(t, "example.com", name)
				assert.Equal(t, 1, id)
				assert.Equal(t, expected, req)

				return &testRecord, nil, nil
			},
		},
	}

	withTestClient(client, func(c *TestConfig) {
		ns := "test"
		c.Set(ns, doit.ArgDomainName, "example.com")
		c.Set(ns, doit.ArgRecordID, 1)
		c.Set(ns, doit.ArgRecordType, "A")
		c.Set(ns, doit.ArgRecordName, "foo.example.com.")
		c.Set(ns, doit.ArgRecordData, "192.168.1.1")

		err := RunRecordUpdate(ns, ioutil.Discard)
		assert.NoError(t, err)
	})
}

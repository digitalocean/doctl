package commands

import (
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

func TestDomainsCommand(t *testing.T) {
	cmd := Domain()
	assert.NotNil(t, cmd)
	assertCommandNames(t, cmd, "create", "list", "get", "delete", "records")
}

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

	withTestClient(client, func(config *cmdConfig) {
		config.args = append(config.args, testDomain.Name)
		config.doitConfig.Set(config.ns, doit.ArgIPAddress, "127.0.0.1")
		err := RunDomainCreate(config)
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

	withTestClient(client, func(config *cmdConfig) {
		err := RunDomainList(config)
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

	withTestClient(client, func(config *cmdConfig) {
		config.args = append(config.args, testDomain.Name)
		err := RunDomainGet(config)
		assert.NoError(t, err)
	})
}

func TestDomainsGet_DomainRequred(t *testing.T) {
	client := &godo.Client{}

	withTestClient(client, func(config *cmdConfig) {
		err := RunDomainGet(config)
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

	withTestClient(client, func(config *cmdConfig) {
		config.args = append(config.args, testDomain.Name)

		err := RunDomainDelete(config)
		assert.NoError(t, err)
	})
}

func TestDomainsGet_RequiredArguments(t *testing.T) {
	client := &godo.Client{}

	withTestClient(client, func(config *cmdConfig) {
		err := RunDomainDelete(config)
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

	withTestClient(client, func(config *cmdConfig) {
		config.args = append(config.args, "example.com")

		err := RunRecordList(config)
		assert.NoError(t, err)
		assert.True(t, recordsDidList)
	})
}

func TestRecordList_RequiredArguments(t *testing.T) {
	client := &godo.Client{}

	withTestClient(client, func(config *cmdConfig) {
		err := RunRecordList(config)
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

	withTestClient(client, func(config *cmdConfig) {
		config.doitConfig.Set(config.ns, doit.ArgRecordType, "A")
		config.doitConfig.Set(config.ns, doit.ArgRecordName, "foo.example.com.")
		config.doitConfig.Set(config.ns, doit.ArgRecordData, "192.168.1.1")

		config.args = append(config.args, "example.com")

		err := RunRecordCreate(config)
		assert.NoError(t, err)
	})
}

func TestRecordCreate_RequiredArguments(t *testing.T) {
	client := &godo.Client{}

	withTestClient(client, func(config *cmdConfig) {
		err := RunRecordCreate(config)
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

	withTestClient(client, func(config *cmdConfig) {
		config.args = append(config.args, "example.com", "1")

		err := RunRecordDelete(config)
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

	withTestClient(client, func(config *cmdConfig) {
		config.doitConfig.Set(config.ns, doit.ArgRecordID, 1)
		config.doitConfig.Set(config.ns, doit.ArgRecordType, "A")
		config.doitConfig.Set(config.ns, doit.ArgRecordName, "foo.example.com.")
		config.doitConfig.Set(config.ns, doit.ArgRecordData, "192.168.1.1")

		config.args = append(config.args, "example.com")

		err := RunRecordUpdate(config)
		assert.NoError(t, err)
	})
}

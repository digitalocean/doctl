package domainrecs

import (
	"flag"
	"reflect"
	"testing"

	"github.com/bryanl/docli/docli"
	"github.com/codegangsta/cli"
	"github.com/digitalocean/godo"
)

var (
	testRecord     = godo.DomainRecord{ID: 1}
	testRecordList = []godo.DomainRecord{testRecord}
)

func TestRecordsList(t *testing.T) {
	recordsDidList := false

	client := &godo.Client{
		Domains: &docli.DomainsServiceMock{
			RecordsFn: func(name string, opts *godo.ListOptions) ([]godo.DomainRecord, *godo.Response, error) {
				recordsDidList = true
				return testRecordList, nil, nil
			},
		},
	}

	cs := &docli.TestClientSource{client}
	fs := flag.NewFlagSet("flag set", 0)
	fs.String("domain-name", "example.com", "domain-name")

	docli.WithinTest(cs, fs, func(c *cli.Context) {
		List(c)
		if !recordsDidList {
			t.Errorf("List() did not run")
		}
	})
}

func TestRecordsGet(t *testing.T) {
	client := &godo.Client{
		Domains: &docli.DomainsServiceMock{
			RecordFn: func(name string, id int) (*godo.DomainRecord, *godo.Response, error) {
				if got, expected := name, "example.com"; got != expected {
					t.Errorf("RecordFn domain = %q; expected %q", got, expected)
				}
				if got, expected := id, testRecord.ID; got != expected {
					t.Errorf("RecordFn id = %d; expected %d", got, expected)
				}
				return &testRecord, nil, nil
			},
		},
	}

	cs := &docli.TestClientSource{client}
	fs := flag.NewFlagSet("flag set", 0)
	fs.String("domain-name", "example.com", "domain-name")
	fs.Int("record-id", testRecord.ID, "record-id")

	docli.WithinTest(cs, fs, func(c *cli.Context) {
		Get(c)
	})
}

func TestRecordsCreate(t *testing.T) {
	client := &godo.Client{
		Domains: &docli.DomainsServiceMock{
			CreateRecordFn: func(name string, req *godo.DomainRecordEditRequest) (*godo.DomainRecord, *godo.Response, error) {
				expected := &godo.DomainRecordEditRequest{
					Type: "A",
					Name: "foo.example.com.",
					Data: "192.168.1.1",
				}

				if got, expected := name, "example.com"; got != expected {
					t.Errorf("CreateFn domain name = %q; expected %q", got, expected)
				}
				if got := req; !reflect.DeepEqual(got, expected) {
					t.Errorf("CreateFn request = %#v; expected %#v", got, expected)
				}
				return &testRecord, nil, nil
			},
		},
	}

	cs := &docli.TestClientSource{client}
	fs := flag.NewFlagSet("flag set", 0)
	fs.String("domain-name", "example.com", "domain-name")
	fs.String("record-type", "A", "record-type")
	fs.String("record-name", "foo.example.com.", "record-name")
	fs.String("record-data", "192.168.1.1", "record-name")

	docli.WithinTest(cs, fs, func(c *cli.Context) {
		Create(c)
	})
}

func TestRecordsUpdate(t *testing.T) {
	client := &godo.Client{
		Domains: &docli.DomainsServiceMock{
			EditRecordFn: func(name string, id int, req *godo.DomainRecordEditRequest) (*godo.DomainRecord, *godo.Response, error) {
				expected := &godo.DomainRecordEditRequest{
					Type: "A",
					Name: "foo.example.com.",
					Data: "192.168.1.1",
				}

				if got, expected := name, "example.com"; got != expected {
					t.Errorf("CreateFn domain name = %q; expected %q", got, expected)
				}
				if got, expected := id, 1; got != expected {
					t.Errorf("CreateFn id = %d; expected %d", got, expected)
				}
				if got := req; !reflect.DeepEqual(got, expected) {
					t.Errorf("CreateFn request = %#v; expected %#v", got, expected)
				}
				return &testRecord, nil, nil
			},
		},
	}

	cs := &docli.TestClientSource{client}
	fs := flag.NewFlagSet("flag set", 0)
	fs.String("domain-name", "example.com", "domain-name")
	fs.Int("record-id", 1, "record-id")
	fs.String("record-type", "A", "record-type")
	fs.String("record-name", "foo.example.com.", "record-name")
	fs.String("record-data", "192.168.1.1", "record-name")

	docli.WithinTest(cs, fs, func(c *cli.Context) {
		Update(c)
	})
}

func TestRecordsDelete(t *testing.T) {
	client := &godo.Client{
		Domains: &docli.DomainsServiceMock{
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

	cs := &docli.TestClientSource{client}
	fs := flag.NewFlagSet("flag set", 0)
	fs.String("domain-name", "example.com", "domain-name")
	fs.Int("record-id", 1, "record-id")

	docli.WithinTest(cs, fs, func(c *cli.Context) {
		Delete(c)
	})
}

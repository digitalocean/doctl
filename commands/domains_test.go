package commands

import (
	"testing"

	"github.com/bryanl/doit"
	"github.com/bryanl/doit/do"
	"github.com/bryanl/doit/do/mocks"
	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"
)

var (
	testDomain     = do.Domain{Domain: &godo.Domain{Name: "example.com"}}
	testDomainList = do.Domains{
		testDomain,
	}
	testRecord     = do.DomainRecord{DomainRecord: &godo.DomainRecord{ID: 1}}
	testRecordList = do.DomainRecords{testRecord}
)

func TestDomainsCommand(t *testing.T) {
	cmd := Domain()
	assert.NotNil(t, cmd)
	assertCommandNames(t, cmd, "create", "list", "get", "delete", "records")
}

func TestDomainsCreate(t *testing.T) {
	withTestClient(func(config *cmdConfig) {
		dos := &mocks.DomainsService{}
		config.dos = dos

		dcr := &godo.DomainCreateRequest{Name: "example.com", IPAddress: "127.0.0.1"}
		dos.On("Create", dcr).Return(&testDomain, nil)

		config.args = append(config.args, testDomain.Name)
		config.doitConfig.Set(config.ns, doit.ArgIPAddress, "127.0.0.1")
		err := RunDomainCreate(config)
		assert.NoError(t, err)
	})
}

func TestDomainsList(t *testing.T) {
	withTestClient(func(config *cmdConfig) {
		dos := &mocks.DomainsService{}
		config.dos = dos

		dos.On("List").Return(testDomainList, nil)

		err := RunDomainList(config)
		assert.NoError(t, err)
	})
}

func TestDomainsGet(t *testing.T) {
	withTestClient(func(config *cmdConfig) {
		dos := &mocks.DomainsService{}
		config.dos = dos

		dos.On("Get", "example.com").Return(&testDomain, nil)

		config.args = append(config.args, testDomain.Name)
		err := RunDomainGet(config)
		assert.NoError(t, err)
	})
}

func TestDomainsGet_DomainRequired(t *testing.T) {
	withTestClient(func(config *cmdConfig) {
		err := RunDomainGet(config)
		assert.Error(t, err)
	})
}

func TestDomainsDelete(t *testing.T) {
	withTestClient(func(config *cmdConfig) {
		dos := &mocks.DomainsService{}
		config.dos = dos

		dos.On("Delete", "example.com").Return(nil)

		config.args = append(config.args, testDomain.Name)

		err := RunDomainDelete(config)
		assert.NoError(t, err)
	})
}

func TestDomainsGet_RequiredArguments(t *testing.T) {
	withTestClient(func(config *cmdConfig) {
		err := RunDomainDelete(config)
		assert.Error(t, err)
	})
}

func TestRecordsList(t *testing.T) {
	withTestClient(func(config *cmdConfig) {
		dos := &mocks.DomainsService{}
		config.dos = dos

		dos.On("Records", "example.com").Return(testRecordList, nil)

		config.args = append(config.args, "example.com")

		err := RunRecordList(config)
		assert.NoError(t, err)
	})
}

func TestRecordList_RequiredArguments(t *testing.T) {
	withTestClient(func(config *cmdConfig) {
		err := RunRecordList(config)
		assert.Error(t, err)
	})
}

func TestRecordsCreate(t *testing.T) {
	withTestClient(func(config *cmdConfig) {
		dos := &mocks.DomainsService{}
		config.dos = dos

		dcer := &godo.DomainRecordEditRequest{Type: "A", Name: "foo.example.com.", Data: "192.168.1.1", Priority: 0, Port: 0, Weight: 0}
		dos.On("CreateRecord", "example.com", dcer).Return(&testRecord, nil)

		config.doitConfig.Set(config.ns, doit.ArgRecordType, "A")
		config.doitConfig.Set(config.ns, doit.ArgRecordName, "foo.example.com.")
		config.doitConfig.Set(config.ns, doit.ArgRecordData, "192.168.1.1")

		config.args = append(config.args, "example.com")

		err := RunRecordCreate(config)
		assert.NoError(t, err)
	})
}

func TestRecordCreate_RequiredArguments(t *testing.T) {
	withTestClient(func(config *cmdConfig) {
		err := RunRecordCreate(config)
		assert.Error(t, err)
	})
}

func TestRecordsDelete(t *testing.T) {
	withTestClient(func(config *cmdConfig) {
		dos := &mocks.DomainsService{}
		config.dos = dos

		dos.On("DeleteRecord", "example.com", 1).Return(nil)

		config.args = append(config.args, "example.com", "1")

		err := RunRecordDelete(config)
		assert.NoError(t, err)
	})
}

func TestRecordsUpdate(t *testing.T) {
	withTestClient(func(config *cmdConfig) {
		dos := &mocks.DomainsService{}
		config.dos = dos

		dcer := &godo.DomainRecordEditRequest{Type: "A", Name: "foo.example.com.", Data: "192.168.1.1", Priority: 0, Port: 0, Weight: 0}
		dos.On("EditRecord", "example.com", 1, dcer).Return(&testRecord, nil)

		config.doitConfig.Set(config.ns, doit.ArgRecordID, 1)
		config.doitConfig.Set(config.ns, doit.ArgRecordType, "A")
		config.doitConfig.Set(config.ns, doit.ArgRecordName, "foo.example.com.")
		config.doitConfig.Set(config.ns, doit.ArgRecordData, "192.168.1.1")

		config.args = append(config.args, "example.com")

		err := RunRecordUpdate(config)
		assert.NoError(t, err)
	})
}

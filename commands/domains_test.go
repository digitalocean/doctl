/*
Copyright 2018 The Doctl Authors All rights reserved.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package commands

import (
	"testing"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/do"
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
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		dcr := &godo.DomainCreateRequest{Name: "example.com", IPAddress: "127.0.0.1"}
		tm.domains.EXPECT().Create(dcr).Return(&testDomain, nil)

		config.Args = append(config.Args, testDomain.Name)
		config.Doit.Set(config.NS, doctl.ArgIPAddress, "127.0.0.1")
		err := RunDomainCreate(config)
		assert.NoError(t, err)
	})
}

func TestDomainsList(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.domains.EXPECT().List().Return(testDomainList, nil)

		err := RunDomainList(config)
		assert.NoError(t, err)
	})
}

func TestDomainsGet(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.domains.EXPECT().Get("example.com").Return(&testDomain, nil)

		config.Args = append(config.Args, testDomain.Name)
		err := RunDomainGet(config)
		assert.NoError(t, err)
	})
}

func TestDomainsGet_DomainRequired(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		err := RunDomainGet(config)
		assert.Error(t, err)
	})
}

func TestDomainsDelete(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.domains.EXPECT().Delete("example.com").Return(nil)

		config.Args = append(config.Args, testDomain.Name)

		config.Doit.Set(config.NS, doctl.ArgForce, true)

		err := RunDomainDelete(config)
		assert.NoError(t, err)
	})
}

func TestDomainsGet_RequiredArguments(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		err := RunDomainDelete(config)
		assert.Error(t, err)
	})
}

func TestRecordsList(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.domains.EXPECT().Records("example.com").Return(testRecordList, nil)

		config.Args = append(config.Args, "example.com")

		err := RunRecordList(config)
		assert.NoError(t, err)
	})
}

func TestRecordList_RequiredArguments(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		err := RunRecordList(config)
		assert.Error(t, err)
	})
}

func TestRecordsCreate(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		port := 0
		dcer := &do.DomainRecordEditRequest{Type: "A", Name: "foo.example.com.", Data: "192.168.1.1", Priority: 0, Port: &port, TTL: 0, Weight: 0}
		tm.domains.EXPECT().CreateRecord("example.com", dcer).Return(&testRecord, nil)

		config.Doit.Set(config.NS, doctl.ArgRecordType, "A")
		config.Doit.Set(config.NS, doctl.ArgRecordName, "foo.example.com.")
		config.Doit.Set(config.NS, doctl.ArgRecordData, "192.168.1.1")
		config.Doit.Set(config.NS, doctl.ArgRecordPort, "0")

		config.Args = append(config.Args, "example.com")

		err := RunRecordCreate(config)
		assert.NoError(t, err)
	})
}

func TestRecordCreate_RequiredArguments(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		err := RunRecordCreate(config)
		assert.Error(t, err)
	})
}

func TestRecordsDelete(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.domains.EXPECT().DeleteRecord("example.com", 1).Return(nil)

		config.Args = append(config.Args, "example.com", "1")

		config.Doit.Set(config.NS, doctl.ArgForce, true)

		err := RunRecordDelete(config)
		assert.NoError(t, err)
	})
}

func TestRecordsUpdate(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		port := 0
		dcer := &do.DomainRecordEditRequest{Type: "A", Name: "foo.example.com.", Data: "192.168.1.1", Priority: 0, Port: &port, TTL: 0, Weight: 0}
		tm.domains.EXPECT().EditRecord("example.com", 1, dcer).Return(&testRecord, nil)

		config.Doit.Set(config.NS, doctl.ArgRecordID, 1)
		config.Doit.Set(config.NS, doctl.ArgRecordType, "A")
		config.Doit.Set(config.NS, doctl.ArgRecordName, "foo.example.com.")
		config.Doit.Set(config.NS, doctl.ArgRecordData, "192.168.1.1")
		config.Doit.Set(config.NS, doctl.ArgRecordPort, "0")

		config.Args = append(config.Args, "example.com")

		err := RunRecordUpdate(config)
		assert.NoError(t, err)
	})
}

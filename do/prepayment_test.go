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

package do_test

import (
	"context"
	"reflect"
	"testing"

	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

type MockGodoPrepaymentService struct {
	ctrl     *gomock.Controller
	recorder *MockGodoPrepaymentServiceMockRecorder
}

type MockGodoPrepaymentServiceMockRecorder struct {
	mock *MockGodoPrepaymentService
}

func NewMockGodoPrepaymentService(ctrl *gomock.Controller) *MockGodoPrepaymentService {
	mock := &MockGodoPrepaymentService{ctrl: ctrl}
	mock.recorder = &MockGodoPrepaymentServiceMockRecorder{mock}
	return mock
}

func (m *MockGodoPrepaymentService) EXPECT() *MockGodoPrepaymentServiceMockRecorder {
	return m.recorder
}

func (m *MockGodoPrepaymentService) GetConfig(arg0 context.Context) (*godo.PrepaymentConfigResponse, *godo.Response, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetConfig", arg0)
	ret0, _ := ret[0].(*godo.PrepaymentConfigResponse)
	ret1, _ := ret[1].(*godo.Response)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

func (mr *MockGodoPrepaymentServiceMockRecorder) GetConfig(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetConfig", reflect.TypeOf((*MockGodoPrepaymentService)(nil).GetConfig), arg0)
}

func (m *MockGodoPrepaymentService) GetStatus(arg0 context.Context) (*godo.PrepaymentStatusResponse, *godo.Response, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetStatus", arg0)
	ret0, _ := ret[0].(*godo.PrepaymentStatusResponse)
	ret1, _ := ret[1].(*godo.Response)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

func (mr *MockGodoPrepaymentServiceMockRecorder) GetStatus(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetStatus", reflect.TypeOf((*MockGodoPrepaymentService)(nil).GetStatus), arg0)
}

func TestPrepaymentServiceGetConfig(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	gPrepaymentSvc := NewMockGodoPrepaymentService(ctrl)
	gResp := &godo.PrepaymentConfigResponse{
		Config: &godo.PrepaymentConfig{SpendLimit: "100.00"},
		Status: &godo.PrepaymentStatus{Balance: "25.00"},
	}
	gPrepaymentSvc.EXPECT().GetConfig(context.TODO()).Return(gResp, nil, nil)

	client := &godo.Client{
		Prepayment: gPrepaymentSvc,
	}
	ps := do.NewPrepaymentService(client)

	resp, err := ps.GetConfig()
	assert.NoError(t, err)
	assert.Equal(t, "100.00", resp.Config.SpendLimit)
	assert.Equal(t, "25.00", resp.Status.Balance)
}

func TestPrepaymentServiceGetStatus(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	gPrepaymentSvc := NewMockGodoPrepaymentService(ctrl)
	gResp := &godo.PrepaymentStatusResponse{
		Status: &godo.PrepaymentStatus{MonthToDateBalance: "75.00"},
	}
	gPrepaymentSvc.EXPECT().GetStatus(context.TODO()).Return(gResp, nil, nil)

	client := &godo.Client{
		Prepayment: gPrepaymentSvc,
	}
	ps := do.NewPrepaymentService(client)

	resp, err := ps.GetStatus()
	assert.NoError(t, err)
	assert.Equal(t, "75.00", resp.Status.MonthToDateBalance)
}

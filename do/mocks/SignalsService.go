package mocks

import (
	do "github.com/digitalocean/doctl/do"
	gomock "go.uber.org/mock/gomock"
	"reflect"
)

// MockSignalsService is a mock of SignalsService interface.
type MockSignalsService struct {
	ctrl     *gomock.Controller
	recorder *MockSignalsServiceMockRecorder
}

// MockSignalsServiceMockRecorder is the mock recorder for MockSignalsService.
type MockSignalsServiceMockRecorder struct {
	mock *MockSignalsService
}

// NewMockSignalsService creates a new mock instance.
func NewMockSignalsService(ctrl *gomock.Controller) *MockSignalsService {
	mock := &MockSignalsService{ctrl: ctrl}
	mock.recorder = &MockSignalsServiceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockSignalsService) EXPECT() *MockSignalsServiceMockRecorder {
	return m.recorder
}

// ListConsents mocks base method.
func (m *MockSignalsService) ListConsents() (do.SignalsConsents, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListConsents")
	ret0, _ := ret[0].(do.SignalsConsents)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListConsents indicates an expected call of ListConsents.
func (mr *MockSignalsServiceMockRecorder) ListConsents() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListConsents", reflect.TypeOf((*MockSignalsService)(nil).ListConsents))
}

// GetConsent mocks base method.
func (m *MockSignalsService) GetConsent(agentID string) (*do.SignalsConsent, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetConsent", agentID)
	ret0, _ := ret[0].(*do.SignalsConsent)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetConsent indicates an expected call of GetConsent.
func (mr *MockSignalsServiceMockRecorder) GetConsent(agentID any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetConsent", reflect.TypeOf((*MockSignalsService)(nil).GetConsent), agentID)
}

// SetConsent mocks base method.
func (m *MockSignalsService) SetConsent(agentID string, enabled bool) (*do.SignalsConsent, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SetConsent", agentID, enabled)
	ret0, _ := ret[0].(*do.SignalsConsent)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// SetConsent indicates an expected call of SetConsent.
func (mr *MockSignalsServiceMockRecorder) SetConsent(agentID, enabled any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetConsent", reflect.TypeOf((*MockSignalsService)(nil).SetConsent), agentID, enabled)
}

// ListExports mocks base method.
func (m *MockSignalsService) ListExports(opts *do.SignalsExportListOptions) (do.SignalsExports, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListExports", opts)
	ret0, _ := ret[0].(do.SignalsExports)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListExports indicates an expected call of ListExports.
func (mr *MockSignalsServiceMockRecorder) ListExports(opts any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListExports", reflect.TypeOf((*MockSignalsService)(nil).ListExports), opts)
}

// CreateExport mocks base method.
func (m *MockSignalsService) CreateExport(req *do.SignalsCreateExportRequest) (*do.SignalsExport, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateExport", req)
	ret0, _ := ret[0].(*do.SignalsExport)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateExport indicates an expected call of CreateExport.
func (mr *MockSignalsServiceMockRecorder) CreateExport(req any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateExport", reflect.TypeOf((*MockSignalsService)(nil).CreateExport), req)
}

// GetExport mocks base method.
func (m *MockSignalsService) GetExport(exportID string) (*do.SignalsExport, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetExport", exportID)
	ret0, _ := ret[0].(*do.SignalsExport)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetExport indicates an expected call of GetExport.
func (mr *MockSignalsServiceMockRecorder) GetExport(exportID any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetExport", reflect.TypeOf((*MockSignalsService)(nil).GetExport), exportID)
}

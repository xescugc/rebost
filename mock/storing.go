// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/xescugc/rebost/storing (interfaces: Service)

// Package mock is a generated GoMock package.
package mock

import (
	context "context"
	gomock "github.com/golang/mock/gomock"
	config "github.com/xescugc/rebost/config"
	io "io"
	reflect "reflect"
)

// Storing is a mock of Service interface
type Storing struct {
	ctrl     *gomock.Controller
	recorder *StoringMockRecorder
}

// StoringMockRecorder is the mock recorder for Storing
type StoringMockRecorder struct {
	mock *Storing
}

// NewStoring creates a new mock instance
func NewStoring(ctrl *gomock.Controller) *Storing {
	mock := &Storing{ctrl: ctrl}
	mock.recorder = &StoringMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *Storing) EXPECT() *StoringMockRecorder {
	return m.recorder
}

// Config mocks base method
func (m *Storing) Config(arg0 context.Context) (*config.Config, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Config", arg0)
	ret0, _ := ret[0].(*config.Config)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Config indicates an expected call of Config
func (mr *StoringMockRecorder) Config(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Config", reflect.TypeOf((*Storing)(nil).Config), arg0)
}

// CreateFile mocks base method
func (m *Storing) CreateFile(arg0 context.Context, arg1 string, arg2 io.ReadCloser, arg3 int) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateFile", arg0, arg1, arg2, arg3)
	ret0, _ := ret[0].(error)
	return ret0
}

// CreateFile indicates an expected call of CreateFile
func (mr *StoringMockRecorder) CreateFile(arg0, arg1, arg2, arg3 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateFile", reflect.TypeOf((*Storing)(nil).CreateFile), arg0, arg1, arg2, arg3)
}

// CreateReplica mocks base method
func (m *Storing) CreateReplica(arg0 context.Context, arg1 string, arg2 io.ReadCloser) (string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateReplica", arg0, arg1, arg2)
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateReplica indicates an expected call of CreateReplica
func (mr *StoringMockRecorder) CreateReplica(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateReplica", reflect.TypeOf((*Storing)(nil).CreateReplica), arg0, arg1, arg2)
}

// DeleteFile mocks base method
func (m *Storing) DeleteFile(arg0 context.Context, arg1 string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteFile", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteFile indicates an expected call of DeleteFile
func (mr *StoringMockRecorder) DeleteFile(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteFile", reflect.TypeOf((*Storing)(nil).DeleteFile), arg0, arg1)
}

// GetFile mocks base method
func (m *Storing) GetFile(arg0 context.Context, arg1 string) (io.ReadCloser, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetFile", arg0, arg1)
	ret0, _ := ret[0].(io.ReadCloser)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetFile indicates an expected call of GetFile
func (mr *StoringMockRecorder) GetFile(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetFile", reflect.TypeOf((*Storing)(nil).GetFile), arg0, arg1)
}

// HasFile mocks base method
func (m *Storing) HasFile(arg0 context.Context, arg1 string) (bool, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "HasFile", arg0, arg1)
	ret0, _ := ret[0].(bool)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// HasFile indicates an expected call of HasFile
func (mr *StoringMockRecorder) HasFile(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "HasFile", reflect.TypeOf((*Storing)(nil).HasFile), arg0, arg1)
}

// UpdateFileReplica mocks base method
func (m *Storing) UpdateFileReplica(arg0 context.Context, arg1 string, arg2 []string, arg3 int) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateFileReplica", arg0, arg1, arg2, arg3)
	ret0, _ := ret[0].(error)
	return ret0
}

// UpdateFileReplica indicates an expected call of UpdateFileReplica
func (mr *StoringMockRecorder) UpdateFileReplica(arg0, arg1, arg2, arg3 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateFileReplica", reflect.TypeOf((*Storing)(nil).UpdateFileReplica), arg0, arg1, arg2, arg3)
}

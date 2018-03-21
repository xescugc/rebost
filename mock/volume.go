// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/xescugc/rebost/volume (interfaces: Volume)

// Package mock is a generated GoMock package.
package mock

import (
	context "context"
	gomock "github.com/golang/mock/gomock"
	file "github.com/xescugc/rebost/file"
	io "io"
	reflect "reflect"
)

// Volume is a mock of Volume interface
type Volume struct {
	ctrl     *gomock.Controller
	recorder *VolumeMockRecorder
}

// VolumeMockRecorder is the mock recorder for Volume
type VolumeMockRecorder struct {
	mock *Volume
}

// NewVolume creates a new mock instance
func NewVolume(ctrl *gomock.Controller) *Volume {
	mock := &Volume{ctrl: ctrl}
	mock.recorder = &VolumeMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *Volume) EXPECT() *VolumeMockRecorder {
	return m.recorder
}

// CreateFile mocks base method
func (m *Volume) CreateFile(arg0 context.Context, arg1 string, arg2 io.Reader) (*file.File, error) {
	ret := m.ctrl.Call(m, "CreateFile", arg0, arg1, arg2)
	ret0, _ := ret[0].(*file.File)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateFile indicates an expected call of CreateFile
func (mr *VolumeMockRecorder) CreateFile(arg0, arg1, arg2 interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateFile", reflect.TypeOf((*Volume)(nil).CreateFile), arg0, arg1, arg2)
}

// DeleteFile mocks base method
func (m *Volume) DeleteFile(arg0 context.Context, arg1 string) error {
	ret := m.ctrl.Call(m, "DeleteFile", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteFile indicates an expected call of DeleteFile
func (mr *VolumeMockRecorder) DeleteFile(arg0, arg1 interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteFile", reflect.TypeOf((*Volume)(nil).DeleteFile), arg0, arg1)
}

// GetFile mocks base method
func (m *Volume) GetFile(arg0 context.Context, arg1 string) (io.Reader, error) {
	ret := m.ctrl.Call(m, "GetFile", arg0, arg1)
	ret0, _ := ret[0].(io.Reader)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetFile indicates an expected call of GetFile
func (mr *VolumeMockRecorder) GetFile(arg0, arg1 interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetFile", reflect.TypeOf((*Volume)(nil).GetFile), arg0, arg1)
}

// HasFile mocks base method
func (m *Volume) HasFile(arg0 context.Context, arg1 string) (bool, error) {
	ret := m.ctrl.Call(m, "HasFile", arg0, arg1)
	ret0, _ := ret[0].(bool)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// HasFile indicates an expected call of HasFile
func (mr *VolumeMockRecorder) HasFile(arg0, arg1 interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "HasFile", reflect.TypeOf((*Volume)(nil).HasFile), arg0, arg1)
}

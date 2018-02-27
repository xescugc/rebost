// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/xescugc/rebost/uow (interfaces: UnitOfWork)

// Package mock is a generated GoMock package.
package mock

import (
	gomock "github.com/golang/mock/gomock"
	file "github.com/xescugc/rebost/file"
	idxkey "github.com/xescugc/rebost/idxkey"
	reflect "reflect"
)

// UnitOfWork is a mock of UnitOfWork interface
type UnitOfWork struct {
	ctrl     *gomock.Controller
	recorder *UnitOfWorkMockRecorder
}

// UnitOfWorkMockRecorder is the mock recorder for UnitOfWork
type UnitOfWorkMockRecorder struct {
	mock *UnitOfWork
}

// NewUnitOfWork creates a new mock instance
func NewUnitOfWork(ctrl *gomock.Controller) *UnitOfWork {
	mock := &UnitOfWork{ctrl: ctrl}
	mock.recorder = &UnitOfWorkMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *UnitOfWork) EXPECT() *UnitOfWorkMockRecorder {
	return m.recorder
}

// Files mocks base method
func (m *UnitOfWork) Files() file.Repository {
	ret := m.ctrl.Call(m, "Files")
	ret0, _ := ret[0].(file.Repository)
	return ret0
}

// Files indicates an expected call of Files
func (mr *UnitOfWorkMockRecorder) Files() *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Files", reflect.TypeOf((*UnitOfWork)(nil).Files))
}

// IDXKeys mocks base method
func (m *UnitOfWork) IDXKeys() idxkey.Repository {
	ret := m.ctrl.Call(m, "IDXKeys")
	ret0, _ := ret[0].(idxkey.Repository)
	return ret0
}

// IDXKeys indicates an expected call of IDXKeys
func (mr *UnitOfWorkMockRecorder) IDXKeys() *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "IDXKeys", reflect.TypeOf((*UnitOfWork)(nil).IDXKeys))
}
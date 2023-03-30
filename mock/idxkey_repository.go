// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/xescugc/rebost/idxkey (interfaces: Repository)

// Package mock is a generated GoMock package.
package mock

import (
	context "context"
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	idxkey "github.com/xescugc/rebost/idxkey"
)

// IDXKeyRepository is a mock of Repository interface.
type IDXKeyRepository struct {
	ctrl     *gomock.Controller
	recorder *IDXKeyRepositoryMockRecorder
}

// IDXKeyRepositoryMockRecorder is the mock recorder for IDXKeyRepository.
type IDXKeyRepositoryMockRecorder struct {
	mock *IDXKeyRepository
}

// NewIDXKeyRepository creates a new mock instance.
func NewIDXKeyRepository(ctrl *gomock.Controller) *IDXKeyRepository {
	mock := &IDXKeyRepository{ctrl: ctrl}
	mock.recorder = &IDXKeyRepositoryMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *IDXKeyRepository) EXPECT() *IDXKeyRepositoryMockRecorder {
	return m.recorder
}

// CreateOrReplace mocks base method.
func (m *IDXKeyRepository) CreateOrReplace(arg0 context.Context, arg1 *idxkey.IDXKey) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateOrReplace", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// CreateOrReplace indicates an expected call of CreateOrReplace.
func (mr *IDXKeyRepositoryMockRecorder) CreateOrReplace(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateOrReplace", reflect.TypeOf((*IDXKeyRepository)(nil).CreateOrReplace), arg0, arg1)
}

// DeleteAll mocks base method.
func (m *IDXKeyRepository) DeleteAll(arg0 context.Context) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteAll", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteAll indicates an expected call of DeleteAll.
func (mr *IDXKeyRepositoryMockRecorder) DeleteAll(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteAll", reflect.TypeOf((*IDXKeyRepository)(nil).DeleteAll), arg0)
}

// DeleteByKey mocks base method.
func (m *IDXKeyRepository) DeleteByKey(arg0 context.Context, arg1 string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteByKey", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteByKey indicates an expected call of DeleteByKey.
func (mr *IDXKeyRepositoryMockRecorder) DeleteByKey(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteByKey", reflect.TypeOf((*IDXKeyRepository)(nil).DeleteByKey), arg0, arg1)
}

// FindByKey mocks base method.
func (m *IDXKeyRepository) FindByKey(arg0 context.Context, arg1 string) (*idxkey.IDXKey, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "FindByKey", arg0, arg1)
	ret0, _ := ret[0].(*idxkey.IDXKey)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// FindByKey indicates an expected call of FindByKey.
func (mr *IDXKeyRepositoryMockRecorder) FindByKey(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "FindByKey", reflect.TypeOf((*IDXKeyRepository)(nil).FindByKey), arg0, arg1)
}

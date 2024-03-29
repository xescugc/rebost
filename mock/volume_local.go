// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/xescugc/rebost/volume (interfaces: Local)

// Package mock is a generated GoMock package.
package mock

import (
	context "context"
	io "io"
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	replica "github.com/xescugc/rebost/replica"
	state "github.com/xescugc/rebost/state"
)

// VolumeLocal is a mock of Local interface.
type VolumeLocal struct {
	ctrl     *gomock.Controller
	recorder *VolumeLocalMockRecorder
}

// VolumeLocalMockRecorder is the mock recorder for VolumeLocal.
type VolumeLocalMockRecorder struct {
	mock *VolumeLocal
}

// NewVolumeLocal creates a new mock instance.
func NewVolumeLocal(ctrl *gomock.Controller) *VolumeLocal {
	mock := &VolumeLocal{ctrl: ctrl}
	mock.recorder = &VolumeLocalMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *VolumeLocal) EXPECT() *VolumeLocalMockRecorder {
	return m.recorder
}

// Close mocks base method.
func (m *VolumeLocal) Close() error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Close")
	ret0, _ := ret[0].(error)
	return ret0
}

// Close indicates an expected call of Close.
func (mr *VolumeLocalMockRecorder) Close() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Close", reflect.TypeOf((*VolumeLocal)(nil).Close))
}

// CreateFile mocks base method.
func (m *VolumeLocal) CreateFile(arg0 context.Context, arg1 string, arg2 io.ReadCloser, arg3 int) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateFile", arg0, arg1, arg2, arg3)
	ret0, _ := ret[0].(error)
	return ret0
}

// CreateFile indicates an expected call of CreateFile.
func (mr *VolumeLocalMockRecorder) CreateFile(arg0, arg1, arg2, arg3 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateFile", reflect.TypeOf((*VolumeLocal)(nil).CreateFile), arg0, arg1, arg2, arg3)
}

// DeleteFile mocks base method.
func (m *VolumeLocal) DeleteFile(arg0 context.Context, arg1 string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteFile", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteFile indicates an expected call of DeleteFile.
func (mr *VolumeLocalMockRecorder) DeleteFile(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteFile", reflect.TypeOf((*VolumeLocal)(nil).DeleteFile), arg0, arg1)
}

// GetFile mocks base method.
func (m *VolumeLocal) GetFile(arg0 context.Context, arg1 string) (io.ReadCloser, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetFile", arg0, arg1)
	ret0, _ := ret[0].(io.ReadCloser)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetFile indicates an expected call of GetFile.
func (mr *VolumeLocalMockRecorder) GetFile(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetFile", reflect.TypeOf((*VolumeLocal)(nil).GetFile), arg0, arg1)
}

// GetState mocks base method.
func (m *VolumeLocal) GetState(arg0 context.Context) (*state.State, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetState", arg0)
	ret0, _ := ret[0].(*state.State)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetState indicates an expected call of GetState.
func (mr *VolumeLocalMockRecorder) GetState(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetState", reflect.TypeOf((*VolumeLocal)(nil).GetState), arg0)
}

// HasFile mocks base method.
func (m *VolumeLocal) HasFile(arg0 context.Context, arg1 string) (string, bool, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "HasFile", arg0, arg1)
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(bool)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

// HasFile indicates an expected call of HasFile.
func (mr *VolumeLocalMockRecorder) HasFile(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "HasFile", reflect.TypeOf((*VolumeLocal)(nil).HasFile), arg0, arg1)
}

// ID mocks base method.
func (m *VolumeLocal) ID() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ID")
	ret0, _ := ret[0].(string)
	return ret0
}

// ID indicates an expected call of ID.
func (mr *VolumeLocalMockRecorder) ID() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ID", reflect.TypeOf((*VolumeLocal)(nil).ID))
}

// NextReplica mocks base method.
func (m *VolumeLocal) NextReplica(arg0 context.Context) (*replica.Replica, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "NextReplica", arg0)
	ret0, _ := ret[0].(*replica.Replica)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// NextReplica indicates an expected call of NextReplica.
func (mr *VolumeLocalMockRecorder) NextReplica(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "NextReplica", reflect.TypeOf((*VolumeLocal)(nil).NextReplica), arg0)
}

// Reset mocks base method.
func (m *VolumeLocal) Reset(arg0 context.Context) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Reset", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// Reset indicates an expected call of Reset.
func (mr *VolumeLocalMockRecorder) Reset(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Reset", reflect.TypeOf((*VolumeLocal)(nil).Reset), arg0)
}

// SynchronizeReplicas mocks base method.
func (m *VolumeLocal) SynchronizeReplicas(arg0 context.Context, arg1 string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SynchronizeReplicas", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// SynchronizeReplicas indicates an expected call of SynchronizeReplicas.
func (mr *VolumeLocalMockRecorder) SynchronizeReplicas(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SynchronizeReplicas", reflect.TypeOf((*VolumeLocal)(nil).SynchronizeReplicas), arg0, arg1)
}

// UpdateFileReplica mocks base method.
func (m *VolumeLocal) UpdateFileReplica(arg0 context.Context, arg1 string, arg2 []string, arg3 int) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateFileReplica", arg0, arg1, arg2, arg3)
	ret0, _ := ret[0].(error)
	return ret0
}

// UpdateFileReplica indicates an expected call of UpdateFileReplica.
func (mr *VolumeLocalMockRecorder) UpdateFileReplica(arg0, arg1, arg2, arg3 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateFileReplica", reflect.TypeOf((*VolumeLocal)(nil).UpdateFileReplica), arg0, arg1, arg2, arg3)
}

// UpdateReplica mocks base method.
func (m *VolumeLocal) UpdateReplica(arg0 context.Context, arg1 *replica.Replica, arg2 string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateReplica", arg0, arg1, arg2)
	ret0, _ := ret[0].(error)
	return ret0
}

// UpdateReplica indicates an expected call of UpdateReplica.
func (mr *VolumeLocalMockRecorder) UpdateReplica(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateReplica", reflect.TypeOf((*VolumeLocal)(nil).UpdateReplica), arg0, arg1, arg2)
}
